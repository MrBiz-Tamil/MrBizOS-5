package emr

import (
	"context"
	"fmt"
	"strings"

	"drayya-core/internal/domain/document"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/services/email"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/utils/response"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handler struct {
	svc Service
	db  *gorm.DB
}

func NewHandler(svc Service, db *gorm.DB) *Handler {
	return &Handler{svc: svc, db: db}
}

func isEMRAdmin(role string) bool {
	return role == "admin" || role == "super_admin"
}

func isEMRDoctor(role string) bool {
	return role == "doctor" || role == "chief_doctor" || role == "hospital_head_doctor"
}

func (h *Handler) doctorHasActiveAccessToPatient(doctorID, patientID uint) bool {
	if h.db == nil {
		return true // Safe fallback for unit testing where DB is not injected
	}
	var exists bool
	err := h.db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM emr_access_grants 
			WHERE doctor_id = ? AND patient_id = ? AND expires_at > ?
		)`, doctorID, patientID, time.Now()).Scan(&exists).Error
	return err == nil && exists
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Standard EMR Group (/emr/...)
	emrGroup := router.Group("/emr", middleware.Protected())
	h.registerEMRRoutes(emrGroup)

	// Compatibility Aliases (No /emr prefix)
	// Instead of router.Group("/", middleware.Protected()) which bleeds middleware onto all subsequent routes in v1,
	// we register the specific prefixes as separate groups.
	prescGroup := router.Group("/prescriptions", middleware.Protected())
	// Only doctors may issue prescriptions (admins bypass RequireRoles). Without
	// this, any authenticated user could self-issue a prescription as themselves.
	prescGroup.Post("/", middleware.RequireRoles("doctor"), h.CreatePrescription)
	prescGroup.Get("/", h.GetPrescriptions)
	prescGroup.Get("/my", h.GetMyPrescriptions)
	prescGroup.Get("/appointment/:appointment_id", h.GetPrescriptionByAppointment)
	prescGroup.Get("/patient/:patient_id", h.GetPatientPrescriptionsByID)
	prescGroup.Get("/:id", h.GetPrescriptionByID)
	prescGroup.Get("/:id/download", h.DownloadPrescription)
	// P5: refill + adherence
	prescGroup.Get("/:id/items", GetPrescriptionItems)
	prescGroup.Post("/items/:id/refill", RecordRefillOrder)
	prescGroup.Get("/items/:id/adherence", GetItemAdherence)

	diagGroup := router.Group("/diagnosis", middleware.Protected())
	diagGroup.Post("/", h.CreateDiagnosis)
	diagGroup.Get("/history", h.GetDiagnosisHistory)
	diagGroup.Get("/:id", h.GetDiagnosisHistory)
	diagGroup.Put("/:id", h.UpdateDiagnosis)
	diagGroup.Delete("/:id", h.DeleteDiagnosis)
	diagGroup.Get("/appointment/:appointment_id", h.GetDiagnosisByAppointment)

	medRecsGroup := router.Group("/medical-records", middleware.Protected())
	medRecsGroup.Post("/", h.CreateMedicalRecord)
	medRecsGroup.Get("/", h.GetMedicalRecords)
	medRecsGroup.Get("/me", h.GetMyMedicalRecords)
	medRecsGroup.Get("/:id", h.GetRecordDetails)
	medRecsGroup.Put("/:id", h.UpdateMedicalRecord)
	medRecsGroup.Delete("/:id", h.DeleteMedicalRecord)
	medRecsGroup.Get("/timeline/:id", h.GetPatientTimeline)

	patientGroup := router.Group("/patient", middleware.Protected())
	patientGroup.Get("/:id/history", h.GetPatientHistory)
}

func (h *Handler) registerEMRRoutes(group fiber.Router) {
	// Prescriptions
	group.Post("/prescriptions", middleware.RequireRoles("doctor"), h.CreatePrescription) // doctors only
	group.Get("/prescriptions", h.GetPrescriptions)                                    // doctor: their issued prescriptions
	group.Get("/prescriptions/my", h.GetMyPrescriptions)                               // patient: their received prescriptions
	group.Get("/prescriptions/appointment/:appointment_id", h.GetPrescriptionByAppointment) // by appointment
	group.Get("/prescriptions/patient/:patient_id", h.GetPatientPrescriptionsByID)     // admin/doctor: list patient prescriptions
	group.Get("/prescriptions/:id", h.GetPrescriptionByID)                             // get single prescription by ID
	group.Get("/prescriptions/:id/download", h.DownloadPrescription)                   // stream PDF

	// Diagnosis
	group.Post("/diagnosis", h.CreateDiagnosis)
	group.Get("/diagnosis/history", h.GetDiagnosisHistory)
	group.Get("/diagnosis/:id", h.GetDiagnosisHistory) // Alias for history/details
	group.Put("/diagnosis/:id", h.UpdateDiagnosis)
	group.Delete("/diagnosis/:id", h.DeleteDiagnosis)
	group.Get("/diagnosis/appointment/:appointment_id", h.GetDiagnosisByAppointment)

	// Medical Records
	medicalRecords := group.Group("/medical-records")
	medicalRecords.Post("/", h.CreateMedicalRecord)
	medicalRecords.Get("/", h.GetMedicalRecords)
	medicalRecords.Get("/me", h.GetMyMedicalRecords)
	medicalRecords.Get("/:id", h.GetRecordDetails)
	medicalRecords.Put("/:id", h.UpdateMedicalRecord)
	medicalRecords.Delete("/:id", h.DeleteMedicalRecord)
	medicalRecords.Get("/timeline/:id", h.GetPatientTimeline)
	
	// Patient History Alias
	group.Get("/patient/:id/history", h.GetPatientHistory)
}

// CreatePrescription creates a new prescription for a patient
// @Summary Create Prescription
// @Description Issue a new digital prescription for a patient including medication and dosage instructions
// @Tags EMR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body CreatePrescriptionRequest true "Prescription Request"
// @Success 201 {object} response.StandardResponse "Prescription created"
// @Router /api/v1/emr/prescriptions [post]
func (h *Handler) CreatePrescription(c *fiber.Ctx) error {
	var req CreatePrescriptionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	doctorID := c.Locals("user_id").(uint)
	p, err := h.svc.CreatePrescription(c.Context(), req, doctorID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}

	go func() {
		var patient, doctor identity.User
		if err := h.db.Select("full_name, email, phone_number").First(&patient, p.PatientID).Error; err != nil {
			return
		}
		if err := h.db.Select("full_name, email").First(&doctor, doctorID).Error; err != nil {
			return
		}
		dateStr := time.Now().Format("02 Jan 2006")
		notification.EnqueueEmail(context.Background(),
			patient.Email,
			"Prescription Issued — Dr.Ayya",
			email.PrescriptionIssuedEmail(email.PrescriptionEmailData{
				PatientName: patient.FullName,
				DoctorName:  doctor.FullName,
				Date:        dateStr,
				Diagnosis:   fmt.Sprintf("Prescription #%d", p.ID),
				PrescID:     p.ID,
			}))
		if patient.PhoneNumber != "" {
			notification.WhatsAppPrescriptionIssued(patient.PhoneNumber, patient.FullName, doctor.FullName, dateStr)
		}
	}()

	return response.JSON(c, fiber.StatusCreated, p, "Prescription created")
}

// GetMyPrescriptions returns prescriptions for the authenticated patient.
// GET /emr/prescriptions/my  (optional: ?appointment_id=X)
func (h *Handler) GetMyPrescriptions(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	prescs, err := h.svc.GetPatientPrescriptions(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	// Optional filter by appointment_id
	if apptIDStr := c.Query("appointment_id"); apptIDStr != "" {
		apptID, _ := strconv.ParseUint(apptIDStr, 10, 32)
		filtered := prescs[:0]
		for _, p := range prescs {
			if p.AppointmentID == uint(apptID) {
				filtered = append(filtered, p)
			}
		}
		prescs = filtered
	}

	return response.JSON(c, fiber.StatusOK, prescs, "Prescriptions fetched")
}

// DownloadPrescription streams the PDF for a prescription.
// GET /emr/prescriptions/:id/download
// Both the issuing doctor and the patient can download.
func (h *Handler) DownloadPrescription(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid prescription ID")
	}

	userID := c.Locals("user_id").(uint)

	var presc Prescription
	if err := h.db.First(&presc, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Prescription not found")
	}

	// Access control: only the doctor who issued it, the patient, admin, or a doctor with active access to the patient can download
	isAuthorized := presc.DoctorID == userID || presc.PatientID == userID
	role, _ := c.Locals("role").(string)
	if !isAuthorized && (role == "admin" || role == "super_admin") {
		isAuthorized = true
	}
	if !isAuthorized && isEMRDoctor(role) && h.doctorHasActiveAccessToPatient(userID, presc.PatientID) {
		isAuthorized = true
	}

	if !isAuthorized {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied")
	}

	if presc.PdfURL == "" {
		return response.Error(c, fiber.StatusNotFound, "NO_PDF", "PDF not generated for this prescription")
	}

	// Vault-stored PDF (vault:<id>) — redirect to a short-lived signed URL. The
	// access check above already authorized this caller.
	if strings.HasPrefix(presc.PdfURL, "vault:") {
		if did, e := strconv.Atoi(strings.TrimPrefix(presc.PdfURL, "vault:")); e == nil {
			url, ue := document.SignedURLForID(c.UserContext(), uint(did))
			if ue != nil {
				return response.Error(c, fiber.StatusNotFound, "PDF_MISSING", "PDF not available")
			}
			return c.Redirect(url, fiber.StatusFound)
		}
		return response.Error(c, fiber.StatusNotFound, "PDF_MISSING", "PDF not available")
	}

	// Legacy local file path (e.g. /tmp/prescription_123_abc.pdf).
	if _, err := os.Stat(presc.PdfURL); os.IsNotExist(err) {
		return response.Error(c, fiber.StatusNotFound, "PDF_MISSING", "PDF file not found on server")
	}

	filename := fmt.Sprintf("prescription-%d.pdf", presc.ID)
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(filename)))
	c.Set("Content-Type", "application/pdf")
	return c.SendFile(presc.PdfURL)
}

// GetPrescriptions returns prescriptions issued by the authenticated doctor
// @Summary Get Doctor Prescriptions
// @Description Fetch all prescriptions written by the current logged-in doctor
// @Tags EMR
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse "Prescriptions fetched"
// @Router /api/v1/emr/prescriptions [get]
func (h *Handler) GetPrescriptions(c *fiber.Ctx) error {
	doctorID := c.Locals("user_id").(uint)
	prescs, err := h.svc.GetDoctorPrescriptions(c.Context(), doctorID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, prescs, "Prescriptions fetched")
}

// GetPrescriptionByID returns a single prescription by its ID.
// Access: the issuing doctor, the patient it was issued for, or an admin.
// GET /emr/prescriptions/:id
func (h *Handler) GetPrescriptionByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid prescription ID")
	}

	callerID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	presc, err := h.svc.GetPrescriptionByID(c.Context(), uint(id), callerID, role)
	if err != nil {
		if err.Error() == "forbidden: access denied" {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied")
		}
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Prescription not found")
	}

	return response.JSON(c, fiber.StatusOK, presc, "Prescription fetched")
}

// GetPrescriptionByAppointment returns the prescription for a given appointment.
// Access: the doctor who owns the appointment or the patient.
// GET /emr/prescriptions/appointment/:appointment_id
func (h *Handler) GetPrescriptionByAppointment(c *fiber.Ctx) error {
	apptID, err := strconv.ParseUint(c.Params("appointment_id"), 10, 32)
	if err != nil || apptID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid appointment ID")
	}

	callerID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	presc, err := h.svc.GetPrescriptionByAppointment(c.Context(), uint(apptID), callerID, role)
	if err != nil {
		if err.Error() == "forbidden: access denied" {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied")
		}
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Prescription not found for this appointment")
	}

	return response.JSON(c, fiber.StatusOK, presc, "Prescription fetched")
}

// GetPatientPrescriptionsByID lists all prescriptions for a given patient ID.
// Access: the patient themselves, their treating doctors, or admins.
// GET /emr/prescriptions/patient/:patient_id
func (h *Handler) GetPatientPrescriptionsByID(c *fiber.Ctx) error {
	patientID, err := strconv.ParseUint(c.Params("patient_id"), 10, 32)
	if err != nil || patientID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid patient ID")
	}

	callerID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	prescs, err := h.svc.GetPatientPrescriptionsByID(c.Context(), uint(patientID), callerID, role)
	if err != nil {
		if err.Error() == "forbidden: access denied" {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied")
		}
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, prescs, "Patient prescriptions fetched")
}

// CreateDiagnosis records a new patient diagnosis
// @Summary Create Diagnosis
// @Description Save a clinical diagnosis for a patient during or after an appointment
// @Tags EMR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param diagnosis body Diagnosis true "Diagnosis Data"
// @Success 201 {object} response.StandardResponse "Diagnosis saved"
// @Router /api/v1/emr/diagnosis [post]
func (h *Handler) CreateDiagnosis(c *fiber.Ctx) error {
	var diag Diagnosis
	if err := c.BodyParser(&diag); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	// Force DoctorID from token — never trust client-supplied value
	doctorID := c.Locals("user_id").(uint)
	diag.DoctorID = doctorID

	res, err := h.svc.CreateDiagnosis(c.Context(), diag)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to save diagnosis")
	}

	return response.JSON(c, fiber.StatusCreated, res, "Diagnosis saved")
}

// GetDiagnosisHistory returns diagnosis history for a patient
// @Summary Get Diagnosis History
// @Description Fetch all past diagnoses recorded for a specific patient
// @Tags EMR
// @Produce json
// @Security ApiKeyAuth
// @Param patient_id query string true "Patient ID"
// @Success 200 {object} response.StandardResponse "History fetched"
// @Router /api/v1/emr/diagnosis/history [get]
func (h *Handler) GetDiagnosisHistory(c *fiber.Ctx) error {
	patientIDStr := c.Query("patient_id")
	if patientIDStr == "" {
		return response.Error(c, fiber.StatusBadRequest, "PATIENT_ID_REQUIRED", "patient_id required")
	}

	pID, _ := strconv.ParseUint(patientIDStr, 10, 32)
	if pID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_PATIENT_ID", "Invalid patient ID")
	}
	callerID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	if !isEMRAdmin(role) {
		if uint(pID) == callerID {
			// Patients can read their own diagnosis history.
		} else if isEMRDoctor(role) && h.doctorHasActiveAccessToPatient(callerID, uint(pID)) {
			// Doctors can read diagnosis history only for patients they have an active/recent appointment with.
		} else {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied")
		}
	}

	history, err := h.svc.GetDiagnosisHistory(c.Context(), uint(pID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, history, "Diagnosis history fetched")
}

// CreateMedicalRecord creates a new general medical record
// @Summary Create Medical Record
// @Description Save a general medical record (e.g., notes, clinical summaries) for a patient
// @Tags EMR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param record body MedicalRecord true "Record Data"
// @Success 201 {object} response.StandardResponse "Record created"
// @Router /api/v1/emr/medical-records [post]
func (h *Handler) CreateMedicalRecord(c *fiber.Ctx) error {
	var rec MedicalRecord
	if err := c.BodyParser(&rec); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	callerID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	// If patient_id is not provided, default to current user if they are a patient
	if rec.PatientID == 0 && role == "patient" {
		rec.PatientID = callerID
	}

	// Basic Validation
	if rec.PatientID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_PATIENT", "Patient ID is required")
	}

	res, err := h.svc.CreateMedicalRecord(c.Context(), rec)
	if err != nil {
		fmt.Printf("[EMR ERROR] CreateMedicalRecord failed for user %d: %v\n", callerID, err)
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create medical record: "+err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, res, "Medical record created")
}

// GetMedicalRecords lists medical records with filters
// @Summary List Medical Records
// @Description Fetch medical records filtered by patient or doctor ID
// @Tags EMR
// @Produce json
// @Security ApiKeyAuth
// @Param patient_id query string false "Patient ID"
// @Param doctor_id query string false "Doctor ID"
// @Success 200 {object} response.StandardResponse "Records fetched"
// @Router /api/v1/emr/medical-records [get]
func (h *Handler) GetMedicalRecords(c *fiber.Ctx) error {
	callerID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	pIDStr := c.Query("patient_id")
	dIDStr := c.Query("doctor_id")

	var pID, dID uint64
	if pIDStr != "" {
		pID, _ = strconv.ParseUint(pIDStr, 10, 32)
	}
	if dIDStr != "" {
		dID, _ = strconv.ParseUint(dIDStr, 10, 32)
	}

	// Non-admin callers can only query their own records
	if role != "admin" && role != "super_admin" {
		if uint(pID) != 0 && uint(pID) != callerID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You can only access your own records")
		}
		if uint(dID) != 0 && uint(dID) != callerID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You can only access your own records")
		}
		// If neither filter is set, default to own records
		if pID == 0 && dID == 0 {
			pID = uint64(callerID)
		}
	}

	records, err := h.svc.GetMedicalRecords(c.Context(), uint(pID), uint(dID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, records, "Medical records fetched")
}

// GetMyMedicalRecords returns records for the authenticated patient
// @Summary Get My Medical Records
// @Description Fetch all medical records belonging to the current authenticated patient
// @Tags EMR
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse "Records fetched"
// @Router /api/v1/emr/medical-records/me [get]
func (h *Handler) GetMyMedicalRecords(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	records, err := h.svc.GetMedicalRecords(c.Context(), userID, 0)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch records")
	}

	return response.JSON(c, fiber.StatusOK, records, "My medical records fetched")
}

// GetRecordDetails returns details of a specific medical record
// @Summary Get Medical Record Details
// @Description Fetch full details of a single medical record by ID
// @Tags EMR
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Record ID"
// @Success 200 {object} response.StandardResponse "Details fetched"
// @Router /api/v1/emr/medical-records/{id} [get]
func (h *Handler) GetRecordDetails(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)
	record, err := h.svc.GetRecordDetails(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Record not found")
	}

	// Ownership check: only the patient, the recording doctor, admin, or a doctor with active access to the patient can view
	callerID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	isAuthorized := role == "admin" || role == "super_admin" || record.PatientID == callerID || record.DoctorID == callerID
	if !isAuthorized && isEMRDoctor(role) && h.doctorHasActiveAccessToPatient(callerID, record.PatientID) {
		isAuthorized = true
	}

	if !isAuthorized {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied")
	}

	return response.JSON(c, fiber.StatusOK, record, "Medical record details fetched")
}

// UpdateMedicalRecord updates an existing medical record
// @Summary Update Medical Record
// @Description Update the content or metadata of an existing medical record
// @Tags EMR
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Record ID"
// @Param updates body map[string]interface{} true "Fields to update"
// @Success 200 {object} response.StandardResponse "Record updated"
// @Router /api/v1/emr/medical-records/{id} [put]
func (h *Handler) UpdateMedicalRecord(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Ownership check: only the doctor who created this record (or admin) may update it
	callerID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	if role != "admin" && role != "super_admin" {
		record, err := h.svc.GetRecordDetails(c.Context(), uint(id))
		if err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Record not found")
		}
		if record.DoctorID != callerID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Only the recording doctor can update this record")
		}
	}

	if err := h.svc.UpdateMedicalRecord(c.Context(), uint(id), updates); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Update failed")
	}

	return response.JSON(c, fiber.StatusOK, nil, "Record updated")
}

// DeleteMedicalRecord removes a medical record
// @Summary Delete Medical Record
// @Description Soft-delete or remove a specific medical record from the system
// @Tags EMR
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Record ID"
// @Success 200 {object} response.StandardResponse "Record deleted"
// @Router /api/v1/emr/medical-records/{id} [delete]
func (h *Handler) DeleteMedicalRecord(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)

	// Ownership check: only the doctor who created this record (or admin) may delete it
	callerID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	if role != "admin" && role != "super_admin" {
		record, err := h.svc.GetRecordDetails(c.Context(), uint(id))
		if err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Record not found")
		}
		if record.DoctorID != callerID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Only the recording doctor can delete this record")
		}
	}

	if err := h.svc.DeleteMedicalRecord(c.Context(), uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", "Delete failed")
	}
	return response.JSON(c, fiber.StatusOK, nil, "Record deleted")
}

// GetPatientTimeline returns a chronological timeline of patient encounters
// @Summary Get Patient Timeline
// @Description Fetch a chronological list of all medical encounters and events for a patient
// @Tags EMR
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Patient ID"
// @Success 200 {object} response.StandardResponse "Timeline fetched"
// @Router /api/v1/emr/medical-records/timeline/{id} [get]
func (h *Handler) GetPatientTimeline(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)

	// Only the patient themselves or a doctor/admin may view a patient's timeline
	callerID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	if role != "admin" && role != "super_admin" && (role != "doctor" || !h.doctorHasActiveAccessToPatient(callerID, uint(id))) && uint(id) != callerID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied")
	}

	timeline, err := h.svc.GetPatientTimeline(c.Context(), uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, timeline, "Patient timeline fetched")
}

// GetPatientHistory returns a patient's historical medical timeline
// @Summary Get Patient History
// @Description Fetch comprehensive chronological history for a patient
// @Tags EMR
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Patient ID"
// @Success 200 {object} response.StandardResponse "History fetched"
// @Router /api/v1/patient/{id}/history [get]
func (h *Handler) GetPatientHistory(c *fiber.Ctx) error {
	return h.GetPatientTimeline(c)
}

func (h *Handler) UpdateDiagnosis(c *fiber.Ctx) error {
	id := c.Params("id")
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}
	// Simplified update
	if err := h.db.Table("diagnosis").Where("id = ?", id).Updates(updates).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Diagnosis updated")
}

func (h *Handler) DeleteDiagnosis(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.db.Table("diagnosis").Delete(nil, id).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Diagnosis deleted")
}

func (h *Handler) GetDiagnosisByAppointment(c *fiber.Ctx) error {
	apptID := c.Params("appointment_id")
	var history []Diagnosis
	if err := h.db.Where("appointment_id = ?", apptID).Find(&history).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	if len(history) > 0 {
		callerID := c.Locals("user_id").(uint)
		role, _ := c.Locals("role").(string)
		patientID := history[0].PatientID

		isAuthorized := role == "admin" || role == "super_admin" || patientID == callerID
		if !isAuthorized {
			for _, diag := range history {
				if diag.DoctorID == callerID {
					isAuthorized = true
					break
				}
			}
		}
		if !isAuthorized && isEMRDoctor(role) && h.doctorHasActiveAccessToPatient(callerID, patientID) {
			isAuthorized = true
		}

		if !isAuthorized {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Access denied")
		}
	}

	return response.JSON(c, fiber.StatusOK, history, "Diagnosis fetched for appointment")
}
