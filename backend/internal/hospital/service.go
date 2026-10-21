//go:build ignore
// +build ignore

package hospital

import (
	"encoding/json"
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/models"
	"time"
)

type DashboardStats struct {
	TotalWards    int `json:"total_wards"`
	TotalBeds     int `json:"total_beds"`
	AvailableBeds int `json:"available_beds"`
	OccupiedBeds  int `json:"occupied_beds"`
	OPDToday      int `json:"opd_today"`
	OPDWaiting    int `json:"opd_waiting"`
	IPDCurrent    int `json:"ipd_current"`
}

func GetDashboardStats(hospitalID uint) (DashboardStats, error) {
	var totalBeds, occupiedBeds int64
	database.DB.Model(&models.Bed{}).Count(&totalBeds)
	database.DB.Model(&models.Bed{}).Where("status = ?", "occupied").Count(&occupiedBeds)

	return DashboardStats{
		TotalWards:    5, // Mock
		TotalBeds:     int(totalBeds),
		AvailableBeds: int(totalBeds - occupiedBeds),
		OccupiedBeds:  int(occupiedBeds),
		OPDToday:      24, // Mock
		OPDWaiting:    12, // Mock
		IPDCurrent:    int(occupiedBeds),
	}, nil
}

func GetOPDQueue(hospitalID uint) ([]appointment.Appointment, error) {
	var appts []appointment.Appointment
	err := database.DB.Preload("Patient").Preload("Doctor").
		Where("status = ?", "confirmed").
		Limit(20).Find(&appts).Error
	return appts, err
}

func AddWalkInToQueue(hospitalID uint, patientID uint, doctorID uint, notes string) (*appointment.Appointment, error) {
	appointment := appointment.Appointment{
		PatientID:        patientID,
		DoctorID:         &doctorID,
		StartTime:        time.Now(),
		EndTime:          time.Now().Add(15 * time.Minute), // Estimated 15-minute slot
		Status:           appointment.StatusConfirmed,      // Immediately confirmed
		AppointmentType:  "walk-in",
		ConsultationMode: "in-person",
		Notes:            notes,
	}

	if err := database.DB.Create(&appointment).Error; err != nil {
		return nil, err
	}

	return &appointment, nil
}

func AdmitPatient(hospitalID uint, patientID uint, doctorID uint, bedID uint) (*models.IPDAdmission, error) {
	tx := database.DB.Begin()

	// 1. Check Bed Availability
	var bed models.Bed
	if err := tx.First(&bed, bedID).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if bed.Status != "available" {
		tx.Rollback()
		return nil, fmt.Errorf("bed %s is currently %s", bed.BedNumber, bed.Status)
	}

	// 2. Create Admission
	admission := models.IPDAdmission{
		HospitalID:    hospitalID,
		PatientID:     patientID,
		DoctorID:      doctorID,
		BedID:         bedID,
		AdmissionDate: time.Now(),
		Status:        "admitted",
	}

	if err := tx.Create(&admission).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 3. Update Bed Status
	if err := tx.Model(&bed).Update("status", "occupied").Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	return &admission, tx.Commit().Error
}

func DischargePatient(admissionID uint, summary string) error {
	tx := database.DB.Begin()

	var admission models.IPDAdmission
	if err := tx.First(&admission, admissionID).Error; err != nil {
		tx.Rollback()
		return err
	}

	if admission.Status == "discharged" {
		tx.Rollback()
		return fmt.Errorf("patient already discharged")
	}

	// 1. Update Admission
	now := time.Now()
	admission.DischargeDate = &now
	admission.Status = "discharged"
	admission.DischargeSummary = summary

	if err := tx.Save(&admission).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 2. Free the Bed
	if err := tx.Model(&models.Bed{}).Where("id = ?", admission.BedID).Update("status", "available").Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func GetIPDAdmissions(hospitalID uint) ([]models.IPDAdmission, error) {
	var admissions []models.IPDAdmission
	err := database.DB.Preload("Patient").Preload("Doctor").Preload("Bed").
		Where("hospital_id = ? AND status != ?", hospitalID, "discharged").
		Order("admission_date desc").Find(&admissions).Error
	return admissions, err
}

func GetAvailableBeds(hospitalID uint) ([]models.Bed, error) {
	var beds []models.Bed
	err := database.DB.Where("hospital_id = ? AND status = ?", hospitalID, "available").Find(&beds).Error
	return beds, err
}

func GetHospitalInventory(hospitalID uint) ([]pharmacy.PharmacyInventory, error) {
	var inventory []pharmacy.PharmacyInventory
	err := database.DB.Preload("Medicine").
		Limit(50).Find(&inventory).Error
	return inventory, err
}

func UpdateHospitalInventory(itemID uint, quantityChange int, reason string) error {
	// Simple update for now, transaction ideally
	var item pharmacy.PharmacyInventory
	if err := database.DB.First(&item, itemID).Error; err != nil {
		return err
	}

	item.StockQuantity += quantityChange
	if item.StockQuantity < 0 {
		item.StockQuantity = 0 // Prevent negative
	}

	return database.DB.Save(&item).Error
}

func CreateWardServiceRequest(hospitalID uint, patientID *uint, requestedByID uint, bedID *uint, itemsJSON []byte, priority string, notes string) (*models.WardServiceRequest, error) {
	// 1. Parse Items
	type ItemReq struct {
		MedicineID uint `json:"medicine_id"`
		Qty        int  `json:"qty"`
	}
	var items []ItemReq
	if err := json.Unmarshal(itemsJSON, &items); err != nil {
		return nil, fmt.Errorf("invalid items JSON: %v", err)
	}

	tx := database.DB.Begin()

	// 2. Validate Inventory
	for _, item := range items {
		var inv pharmacy.PharmacyInventory
		// Look for stock in any pharmacy associated with this hospital (simplification: hospital has one pharmacy or we check all)
		err := tx.Where("medicine_id = ? AND stock_quantity >= ?", item.MedicineID, item.Qty).First(&inv).Error
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("insufficient stock for medicine ID %d", item.MedicineID)
		}

		// 3. Deduct Stock (or mark as reserved) - for now we deduct to ensure availability
		inv.StockQuantity -= item.Qty
		if err := tx.Save(&inv).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// 4. Create Request
	req := models.WardServiceRequest{
		HospitalID:    hospitalID,
		PatientID:     patientID,
		RequestedByID: requestedByID,
		BedID:         bedID,
		Items:         itemsJSON,
		Status:        "pending",
		Priority:      priority,
		Notes:         notes,
	}

	if err := tx.Create(&req).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	return &req, tx.Commit().Error
}

func GetWardServiceRequests(hospitalID uint) ([]models.WardServiceRequest, error) {
	var requests []models.WardServiceRequest
	err := database.DB.Where("hospital_id = ?", hospitalID).Order("created_at desc").Find(&requests).Error
	return requests, err
}

func UpdateWardServiceRequest(requestID uint, status string) error {
	return database.DB.Model(&models.WardServiceRequest{}).Where("id = ?", requestID).Update("status", status).Error
}

// Admin CRUD functions

func ListAllHospitals(hospitalType string) ([]models.Hospital, error) {
	var hospitals []models.Hospital
	query := database.DB.Preload("Franchise")
	if hospitalType != "" && hospitalType != "all" {
		query = query.Where("hospital_type = ?", hospitalType)
	}
	err := query.Find(&hospitals).Error
	return hospitals, err
}

func CreateHospital(name, address, contact, hType, hospitalType string, franchiseID *uint) (*models.Hospital, error) {
	h := models.Hospital{
		Name:         name,
		Address:      address,
		ContactInfo:  contact,
		Type:         hType,
		HospitalType: hospitalType,
		FranchiseID:  franchiseID,
		IsActive:     true,
	}
	if err := database.DB.Create(&h).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

func UpdateHospital(id uint, name, address, contact, hType, hospitalType string, isActive bool) (*models.Hospital, error) {
	var h models.Hospital
	if err := database.DB.First(&h, id).Error; err != nil {
		return nil, err
	}
	h.Name = name
	h.Address = address
	h.ContactInfo = contact
	h.Type = hType
	h.HospitalType = hospitalType
	h.IsActive = isActive
	if err := database.DB.Save(&h).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

func ToggleHospital(id uint) (*models.Hospital, error) {
	var h models.Hospital
	if err := database.DB.First(&h, id).Error; err != nil {
		return nil, err
	}
	h.IsActive = !h.IsActive
	if err := database.DB.Save(&h).Error; err != nil {
		return nil, err
	}
	return &h, nil
}
