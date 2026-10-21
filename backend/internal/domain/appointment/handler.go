package appointment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/events"
	"drayya-core/internal/middleware"
	chatSvc "drayya-core/internal/services/chat"
	"drayya-core/internal/services/email"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/utils/response"
	"drayya-core/internal/utils/validator"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// publishApptWS sends a JSON message to a Redis room channel routed by the WS hub.
func publishApptWS(room string, payload map[string]interface{}) {
	if database.Rdb == nil {
		return
	}
	payload["room_id"] = room
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Appt] publishApptWS marshal error: %v", err)
		return
	}
	database.Rdb.Publish(context.Background(), "chat:global", data)
}

type Handler struct {
	Service  Service
	eventBus *events.Bus
}

func NewHandler(service Service) *Handler {
	return &Handler{Service: service}
}

// SetEventBus injects the domain event bus after construction.
func (h *Handler) SetEventBus(bus *events.Bus) {
	h.eventBus = bus
}

func isAdminRole(role string) bool {
	return role == "admin" || role == "super_admin"
}

func isDoctorRole(role string) bool {
	return role == "doctor" || role == "chief_doctor" || role == "hospital_head_doctor"
}

func (h *Handler) currentUser(c *fiber.Ctx) (uint, string) {
	userID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	return userID, role
}

func (h *Handler) authorizeAppointment(c *fiber.Ctx, appt *Appointment, action string) error {
	if appt == nil {
		return response.Error(c, fiber.StatusNotFound, "APPOINTMENT_NOT_FOUND", "Appointment not found")
	}
	userID, role := h.currentUser(c)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	if isAdminRole(role) {
		return nil
	}

	isPatient := appt.PatientID == userID
	isDoctor := appt.DoctorID != nil && *appt.DoctorID == userID

	switch action {
	case "view", "history":
		if isPatient || isDoctor {
			return nil
		}
	case "cancel":
		if isPatient || isDoctor {
			return nil
		}
	case "reschedule":
		if isPatient {
			return nil
		}
	case "start", "complete", "no-show", "notes", "doctor_status", "update":
		if isDoctor && isDoctorRole(role) {
			return nil
		}
	}

	return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You are not allowed to access this appointment")
}

func sanitizeAppointmentUpdates(updates map[string]interface{}) map[string]interface{} {
	for _, blocked := range []string{
		"id", "patient_id", "doctor_id", "consultation_fee", "payout_status",
		"created_at", "updated_at", "status",
	} {
		delete(updates, blocked)
		delete(updates, strings.ToUpper(blocked))
	}
	return updates
}

// CreateAppointment handles new appointment creation
// @Summary Create Appointment
// @Description Book a new appointment for a patient
// @Tags Appointment
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body object true "Appointment Data"
// @Success 201 {object} response.StandardResponse{data=Appointment} "Appointment created"
// @Failure 401 {object} response.StandardError "Unauthorized"
// @Router /api/v1/appointments [post]
func (h *Handler) CreateAppointment(c *fiber.Ctx) error {
	var body struct {
		PatientID        uint   `json:"patient_id"`
		DoctorID         *uint  `json:"doctor_id"`
		StartTime        string `json:"start_time"`
		EndTime          string `json:"end_time"`
		AppointmentType  string `json:"appointment_type" validate:"required"`
		ConsultationMode string `json:"consultation_mode" validate:"required"`
		Notes            string `json:"notes"`
		FamilyMemberID   *uint  `json:"family_member_id"`
		Type             string `json:"type"`           // Backwards compatibility with alternative payload keys
		MedicalSystem    string `json:"medical_system"` // siddha | ayurveda | yoga | psychology
	}

	if err := c.BodyParser(&body); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	if err := validator.ValidateStruct(c, &body); err != nil {
		return err
	}

	appType := body.AppointmentType
	if appType == "" {
		appType = body.Type
	}

	userID, role := h.currentUser(c)
	patientID := userID
	if isAdminRole(role) && body.PatientID != 0 {
		patientID = body.PatientID
	}

	if patientID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized: Patient ID required")
	}

	// Psychology is a restricted medical system — only selected/referred patients
	// may book it. Admins booking on a patient's behalf bypass.
	if strings.EqualFold(body.MedicalSystem, "psychology") && !isAdminRole(role) {
		var eligible bool
		database.DB.Table("user").Where("id = ?", patientID).Select("psychology_eligible").Scan(&eligible)
		if !eligible {
			return response.Error(c, fiber.StatusForbidden, "PSYCHOLOGY_NOT_ELIGIBLE",
				"Psychology consultations are available to selected patients only. Please ask your doctor for a referral.")
		}
	}

	appt, err := h.Service.CreateAppointment(c.Context(), CreateAppointmentRequest{
		PatientID:      patientID,
		DoctorID:       body.DoctorID,
		Type:           appType,
		Mode:           body.ConsultationMode,
		Notes:          body.Notes,
		FamilyMemberID: body.FamilyMemberID,
		StartTime:      body.StartTime,
		EndTime:        body.EndTime,
	})

	if err != nil {
		status := fiber.StatusInternalServerError
		code := "APPOINTMENT_CREATION_FAILED"

		if err.Error() == "insufficient wallet balance" {
			status = fiber.StatusPaymentRequired
			code = "INSUFFICIENT_BALANCE"
		} else if err.Error() == "patient not found" || err.Error() == "doctor not found" {
			status = fiber.StatusBadRequest
			code = "INVALID_REQUEST"
		} else if strings.HasPrefix(err.Error(), "invalid ") {
			status = fiber.StatusBadRequest
			code = "INVALID_REQUEST"
		}

		return response.Error(c, status, code, err.Error())
	}

	// Send confirmation email to patient (non-blocking)
	go sendAppointmentEmails(appt, patientID)

	// Real-time WS notifications
	if appt.DoctorID != nil {
		// Scheduled booking: notify the specific assigned doctor
		go func(a *Appointment, pid uint) {
			patientName := fmt.Sprintf("Patient #%d", pid)
			var pu identity.User
			if database.DB.Select("full_name").First(&pu, pid).Error == nil && pu.FullName != "" {
				patientName = pu.FullName
			}
			publishApptWS(fmt.Sprintf("room:user:%d", *a.DoctorID), map[string]interface{}{
				"type": "appointment_booked",
				"metadata": map[string]interface{}{
					"appointment_id":    a.ID,
					"patient_name":      patientName,
					"start_time":        a.StartTime,
					"consultation_mode": a.ConsultationMode,
					"appointment_type":  a.AppointmentType,
				},
			})
		}(appt, patientID)
	} else if appt.AppointmentType == "instant" || appt.AppointmentType == "emergency" {
		// Instant/SOS with no pre-assigned doctor: broadcast to ALL online doctors immediately
		// so the WaitingCallDashboard lights up before the dispatch worker finishes matching.
		go func(a *Appointment, pid uint) {
			patientName := fmt.Sprintf("Patient #%d", pid)
			lang := ""
			var pu identity.User
			if database.DB.Select("full_name, preferred_language").First(&pu, pid).Error == nil {
				if pu.FullName != "" {
					patientName = pu.FullName
				}
				lang = pu.PreferredLanguage
			}
			// preferred_language lets language-matched doctors prioritise the
			// request on their waiting dashboard (Tamil patient → Tamil doctor).
			publishApptWS("room:doctors:instant", map[string]interface{}{
				"type": "instant_call_request",
				"metadata": map[string]interface{}{
					"appointment_id":     a.ID,
					"patient_name":       patientName,
					"consultation_mode":  a.ConsultationMode,
					"type":               a.AppointmentType,
					"preferred_language": lang,
				},
			})
		}(appt, patientID)
	}

	return response.JSON(c, fiber.StatusCreated, fiber.Map{
		"appointment": appt,
		"is_free":     appt.ConsultationFee == 0,
	}, "Appointment created successfully")
}

// sendAppointmentNotifications dispatches confirmation emails + WhatsApp messages
// to both patient and doctor. Runs in a goroutine — never blocks the HTTP response.
func sendAppointmentEmails(appt *Appointment, patientUserID uint) {
	if appt == nil {
		return
	}
	db := database.DB

	var patient identity.User
	if err := db.Select("full_name, email, phone_number").First(&patient, patientUserID).Error; err != nil {
		return
	}

	doctorName := "your doctor"
	doctorEmail := ""
	doctorPhone := ""
	if appt.DoctorID != nil {
		var doc identity.User
		if db.Select("full_name, email, phone_number").First(&doc, *appt.DoctorID).Error == nil {
			doctorName = doc.FullName
			doctorEmail = doc.Email
			doctorPhone = doc.PhoneNumber
		}
	}

	dateStr := appt.StartTime.Format("Mon, 02 Jan 2006")
	timeStr := appt.StartTime.Format("3:04 PM")

	d := email.AppointmentEmailData{
		PatientName:   patient.FullName,
		DoctorName:    doctorName,
		Date:          dateStr,
		Time:          timeStr,
		Mode:          appt.ConsultationMode,
		AppointmentID: appt.ID,
		Fee:           float64(appt.ConsultationFee) / 100.0,
	}

	// ── Patient ──
	if patient.Email != "" {
		notification.EnqueueEmail(context.Background(),
			patient.Email,
			fmt.Sprintf("Appointment Confirmed with Dr. %s — Dr.Ayya", doctorName),
			email.AppointmentConfirmation(d),
		)
	}
	if patient.PhoneNumber != "" {
		notification.WhatsAppAppointmentBooked(patient.PhoneNumber, patient.FullName, doctorName, dateStr, timeStr)
	}

	// ── Doctor ──
	if doctorEmail != "" {
		notification.EnqueueEmail(context.Background(),
			doctorEmail,
			"New Appointment — Dr.Ayya",
			email.DoctorAppointmentNotice(d),
		)
	}
	if doctorPhone != "" {
		notification.WhatsAppDoctorNewAppointment(doctorPhone, doctorName, patient.FullName, dateStr, timeStr)
	}
}

// GetUpcomingAppointments returns future appointments for the user
// @Summary List Upcoming Appointments
// @Description Get all upcoming appointments for the authenticated patient/doctor
// @Tags Appointment
// @Produce json
// @Security ApiKeyAuth
// @Param user_id query uint false "User ID (optional if authenticated)"
// @Success 200 {object} response.StandardResponse{data=[]Appointment} "Appointments fetched"
// @Router /api/v1/appointments/upcoming [get]
func (h *Handler) GetUpcomingAppointments(c *fiber.Ctx) error {
	userID := h.extractUserID(c)
	if userID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "USER_ID_REQUIRED", "user_id required")
	}

	appts, err := h.Service.GetUpcomingAppointments(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, appts, "Upcoming appointments fetched")
}

func (h *Handler) GetPastAppointments(c *fiber.Ctx) error {
	userID := h.extractUserID(c)
	if userID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "USER_ID_REQUIRED", "user_id required")
	}

	appts, err := h.Service.GetPastAppointments(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, appts, "Past appointments fetched")
}

// GetAppointment returns single appointment details
// @Summary Get Appointment Details
// @Description Get full details of a specific appointment
// @Tags Appointment
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Appointment ID"
// @Success 200 {object} response.StandardResponse{data=Appointment} "Details fetched"
// @Failure 404 {object} response.StandardError "Not found"
// @Router /api/v1/appointments/{id} [get]
func (h *Handler) GetAppointment(c *fiber.Ctx) error {
	id := c.Params("id")
	appt, err := h.Service.GetAppointmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "APPOINTMENT_NOT_FOUND", "Appointment not found")
	}
	if err := h.authorizeAppointment(c, appt, "view"); err != nil {
		return err
	}
	return response.JSON(c, fiber.StatusOK, appt, "Appointment details fetched")
}

func (h *Handler) UpdateAppointment(c *fiber.Ctx) error {
	id := c.Params("id")
	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	appt, err := h.Service.GetAppointmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if err := h.authorizeAppointment(c, appt, "update"); err != nil {
		return err
	}
	updates = sanitizeAppointmentUpdates(updates)
	if len(updates) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "No updatable fields provided")
	}

	if err := h.Service.UpdateAppointment(id, updates); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Appointment updated successfully")
}

func (h *Handler) DeleteAppointment(c *fiber.Ctx) error {
	id := c.Params("id")
	appt, err := h.Service.GetAppointmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if err := h.authorizeAppointment(c, appt, "cancel"); err != nil {
		return err
	}
	if err := h.Service.CancelAppointment(id, "Cancelled by user"); err != nil {
		if strings.Contains(err.Error(), "conflict") {
			return response.Error(c, fiber.StatusConflict, "VERSION_CONFLICT", err.Error())
		}
		return response.Error(c, fiber.StatusInternalServerError, "CANCEL_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Appointment cancelled successfully")
}

func (h *Handler) CancelAppointment(c *fiber.Ctx) error {
	id := c.Params("id")
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	appt, err := h.Service.GetAppointmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if err := h.authorizeAppointment(c, appt, "cancel"); err != nil {
		return err
	}

	if err := h.Service.CancelAppointment(id, req.Reason); err != nil {
		if strings.Contains(err.Error(), "conflict") {
			return response.Error(c, fiber.StatusConflict, "VERSION_CONFLICT", err.Error())
		}
		return response.Error(c, fiber.StatusInternalServerError, "CANCEL_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Appointment cancelled successfully")
}

func (h *Handler) RescheduleAppointment(c *fiber.Ctx) error {
	id := c.Params("id")
	var req struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	appt, err := h.Service.GetAppointmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if err := h.authorizeAppointment(c, appt, "reschedule"); err != nil {
		return err
	}

	if err := h.Service.RescheduleAppointment(id, req.StartTime, req.EndTime); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "RESCHEDULE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Appointment rescheduled successfully")
}

func (h *Handler) CompleteAppointment(c *fiber.Ctx) error {
	id := c.Params("id")
	appt, err := h.Service.GetAppointmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if err := h.authorizeAppointment(c, appt, "complete"); err != nil {
		return err
	}
	if err := h.Service.CompleteAppointment(c.Context(), id); err != nil {
		if strings.Contains(err.Error(), "conflict") {
			return response.Error(c, fiber.StatusConflict, "VERSION_CONFLICT", err.Error())
		}
		return response.Error(c, fiber.StatusInternalServerError, "COMPLETION_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Appointment marked as complete")
}

func (h *Handler) StartAppointment(c *fiber.Ctx) error {
	id := c.Params("id")
	appt, err := h.Service.GetAppointmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if err := h.authorizeAppointment(c, appt, "start"); err != nil {
		return err
	}
	if err := h.Service.StartAppointment(id); err != nil {
		if strings.Contains(err.Error(), "conflict") {
			return response.Error(c, fiber.StatusConflict, "VERSION_CONFLICT", err.Error())
		}
		return response.Error(c, fiber.StatusInternalServerError, "START_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Appointment started")
}

// AcceptAppointment allows a doctor to accept a pending appointment
// @Summary Accept Appointment
// @Description Doctor accepts a pending appointment, assigning themselves to it
// @Tags Appointment
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Appointment ID"
// @Success 200 {object} response.StandardResponse "Appointment accepted"
// @Failure 400 {object} response.StandardError "Invalid status"
// @Failure 401 {object} response.StandardError "Unauthorized"
// @Failure 404 {object} response.StandardError "Not found"
// @Router /api/v1/appointments/{id}/accept [post]
func (h *Handler) AcceptAppointment(c *fiber.Ctx) error {
	id := c.Params("id")

	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Doctor ID not found in token")
	}
	role, _ := c.Locals("role").(string)
	if !isDoctorRole(role) && !isAdminRole(role) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Only doctors can accept appointments")
	}

	// SOS safety gate: an emergency request may only be accepted by an
	// emergency-capable doctor (scope-of-practice + medico-legal). Admins bypass
	// for operational override.
	if !isAdminRole(role) {
		if appt, aerr := h.Service.GetAppointmentByID(id); aerr == nil && appt.AppointmentType == "emergency" {
			var emergencyCapable bool
			database.DB.Table("doctorprofile").Where("user_id = ?", doctorID).
				Select("emergency_capable").Scan(&emergencyCapable)
			if !emergencyCapable {
				return response.Error(c, fiber.StatusForbidden, "NOT_EMERGENCY_CAPABLE",
					"Only emergency-capable doctors can accept SOS/emergency requests")
			}
		}
	}

	if err := h.Service.AcceptAppointment(id, doctorID); err != nil {
		if err.Error() == "appointment not found" {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", err.Error())
		}
		if strings.Contains(err.Error(), "conflict") {
			return response.Error(c, fiber.StatusConflict, "VERSION_CONFLICT", err.Error())
		}
		return response.Error(c, fiber.StatusBadRequest, "ACCEPT_FAILED", err.Error())
	}

	// Notify patient + pre-create chat room (non-blocking)
	go func() {
		appt, err := h.Service.GetAppointmentByID(id)
		if err != nil {
			return
		}

		// Pre-create chat room so both sides can join immediately
		if appt.ConsultationMode == "chat" && appt.DoctorID != nil {
			if _, cerr := chatSvc.GetOrCreateConsultationRoom(appt.ID, *appt.DoctorID, appt.PatientID); cerr != nil {
				log.Printf("[Appt] failed to create chat room for appointment %d: %v", appt.ID, cerr)
			}
		}

		// Real-time WebSocket push (stays in goroutine — this is the WS fanout path)
		publishApptWS(fmt.Sprintf("room:user:%d", appt.PatientID), map[string]interface{}{
			"type": "appointment_confirmed",
			"metadata": map[string]interface{}{
				"appointment_id":    appt.ID,
				"consultation_mode": appt.ConsultationMode,
				"doctor_id":         doctorID,
			},
		})
	}()

	// Publish domain event for reliable notification processing (FCM, SMS, etc.)
	if h.eventBus != nil {
		h.eventBus.Publish(c.Context(), events.EventAppointmentAccepted, doctorID, map[string]interface{}{
			"appointment_id": id,
			"doctor_id":      doctorID,
		})
	}

	return response.JSON(c, fiber.StatusOK, nil, "Appointment accepted successfully")
}

// DeclineAppointment allows the assigned doctor to decline a pending appointment.
func (h *Handler) DeclineAppointment(c *fiber.Ctx) error {
	idStr := c.Params("id")
	var appointmentID uint
	if strings.HasPrefix(idStr, "APT-") {
		appt, err := h.Service.GetAppointmentByID(idStr)
		if err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
		}
		appointmentID = appt.ID
	} else {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil || id == 0 {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid appointment ID")
		}
		appointmentID = uint(id)
	}

	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Doctor ID not found in token")
	}
	role, _ := c.Locals("role").(string)
	if !isDoctorRole(role) && !isAdminRole(role) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Only doctors can decline appointments")
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.BodyParser(&req) // reason is optional

	if err := h.Service.DeclineAppointment(appointmentID, doctorID, req.Reason); err != nil {
		if strings.Contains(err.Error(), "unauthorized") {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", err.Error())
		}
		if err.Error() == "appointment not found" {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", err.Error())
		}
		if strings.Contains(err.Error(), "conflict") {
			return response.Error(c, fiber.StatusConflict, "VERSION_CONFLICT", err.Error())
		}
		return response.Error(c, fiber.StatusBadRequest, "DECLINE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Appointment declined successfully")
}

func (h *Handler) MarkNoShow(c *fiber.Ctx) error {
	id := c.Params("id")
	appt, err := h.Service.GetAppointmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if err := h.authorizeAppointment(c, appt, "no-show"); err != nil {
		return err
	}
	if err := h.Service.MarkNoShow(id); err != nil {
		if strings.Contains(err.Error(), "conflict") {
			return response.Error(c, fiber.StatusConflict, "VERSION_CONFLICT", err.Error())
		}
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Appointment marked as No Show")
}

func (h *Handler) AddAppointmentNotes(c *fiber.Ctx) error {
	id := c.Params("id")
	var req struct {
		Notes string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	appt, err := h.Service.GetAppointmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if err := h.authorizeAppointment(c, appt, "notes"); err != nil {
		return err
	}

	if err := h.Service.AddNotes(id, req.Notes); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ADD_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Notes added successfully")
}

func (h *Handler) GetAppointmentHistory(c *fiber.Ctx) error {
	id := c.Params("id")
	appt, err := h.Service.GetAppointmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "APPOINTMENT_NOT_FOUND", "Appointment not found")
	}
	if err := h.authorizeAppointment(c, appt, "history"); err != nil {
		return err
	}

	history, err := h.Service.GetAppointmentHistory(&appt.PatientID, appt.DoctorID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, history, "Appointment history fetched")
}

func (h *Handler) GetUpcomingAppointmentsForDoctor(c *fiber.Ctx) error {
	doctorID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_DOCTOR_ID", "Invalid doctor ID")
	}
	
	callerID, role := h.currentUser(c)
	isOwner := !isAdminRole(role) && callerID == uint(doctorID)
	isAdmin := isAdminRole(role)

	appts, err := h.Service.GetUpcomingAppointmentsForDoctor(uint(doctorID))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	if appts == nil {
		appts = []Appointment{}
	}

	// If not admin and not the doctor themselves, sanitize the output to protect patient privacy
	if !isAdmin && !isOwner {
		sanitized := make([]fiber.Map, 0, len(appts))
		for _, a := range appts {
			sanitized = append(sanitized, fiber.Map{
				"id":         a.ID,
				"doctor_id":  a.DoctorID,
				"start_time": a.StartTime,
				"end_time":   a.EndTime,
				"status":     a.Status,
				// Do not include patient_id, notes, or patient profile
			})
		}
		return response.JSON(c, fiber.StatusOK, sanitized, "Upcoming doctor appointments fetched (sanitized)")
	}

	return response.JSON(c, fiber.StatusOK, appts, "Upcoming doctor appointments fetched")
}

func (h *Handler) MarkDoctorEnRoute(c *fiber.Ctx) error {
	id := c.Params("id")
	appt, err := h.Service.GetAppointmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if err := h.authorizeAppointment(c, appt, "doctor_status"); err != nil {
		return err
	}
	if err := h.Service.UpdateDoctorStatus(id, StatusDoctorEnRoute); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Doctor status updated to En Route")
}

func (h *Handler) MarkDoctorArrived(c *fiber.Ctx) error {
	id := c.Params("id")
	appt, err := h.Service.GetAppointmentByID(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if err := h.authorizeAppointment(c, appt, "doctor_status"); err != nil {
		return err
	}
	if err := h.Service.UpdateDoctorStatus(id, StatusDoctorArrived); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Doctor status updated to Arrived")
}

func (h *Handler) extractUserID(c *fiber.Ctx) uint {
	if val := c.Locals("user_id"); val != nil {
		if id, ok := val.(uint); ok {
			return id
		}
		// user_id local is set but is not uint — log for diagnosis
		log.Printf("[Appointment] WARNING: user_id local has unexpected type %T (value=%v)", val, val)
	}
	// Fallback: read from user_id_str (set when ParseUint fails in Protected())
	if str, _ := c.Locals("user_id_str").(string); str != "" {
		if id, err := strconv.ParseUint(str, 10, 32); err == nil {
			return uint(id)
		}
	}
	if role, _ := c.Locals("role").(string); isAdminRole(role) {
		if qID := c.Query("user_id"); qID != "" {
			if id, err := strconv.ParseUint(qID, 10, 32); err == nil {
				return uint(id)
			}
		}
	}
	return 0
}

func (h *Handler) GetInstantPending(c *fiber.Ctx) error {
	var appts []Appointment
	var err error

	// Doctors get a personalised view: all pending (queue) + any confirmed assigned to them.
	// This ensures the WaitingCallDashboard stays populated even after the dispatch worker has
	// pre-confirmed and set doctor_id — a state the generic 'pending-only' query would miss.
	role, _ := c.Locals("role").(string)
	callerID, _ := c.Locals("user_id").(uint)

	if role == "doctor" && callerID > 0 {
		appts, err = h.Service.GetInstantQueueForDoctor(callerID)
	} else {
		appts, err = h.Service.GetInstantPendingAppointments()
	}

	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	if appts == nil {
		appts = []Appointment{}
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"requests": appts, "total": len(appts)}, "Instant pending fetched")
}

// GetMyPatients returns every distinct patient who has ever booked with the
// calling doctor — with last-visit date, total visits, age, gender. Real data,
// not a mock array. The doctor's own user_id is the filter.
//
// GET /appointments/doctor/my-patients?q=<search>
func (h *Handler) GetMyPatients(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Doctor ID not found in token")
	}
	role, _ := c.Locals("role").(string)
	if !isDoctorRole(role) && !isAdminRole(role) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Only doctors can view their patient list")
	}

	type row struct {
		ID           uint    `json:"id"`
		Name         string  `json:"name"`
		Phone        string  `json:"phone"`
		Email        string  `json:"email"`
		Gender       string  `json:"gender"`
		DateOfBirth  *string `json:"date_of_birth"`
		LastVisit    *string `json:"last_visit"`
		TotalVisits  int64   `json:"total_visits"`
	}
	q := database.DB.Table("appointments AS a").
		Select(`u.id, u.full_name AS name, u.phone_number AS phone, u.email,
			COALESCE(pp.gender, u.gender) AS gender,
			pp.date_of_birth,
			MAX(a.start_time) AS last_visit,
			COUNT(*) AS total_visits`).
		Joins(`JOIN "user" u ON u.id = a.patient_id`).
		Joins(`LEFT JOIN patient_profiles pp ON pp.user_id = u.id`).
		Where("a.doctor_id = ?", doctorID).
		Group("u.id, u.full_name, u.phone_number, u.email, pp.gender, u.gender, pp.date_of_birth").
		Order("MAX(a.start_time) DESC")
	if search := strings.TrimSpace(c.Query("q")); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		q = q.Having("LOWER(u.full_name) LIKE ? OR u.phone_number LIKE ?", like, like)
	}
	var rows []row
	if err := q.Limit(200).Scan(&rows).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rows, "Patients fetched")
}

// SetPsychologyEligibility grants/revokes a patient's access to psychology
// consultations — a treating doctor (referral) or an admin only.
//
// POST /appointments/patients/:id/psychology-eligibility   body: { eligible }
func (h *Handler) SetPsychologyEligibility(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	if !isDoctorRole(role) && !isAdminRole(role) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Only a doctor or admin can grant psychology eligibility")
	}
	pid, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || pid == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid patient ID")
	}
	var body struct {
		Eligible bool `json:"eligible"`
	}
	_ = c.BodyParser(&body)
	if err := database.DB.WithContext(c.UserContext()).Table("user").
		Where("id = ?", pid).Update("psychology_eligible", body.Eligible).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"patient_id": pid, "psychology_eligible": body.Eligible}, "Eligibility updated")
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	apptGroup := router.Group("/appointments")

	// Protected routes
	protected := apptGroup.Group("/", middleware.Protected())
	protected.Post("/", h.CreateAppointment)
	protected.Get("/", h.GetUpcomingAppointments)
	protected.Get("/upcoming", h.GetUpcomingAppointments)
	protected.Get("/past", h.GetPastAppointments)
	protected.Get("/instant/pending", h.GetInstantPending)
	protected.Post("/patients/:id/psychology-eligibility", h.SetPsychologyEligibility)
	protected.Get("/doctor/my-patients", h.GetMyPatients)
	protected.Get("/doctor/:id/upcoming", h.GetUpcomingAppointmentsForDoctor)
	protected.Get("/:id", h.GetAppointment)
	protected.Put("/:id", h.UpdateAppointment)
	protected.Delete("/:id", h.DeleteAppointment)
	protected.Post("/:id/accept", h.AcceptAppointment)
	protected.Post("/:id/decline", h.DeclineAppointment)
	protected.Post("/:id/cancel", h.CancelAppointment)
	protected.Post("/:id/reschedule", h.RescheduleAppointment)
	protected.Post("/:id/start", h.StartAppointment)
	protected.Post("/:id/complete", h.CompleteAppointment)
	protected.Post("/:id/no-show", h.MarkNoShow)
	protected.Post("/:id/notes", h.AddAppointmentNotes)
	protected.Get("/:id/history", h.GetAppointmentHistory)

	// SOS Extra updates
	protected.Post("/:id/en-route", h.MarkDoctorEnRoute)
	protected.Post("/:id/arrived", h.MarkDoctorArrived)

	// Doctor-specific appointment routes (protected — doctor identity is sensitive)
}
