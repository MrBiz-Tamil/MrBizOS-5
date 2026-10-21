//go:build ignore

package emr

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/models"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// Router /emr/medical-records [post]
func CreateMedicalRecord(c *fiber.Ctx) error {
	type Request struct {
		PatientID uint   `json:"patient_id"`
		DoctorID  uint   `json:"doctor_id"`
		Diagnosis string `json:"diagnosis"`
		Notes     string `json:"notes"`
		File      string `json:"file_url"`
		FileType  string `json:"file_type"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	record := models.MedicalRecord{
		PatientID: req.PatientID,
		DoctorID:  req.DoctorID,
		Diagnosis: req.Diagnosis,
		Notes:     req.Notes,
		FileURL:   req.File,
		FileType:  req.FileType,
	}

	if err := database.DB.Create(&record).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_RECORD_FAILED", "Failed to create record")
	}

	return response.JSON(c, fiber.StatusCreated, record, "Created")
}

// Router /emr/medical-records [get]
func GetMedicalRecords(c *fiber.Ctx) error {
	patientID := c.Query("patient_id")
	doctorID := c.Query("doctor_id")

	db := database.DB
	var records []models.MedicalRecord

	query := db.Model(&models.MedicalRecord{})

	if patientID != "" {
		query = query.Where("patient_id = ?", patientID)
	}
	if doctorID != "" {
		query = query.Where("doctor_id = ?", doctorID)
	}

	query.Order("created_at desc").Find(&records)

	return response.JSON(c, fiber.StatusOK, records, "Fetched")
}

// GetRecordDetails retrieves a single record
func GetRecordDetails(c *fiber.Ctx) error {
	id := c.Params("id")
	var record models.MedicalRecord
	if err := database.DB.First(&record, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "RECORD_NOT_FOUND", "Record not found")
	}
	return response.JSON(c, fiber.StatusOK, record, "Fetched")
}

// UpdateMedicalRecord updates an existing record
func UpdateMedicalRecord(c *fiber.Ctx) error {
	id := c.Params("id")
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	if err := database.DB.Model(&models.MedicalRecord{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Update failed")
	}

	return response.JSON(c, fiber.StatusOK, nil, "Record updated")
}

// DeleteMedicalRecord archives/deletes a record
func DeleteMedicalRecord(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := database.DB.Delete(&models.MedicalRecord{}, id).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Delete failed")
	}
	return response.JSON(c, fiber.StatusOK, nil, "Record deleted")
}

// GetPatientTimeline retrieves a chronological health timeline
func GetPatientTimeline(c *fiber.Ctx) error {
	patientID := c.Params("id")

	type TimelineItem struct {
		Type    string      `json:"type"`
		Date    interface{} `json:"date"`
		Title   string      `json:"title"`
		Detail  string      `json:"detail"`
		RawData interface{} `json:"data"`
	}

	var timeline []TimelineItem

	// 1. Fetch Appointments
	var appointments []appointment.Appointment
	database.DB.Where("patient_id = ?", patientID).Order("created_at desc").Find(&appointments)
	for _, a := range appointments {
		timeline = append(timeline, TimelineItem{
			Type:    "appointment",
			Date:    a.CreatedAt,
			Title:   "Consultation",
			Detail:  string(a.Status),
			RawData: a,
		})
	}

	// 2. Fetch Medical Records
	var records []models.MedicalRecord
	database.DB.Where("patient_id = ?", patientID).Order("created_at desc").Find(&records)
	for _, r := range records {
		timeline = append(timeline, TimelineItem{
			Type:    "medical_record",
			Date:    r.CreatedAt,
			Title:   "Medical Note",
			Detail:  r.Diagnosis,
			RawData: r,
		})
	}

	// 3. Fetch Prescriptions
	var prescriptions []models.Prescription
	database.DB.Where("patient_id = ?", patientID).Order("created_at desc").Find(&prescriptions)
	for _, p := range prescriptions {
		timeline = append(timeline, TimelineItem{
			Type:    "prescription",
			Date:    p.CreatedAt,
			Title:   "Prescription Issued",
			Detail:  p.Diagnosis,
			RawData: p,
		})
	}

	return response.JSON(c, fiber.StatusOK, timeline, "Fetched")
}

// GetMyMedicalRecords retrieves records for the authenticated patient
func GetMyMedicalRecords(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
	}

	var records []models.MedicalRecord
	if err := database.DB.Where("patient_id = ?", userID).Order("created_at desc").Find(&records).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_RECORDS_FAILED", "Failed to fetch records")
	}

	return response.JSON(c, fiber.StatusOK, records, "Fetched")
}
