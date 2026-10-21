package hospital

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/pharmacy"
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

type Service interface {
	GetDashboardStats(ctx context.Context, hospitalID uint) (DashboardStats, error)
	GetOPDQueue(ctx context.Context, hospitalID uint) ([]appointment.Appointment, error)
	AddWalkInToQueue(ctx context.Context, hospitalID uint, patientID uint, doctorID uint, notes string) (*appointment.Appointment, error)
	CallNextPatient(ctx context.Context, hospitalID uint) (*appointment.Appointment, error)
	CompleteOPDConsultation(ctx context.Context, opdID uint) error
	AdmitPatient(ctx context.Context, hospitalID uint, patientID uint, doctorID uint, bedID uint) (*IPDAdmission, error)
	DischargePatient(ctx context.Context, admissionID uint, summary string) error
	GetIPDAdmissions(ctx context.Context, hospitalID uint) ([]IPDAdmission, error)
	GetAvailableBeds(ctx context.Context, hospitalID uint) ([]Bed, error)
	GetHospitalInventory(ctx context.Context, hospitalID uint) ([]pharmacy.PharmacyInventory, error)
	UpdateHospitalInventory(ctx context.Context, itemID uint, quantityChange int, reason string) error
	CreateWardServiceRequest(ctx context.Context, hospitalID uint, patientID *uint, requestedByID uint, bedID *uint, itemsJSON []byte, priority string, notes string) (*WardServiceRequest, error)
	GetWardServiceRequests(ctx context.Context, hospitalID uint) ([]WardServiceRequest, error)
	UpdateWardServiceRequest(ctx context.Context, requestID uint, status string) error
	ListAllHospitals(ctx context.Context, hospitalType string) ([]Hospital, error)
	CreateHospital(ctx context.Context, name, address, contact, hType, hospitalType string, franchiseID *uint) (*Hospital, error)
	UpdateHospital(ctx context.Context, id uint, name, address, contact, hType, hospitalType string, isActive bool) (*Hospital, error)
	ToggleHospital(ctx context.Context, id uint) (*Hospital, error)

	// Wards & beds
	ListWards(ctx context.Context, hospitalID uint) ([]Ward, error)
	CreateWard(ctx context.Context, w *Ward) error
	ListBedsByWard(ctx context.Context, wardID uint) ([]Bed, error)
	ListBedsByHospital(ctx context.Context, hospitalID uint) ([]Bed, error)

	// Tasks
	ListTasks(ctx context.Context, hospitalID uint, status string) ([]NursingTask, error)
	CreateTask(ctx context.Context, t *NursingTask) error
	CompleteTask(ctx context.Context, taskID uint, notes string) error

	// Shifts
	ListShifts(ctx context.Context, hospitalID uint) ([]ShiftSchedule, error)
	CreateShift(ctx context.Context, s *ShiftSchedule) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetDashboardStats(ctx context.Context, hospitalID uint) (DashboardStats, error) {
	totalBeds, _ := s.repo.CountBeds(ctx, hospitalID)
	occupiedBeds, _ := s.repo.CountOccupiedBeds(ctx, hospitalID)
	totalWards, _ := s.repo.CountWards(ctx, hospitalID)

	today := time.Now().Truncate(24 * time.Hour)
	opdToday, _ := s.repo.CountOPDByDate(ctx, hospitalID, today)
	opdWaiting, _ := s.repo.CountOPDByStatus(ctx, hospitalID, "waiting")
	ipdCurrent, _ := s.repo.CountIPDByStatus(ctx, hospitalID, "admitted")

	return DashboardStats{
		TotalWards:    int(totalWards),
		TotalBeds:     int(totalBeds),
		AvailableBeds: int(totalBeds - occupiedBeds),
		OccupiedBeds:  int(occupiedBeds),
		OPDToday:      int(opdToday),
		OPDWaiting:    int(opdWaiting),
		IPDCurrent:    int(ipdCurrent),
	}, nil
}

// ── New service methods ────────────────────────────────────────────────────────

func (s *service) CallNextPatient(ctx context.Context, hospitalID uint) (*appointment.Appointment, error) {
	next, err := s.repo.GetNextWaitingOPD(ctx, hospitalID)
	if err != nil {
		return nil, fmt.Errorf("no patients waiting in OPD queue")
	}
	if err := s.repo.UpdateOPDStatus(ctx, next.ID, "in_consultation"); err != nil {
		return nil, err
	}
	next.Status = "in_consultation"
	return next, nil
}

func (s *service) CompleteOPDConsultation(ctx context.Context, opdID uint) error {
	opd, err := s.repo.GetOPDByID(ctx, opdID)
	if err != nil {
		return fmt.Errorf("OPD entry not found")
	}
	status := string(opd.Status)
	if status != "in_consultation" && status != "confirmed" {
		return fmt.Errorf("cannot complete OPD with status '%s' — must be in_consultation", status)
	}
	return s.repo.UpdateOPDStatus(ctx, opdID, "completed")
}

func (s *service) ListWards(ctx context.Context, hospitalID uint) ([]Ward, error) {
	return s.repo.ListWards(ctx, hospitalID)
}

func (s *service) CreateWard(ctx context.Context, w *Ward) error {
	if w.Name == "" {
		return fmt.Errorf("ward name is required")
	}
	if w.HospitalID == 0 {
		return fmt.Errorf("hospital_id is required")
	}
	if w.WardType == "" {
		w.WardType = "general"
	}
	return s.repo.CreateWard(ctx, w)
}

func (s *service) ListBedsByWard(ctx context.Context, wardID uint) ([]Bed, error) {
	return s.repo.ListBedsByWard(ctx, wardID)
}

func (s *service) ListBedsByHospital(ctx context.Context, hospitalID uint) ([]Bed, error) {
	return s.repo.ListBedsByHospital(ctx, hospitalID)
}

func (s *service) ListTasks(ctx context.Context, hospitalID uint, status string) ([]NursingTask, error) {
	return s.repo.ListTasks(ctx, hospitalID, status)
}

func (s *service) CreateTask(ctx context.Context, t *NursingTask) error {
	if t.Title == "" {
		return fmt.Errorf("task title is required")
	}
	if t.HospitalID == 0 {
		t.HospitalID = 1
	}
	if t.Priority == "" {
		t.Priority = "medium"
	}
	if t.Status == "" {
		t.Status = "pending"
	}
	return s.repo.CreateTask(ctx, t)
}

func (s *service) CompleteTask(ctx context.Context, taskID uint, notes string) error {
	task, err := s.repo.GetTaskByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found")
	}
	if task.Status == "completed" {
		return fmt.Errorf("task already completed")
	}
	if task.Status == "cancelled" {
		return fmt.Errorf("cannot complete a cancelled task")
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":       "completed",
		"completed_at": &now,
	}
	if notes != "" {
		updates["notes"] = notes
	}
	return s.repo.UpdateTask(ctx, taskID, updates)
}

func (s *service) ListShifts(ctx context.Context, hospitalID uint) ([]ShiftSchedule, error) {
	return s.repo.ListShifts(ctx, hospitalID, nil)
}

func (s *service) CreateShift(ctx context.Context, sch *ShiftSchedule) error {
	if sch.StaffID == 0 {
		return fmt.Errorf("staff_id is required")
	}
	if sch.HospitalID == 0 {
		sch.HospitalID = 1
	}
	return s.repo.CreateShift(ctx, sch)
}

func (s *service) GetOPDQueue(ctx context.Context, hospitalID uint) ([]appointment.Appointment, error) {
	return s.repo.GetOPDQueue(ctx, hospitalID)
}

func (s *service) AddWalkInToQueue(ctx context.Context, hospitalID uint, patientID uint, doctorID uint, notes string) (*appointment.Appointment, error) {
	appt := &appointment.Appointment{
		PatientID:        patientID,
		DoctorID:         &doctorID,
		StartTime:        time.Now(),
		EndTime:          time.Now().Add(15 * time.Minute), // Estimated 15-minute slot
		Status:           appointment.StatusConfirmed,      // Immediately confirmed
		AppointmentType:  "walk-in",
		ConsultationMode: "in-person",
		Notes:            notes,
	}

	if err := s.repo.CreateAppointment(ctx, appt); err != nil {
		return nil, err
	}

	return appt, nil
}

func (s *service) AdmitPatient(ctx context.Context, hospitalID uint, patientID uint, doctorID uint, bedID uint) (*IPDAdmission, error) {
	var admission *IPDAdmission
	err := s.repo.ExecuteTransaction(ctx, func(txRepo Repository) error {
		bed, err := txRepo.GetBedByID(ctx, bedID)
		if err != nil {
			return err
		}

		if bed.Status != "available" {
			return fmt.Errorf("bed %s is currently %s", bed.BedNumber, bed.Status)
		}

		newAdmission := IPDAdmission{
			HospitalID:    hospitalID,
			PatientID:     patientID,
			DoctorID:      doctorID,
			BedID:         bedID,
			AdmissionDate: time.Now(),
			Status:        "admitted",
		}

		if err := txRepo.CreateAdmission(ctx, &newAdmission); err != nil {
			return err
		}

		if err := txRepo.UpdateBedStatus(ctx, bedID, "occupied"); err != nil {
			return err
		}

		admission = &newAdmission
		return nil
	})

	return admission, err
}

func (s *service) DischargePatient(ctx context.Context, admissionID uint, summary string) error {
	return s.repo.ExecuteTransaction(ctx, func(txRepo Repository) error {
		admission, err := txRepo.GetAdmissionByID(ctx, admissionID)
		if err != nil {
			return err
		}

		if admission.Status == "discharged" {
			return fmt.Errorf("patient already discharged")
		}

		now := time.Now()
		admission.DischargeDate = &now
		admission.Status = "discharged"
		admission.DischargeSummary = summary

		if err := txRepo.UpdateAdmission(ctx, admission); err != nil {
			return err
		}

		return txRepo.UpdateBedStatus(ctx, admission.BedID, "available")
	})
}

func (s *service) GetIPDAdmissions(ctx context.Context, hospitalID uint) ([]IPDAdmission, error) {
	return s.repo.GetIPDAdmissions(ctx, hospitalID)
}

func (s *service) GetAvailableBeds(ctx context.Context, hospitalID uint) ([]Bed, error) {
	return s.repo.GetAvailableBeds(ctx, hospitalID)
}

func (s *service) GetHospitalInventory(ctx context.Context, hospitalID uint) ([]pharmacy.PharmacyInventory, error) {
	return s.repo.GetHospitalInventory(ctx, hospitalID)
}

func (s *service) UpdateHospitalInventory(ctx context.Context, itemID uint, quantityChange int, reason string) error {
	item, err := s.repo.GetInventoryItem(ctx, itemID)
	if err != nil {
		return err
	}

	item.StockQuantity += quantityChange
	if item.StockQuantity < 0 {
		item.StockQuantity = 0
	}

	return s.repo.UpdateInventoryItem(ctx, item)
}

func (s *service) CreateWardServiceRequest(ctx context.Context, hospitalID uint, patientID *uint, requestedByID uint, bedID *uint, itemsJSON []byte, priority string, notes string) (*WardServiceRequest, error) {
	type ItemReq struct {
		MedicineID uint `json:"medicine_id"`
		Qty        int  `json:"qty"`
	}
	var items []ItemReq
	if err := json.Unmarshal(itemsJSON, &items); err != nil {
		return nil, fmt.Errorf("invalid items JSON: %v", err)
	}

	var req *WardServiceRequest

	err := s.repo.ExecuteTransaction(ctx, func(txRepo Repository) error {
		for _, item := range items {
			inv, err := txRepo.GetInventoryItemByMedicine(ctx, item.MedicineID)
			if err != nil {
				return fmt.Errorf("insufficient stock for medicine ID %d", item.MedicineID)
			}
			if inv.StockQuantity < item.Qty {
				return fmt.Errorf("insufficient stock for medicine ID %d", item.MedicineID)
			}

			inv.StockQuantity -= item.Qty
			if err := txRepo.UpdateInventoryItem(ctx, inv); err != nil {
				return err
			}
		}

		newReq := WardServiceRequest{
			HospitalID:    hospitalID,
			PatientID:     patientID,
			RequestedByID: requestedByID,
			BedID:         bedID,
			Items:         itemsJSON,
			Status:        "pending",
			Priority:      priority,
			Notes:         notes,
		}

		if err := txRepo.CreateWardServiceRequest(ctx, &newReq); err != nil {
			return err
		}
		req = &newReq
		return nil
	})

	return req, err
}

func (s *service) GetWardServiceRequests(ctx context.Context, hospitalID uint) ([]WardServiceRequest, error) {
	return s.repo.GetWardServiceRequests(ctx, hospitalID)
}

func (s *service) UpdateWardServiceRequest(ctx context.Context, requestID uint, status string) error {
	return s.repo.UpdateWardServiceRequestStatus(ctx, requestID, status)
}

func (s *service) ListAllHospitals(ctx context.Context, hospitalType string) ([]Hospital, error) {
	return s.repo.ListAllHospitals(ctx, hospitalType)
}

func (s *service) CreateHospital(ctx context.Context, name, address, contact, hType, hospitalType string, franchiseID *uint) (*Hospital, error) {
	h := &Hospital{
		Name:         name,
		Address:      address,
		ContactInfo:  contact,
		Type:         hType,
		HospitalType: hospitalType,
		FranchiseID:  franchiseID,
		IsActive:     true,
	}
	err := s.repo.CreateHospital(ctx, h)
	return h, err
}

func (s *service) UpdateHospital(ctx context.Context, id uint, name, address, contact, hType, hospitalType string, isActive bool) (*Hospital, error) {
	h, err := s.repo.GetHospitalByID(ctx, id)
	if err != nil {
		return nil, err
	}
	h.Name = name
	h.Address = address
	h.ContactInfo = contact
	h.Type = hType
	h.HospitalType = hospitalType
	h.IsActive = isActive

	err = s.repo.UpdateHospital(ctx, h)
	return h, err
}

func (s *service) ToggleHospital(ctx context.Context, id uint) (*Hospital, error) {
	h, err := s.repo.GetHospitalByID(ctx, id)
	if err != nil {
		return nil, err
	}
	h.IsActive = !h.IsActive
	err = s.repo.UpdateHospital(ctx, h)
	return h, err
}
