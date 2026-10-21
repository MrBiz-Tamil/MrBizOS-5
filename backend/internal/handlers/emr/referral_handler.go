package emr

import (
	"drayya-core/internal/database"
	emrDomain "drayya-core/internal/domain/emr"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CreateReferral creates a new clinical referral
// POST /api/v1/emr/referrals
func CreateReferral(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	var req struct {
		PatientID        uint   `json:"patient_id"`
		ReferredDoctorID *uint  `json:"referred_doctor_id"`
		Specialty        string `json:"specialty"`
		Urgency          string `json:"urgency"`
		Reason           string `json:"reason"`
		ClinicalSummary  string `json:"clinical_summary"`
		Notes            string `json:"notes"`
		DiagnosisCodes   string `json:"diagnosis_codes"`
		ExpiresInDays    int    `json:"expires_in_days"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if req.PatientID == 0 || req.Specialty == "" || req.Reason == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "patient_id, specialty and reason are required")
	}

	urgency := req.Urgency
	if urgency == "" {
		urgency = "routine"
	}

	referral := &emrDomain.ClinicalReferral{
		PatientID:         req.PatientID,
		ReferringDoctorID: userID,
		ReferredDoctorID:  req.ReferredDoctorID,
		Specialty:         req.Specialty,
		Urgency:           urgency,
		Reason:            req.Reason,
		ClinicalSummary:   req.ClinicalSummary,
		Notes:             req.Notes,
		DiagnosisCodes:    req.DiagnosisCodes,
		Status:            "pending",
	}

	if req.ExpiresInDays > 0 {
		expiry := time.Now().AddDate(0, 0, req.ExpiresInDays)
		referral.ExpiresAt = &expiry
	}

	if err := database.DB.Create(referral).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, referral, "Referral created successfully")
}

// GetSentReferrals returns referrals made by the current doctor
// GET /api/v1/emr/referrals/sent
func GetSentReferrals(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	status := c.Query("status")

	query := database.DB.
		Preload("Patient").
		Preload("ReferredDoctor").
		Where("referring_doctor_id = ?", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var referrals []emrDomain.ClinicalReferral
	if err := query.Order("created_at DESC").Find(&referrals).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, referrals, "Referrals fetched")
}

// GetReceivedReferrals returns referrals addressed to the current doctor
// GET /api/v1/emr/referrals/received
func GetReceivedReferrals(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	status := c.Query("status")

	query := database.DB.
		Preload("Patient").
		Preload("ReferringDoctor").
		Where("referred_doctor_id = ?", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var referrals []emrDomain.ClinicalReferral
	if err := query.Order("created_at DESC").Find(&referrals).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, referrals, "Referrals fetched")
}

// GetPatientReferrals returns all referrals for a specific patient
// GET /api/v1/emr/referrals/patient/:patient_id
func GetPatientReferrals(c *fiber.Ctx) error {
	patientID, err := strconv.Atoi(c.Params("patient_id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid patient ID")
	}

	var referrals []emrDomain.ClinicalReferral
	if err := database.DB.
		Preload("ReferringDoctor").
		Preload("ReferredDoctor").
		Where("patient_id = ?", uint(patientID)).
		Order("created_at DESC").
		Find(&referrals).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, referrals, "Patient referrals fetched")
}

// UpdateReferralStatus allows updating referral status
// PATCH /api/v1/emr/referrals/:id/status
func UpdateReferralStatus(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	refID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid referral ID")
	}

	var req struct {
		Status        string `json:"status"`
		AppointmentID *uint  `json:"appointment_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	var referral emrDomain.ClinicalReferral
	if err := database.DB.First(&referral, uint(refID)).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Referral not found")
	}

	// Only the referred doctor can accept/decline
	if req.Status == "accepted" || req.Status == "declined" {
		if referral.ReferredDoctorID == nil || *referral.ReferredDoctorID != userID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Only the referred doctor can accept or decline")
		}
	}

	updates := fiber.Map{"status": req.Status}
	if req.AppointmentID != nil {
		updates["appointment_id"] = req.AppointmentID
	}

	if err := database.DB.Model(&referral).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, referral, "Referral status updated")
}

// AddReferralNote adds a clinical note to a referral
// POST /api/v1/emr/referrals/:id/notes
func AddReferralNote(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	refID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid referral ID")
	}

	var req struct {
		Note    string `json:"note"`
		Outcome string `json:"outcome"`
	}
	if err := c.BodyParser(&req); err != nil || req.Note == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Note text required")
	}

	note := &emrDomain.ReferralNote{
		ReferralID: uint(refID),
		DoctorID:   userID,
		Note:       req.Note,
		Outcome:    req.Outcome,
	}

	if err := database.DB.Create(note).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, note, "Note added")
}

// RegisterReferralRoutes registers all clinical referral routes
func RegisterReferralRoutes(router fiber.Router) {
	ref := router.Group("/emr/referrals", middleware.Protected())
	ref.Post("/", CreateReferral)
	ref.Get("/sent", GetSentReferrals)
	ref.Get("/received", GetReceivedReferrals)
	ref.Get("/patient/:patient_id", GetPatientReferrals)
	ref.Patch("/:id/status", UpdateReferralStatus)
	ref.Post("/:id/notes", AddReferralNote)
}
