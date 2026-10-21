package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/livekit"
	"drayya-core/internal/middleware"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/utils/response"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// publishToRoom sends a JSON event to a Redis channel so the WS hub delivers it
// to every client subscribed to that room.
func publishToRoom(ctx context.Context, roomID string, payload map[string]interface{}) {
	if database.Rdb == nil {
		return
	}
	payload["room_id"] = roomID
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[LiveKit] publishToRoom marshal error: %v", err)
		return
	}
	database.Rdb.Publish(ctx, "chat:global", data)
}

func isLiveKitAdmin(role string) bool {
	return role == "admin" || role == "super_admin"
}

func isLiveKitDoctor(role string) bool {
	return role == "doctor" || role == "chief_doctor" || role == "hospital_head_doctor"
}

func currentLiveKitUser(c *fiber.Ctx) (uint, string) {
	userID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	return userID, role
}

func canAccessConsultation(c *fiber.Ctx, consult communication.VideoConsultation) bool {
	userID, role := currentLiveKitUser(c)
	return userID != 0 && (isLiveKitAdmin(role) || consult.DoctorID == userID || consult.PatientID == userID)
}

func canControlConsultation(c *fiber.Ctx, consult communication.VideoConsultation) bool {
	userID, role := currentLiveKitUser(c)
	return userID != 0 && (isLiveKitAdmin(role) || consult.DoctorID == userID || consult.PatientID == userID)
}

func canRecordConsultation(c *fiber.Ctx, consult communication.VideoConsultation) bool {
	userID, role := currentLiveKitUser(c)
	return userID != 0 && (isLiveKitAdmin(role) || (consult.DoctorID == userID && isLiveKitDoctor(role)))
}

func loadConsultationBySession(sessionID string) (*communication.VideoConsultation, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session_id required")
	}
	var consult communication.VideoConsultation
	if err := database.DB.Where("session_id = ?", sessionID).First(&consult).Error; err != nil {
		return nil, err
	}
	return &consult, nil
}

// ─── Token for any room (admin / generic) ─────────────────────────────────────

type TokenRequest struct {
	RoomName string `json:"room_name"`
	UserName string `json:"user_name"`
	Role     string `json:"role"` // doctor, patient, admin
}

// GetLiveKitToken generates an access token for a named room.
// POST /api/v1/livekit/token
func GetLiveKitToken(c *fiber.Ctx) error {
	var req TokenRequest
	if err := c.BodyParser(&req); err != nil || req.RoomName == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "room_name is required")
	}

	// Use authenticated user ID as identity (not a client-supplied header)
	userID := "guest"
	if uid, ok := c.Locals("user_id").(uint); ok && uid != 0 {
		userID = strconv.Itoa(int(uid))
	}
	callerID, callerRole := currentLiveKitUser(c)
	if strings.HasPrefix(req.RoomName, "appointment_") {
		apptID, err := strconv.Atoi(strings.TrimPrefix(req.RoomName, "appointment_"))
		if err != nil || apptID == 0 {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_ROOM", "Invalid appointment room")
		}
		var appt appointment.Appointment
		if err := database.DB.First(&appt, apptID).Error; err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
		}
		if !isLiveKitAdmin(callerRole) && appt.PatientID != callerID && (appt.DoctorID == nil || *appt.DoctorID != callerID) {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You are not a participant in this appointment")
		}
	} else if !isLiveKitAdmin(callerRole) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Admin access required for generic LiveKit rooms")
	}
	if req.UserName == "" {
		var user identity.User
		database.DB.Select("full_name").First(&user, callerID)
		req.UserName = user.FullName
		if req.UserName == "" {
			req.UserName = userID
		}
	}

	svc := livekit.GetService()
	if svc == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "LIVEKIT_UNAVAILABLE", "LiveKit service not configured")
	}
	token, err := svc.GenerateToken(req.RoomName, userID, req.UserName, req.Role)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TOKEN_FAILED", "Failed to generate token")
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"token":     token,
		"room_name": req.RoomName,
		"role":      req.Role,
		"room_url":  livekit.GetServerURL(),
	}, "Token generated")
}

// ─── Token by appointment (patient join flow) ─────────────────────────────────

// GetTokenForAppointment generates a patient or doctor token scoped to an appointment room.
// GET /api/v1/livekit/token/:appointmentId
func GetTokenForAppointment(c *fiber.Ctx) error {
	apptIDStr := c.Params("appointmentId")
	apptID, err := strconv.Atoi(apptIDStr)
	if err != nil || apptID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid appointment ID")
	}

	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var appt appointment.Appointment
	if err := database.DB.Preload("Patient").Preload("Doctor").First(&appt, apptID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}

	// Only allow joining when appointment is in a valid call state
	if appt.Status != appointment.StatusConfirmed &&
		appt.Status != appointment.StatusInConsult &&
		appt.Status != appointment.AppointmentStatus("in_progress") {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_STATUS",
			fmt.Sprintf("Cannot join appointment with status '%s': must be confirmed or in progress", appt.Status))
	}

	// Verify caller is a participant (patient or assigned doctor)
	isParticipant := appt.PatientID == userID || (appt.DoctorID != nil && *appt.DoctorID == userID)
	callerRole, _ := c.Locals("role").(string)
	if !isParticipant && callerRole != "admin" && callerRole != "super_admin" {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You are not a participant in this appointment")
	}

	// Determine token role
	role := "patient"
	if appt.DoctorID != nil && *appt.DoctorID == userID {
		role = "doctor"
	}

	// Fetch name
	var user identity.User
	database.DB.Select("full_name").First(&user, userID)
	name := user.FullName
	if name == "" {
		name = role
	}

	roomName := fmt.Sprintf("appointment_%d", apptID)
	svc := livekit.GetService()
	if svc == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "LIVEKIT_UNAVAILABLE", "LiveKit service not configured")
	}
	token, err := svc.GenerateToken(roomName, strconv.Itoa(int(userID)), name, role)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TOKEN_FAILED", "Failed to generate token")
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"token":     token,
		"room_name": roomName,
		"room_url":  livekit.GetServerURL(),
		"role":      role,
	}, "Token generated")
}

// GetSessionByAppointment looks up a VideoConsultation record by appointment ID.
// GET /api/v1/livekit/sessions/appointment/:appointmentId
func GetSessionByAppointment(c *fiber.Ctx) error {
	apptIDStr := c.Params("appointmentId")
	apptID, err := strconv.Atoi(apptIDStr)
	if err != nil || apptID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid appointment ID")
	}

	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var consult communication.VideoConsultation
	if err := database.DB.Where("appointment_id = ?", apptID).First(&consult).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "No session found for this appointment")
	}

	// Only participants may fetch session details
	if consult.DoctorID != userID && consult.PatientID != userID {
		callerRole, _ := c.Locals("role").(string)
		if callerRole != "admin" && callerRole != "super_admin" {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You are not a participant in this session")
		}
	}

	return response.JSON(c, fiber.StatusOK, consult, "Session fetched")
}

// ─── Start Call (doctor initiates) ────────────────────────────────────────────

// StartCall creates/joins a VideoConsultation record and returns a doctor token.
// POST /api/v1/livekit/call/start   body: {appointment_id, call_type}
func StartCall(c *fiber.Ctx) error {
	var req struct {
		AppointmentID uint   `json:"appointment_id"`
		CallType      string `json:"call_type"` // video | audio
	}
	if err := c.BodyParser(&req); err != nil || req.AppointmentID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "appointment_id required")
	}

	// Get doctor ID from JWT — not from body (security fix)
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var appt appointment.Appointment
	if err := database.DB.First(&appt, req.AppointmentID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}
	if appt.DoctorID == nil || *appt.DoctorID != doctorID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You are not the assigned doctor for this appointment")
	}
	if appt.Status != appointment.StatusConfirmed && appt.Status != appointment.StatusInConsult {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_STATUS", "Appointment must be confirmed before starting a call")
	}

	roomName := fmt.Sprintf("appointment_%d", req.AppointmentID)
	sessionID := fmt.Sprintf("sess_%d", time.Now().Unix())

	// Create or reuse VideoConsultation record
	var consult communication.VideoConsultation
	result := database.DB.Where("appointment_id = ?", req.AppointmentID).First(&consult)
	if result.Error != nil {
		consult = communication.VideoConsultation{
			SessionID:     sessionID,
			AppointmentID: req.AppointmentID,
			DoctorID:      doctorID,
			PatientID:     appt.PatientID,
			Status:        "in_progress",
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		database.DB.Create(&consult)
	} else {
		sessionID = consult.SessionID
	}

	// Advance appointment status
	if appt.Status != appointment.StatusInConsult {
		database.DB.Model(&appt).Update("status", string(appointment.StatusInConsult))
	}

	// Fetch doctor name
	var doctor identity.User
	database.DB.Select("full_name").First(&doctor, doctorID)
	docName := doctor.FullName
	if docName == "" {
		docName = "Doctor"
	}

	svc := livekit.GetService()
	if svc == nil {
		return response.JSON(c, fiber.StatusOK, fiber.Map{
			"session_id": sessionID,
			"room_name":  roomName,
			"room_url":   livekit.GetServerURL(),
			"token":      "",
			"message":    "Call started (LiveKit token unavailable)",
		}, "Call started")
	}
	// Notify patient — FCM push + WhatsApp + WebSocket
	go func() {
		var patientUser identity.User
		if database.DB.Select("fcm_token, phone_number").First(&patientUser, appt.PatientID).Error == nil {
			notification.NotifyCallStarted(patientUser.FCMToken, docName, req.AppointmentID)
			// Deep link for the patient to join the session
			roomLink := fmt.Sprintf("https://Dr.Ayya.in/consultation/%s", roomName)
			notification.WhatsAppCallStarted(patientUser.PhoneNumber, docName, roomLink)
		}
	}()

	// WebSocket notification so patient can auto-join without polling
	publishToRoom(c.Context(), fmt.Sprintf("room:user:%d", appt.PatientID), map[string]interface{}{
		"type": "call_started",
		"metadata": map[string]interface{}{
			"appointment_id": req.AppointmentID,
			"session_id":     sessionID,
			"room_name":      roomName,
			"doctor_name":    docName,
		},
	})

	token, _ := svc.GenerateToken(roomName, strconv.Itoa(int(doctorID)), docName, "doctor")
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"session_id": sessionID,
		"room_name":  roomName,
		"room_url":   livekit.GetServerURL(),
		"token":      token,
		"message":    "Call started",
	}, "Call started")
}

// ─── Join Call (patient or reconnect) ─────────────────────────────────────────

// JoinCall returns a token to join an existing session by session ID.
// POST /api/v1/livekit/call/join/:sessionId
func JoinCall(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	var consult communication.VideoConsultation
	if err := database.DB.Where("session_id = ?", sessionID).First(&consult).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Session not found")
	}
	if !canAccessConsultation(c, consult) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You are not a participant in this session")
	}

	role := "patient"
	if consult.DoctorID == userID {
		role = "doctor"
	}

	var user identity.User
	database.DB.Select("full_name").First(&user, userID)
	name := user.FullName
	if name == "" {
		name = role
	}

	roomName := fmt.Sprintf("appointment_%d", consult.AppointmentID)
	svc := livekit.GetService()
	if svc == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "LIVEKIT_UNAVAILABLE", "LiveKit service not configured")
	}
	token, err := svc.GenerateToken(roomName, strconv.Itoa(int(userID)), name, role)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TOKEN_FAILED", "Failed to generate token")
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"session_id": sessionID,
		"room_name":  roomName,
		"room_url":   livekit.GetServerURL(),
		"token":      token,
	}, "Joined call")
}

// ─── End Call ──────────────────────────────────────────────────────────────────

// EndCall marks a VideoConsultation as completed and records call quality.
// POST /api/v1/livekit/call/end   body: {session_id, call_quality}
func EndCall(c *fiber.Ctx) error {
	var req struct {
		SessionID   string `json:"session_id"`
		CallQuality string `json:"call_quality"`
	}
	if err := c.BodyParser(&req); err != nil || req.SessionID == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "session_id required")
	}

	var consult communication.VideoConsultation
	if err := database.DB.Where("session_id = ?", req.SessionID).First(&consult).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Session not found")
	}
	if !canControlConsultation(c, consult) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You are not allowed to end this session")
	}

	durationMins := int(time.Since(consult.CreatedAt).Minutes())
	database.DB.Model(&consult).Updates(map[string]interface{}{
		"status":     "completed",
		"updated_at": time.Now(),
	})

	// CompleteAppointment via service: validates status, updates DB, decrements session,
	// removes from queue, and triggers the payout split.
	apptIDStr := strconv.Itoa(int(consult.AppointmentID))
	apptRepo := appointment.NewDefaultRepository()
	financeRepo := finance.NewRepository(database.DB)
	financeSvc := finance.NewService(financeRepo)
	apptSvc := appointment.NewService(apptRepo, financeSvc)

	if err := apptSvc.CompleteAppointment(c.Context(), apptIDStr); err != nil {
		log.Printf("[EndCall] CompleteAppointment failed for appt %d: %v", consult.AppointmentID, err)
		// Fallback: direct status update so appointment is not left in_consultation forever
		database.DB.Model(&appointment.Appointment{}).
			Where("id = ?", consult.AppointmentID).
			Update("status", string(appointment.StatusCompleted))
	}

	// Notify both parties so their UIs can redirect to the summary/feedback page
	ctx := c.Context()
	endPayload := map[string]interface{}{
		"type": "call_ended",
		"metadata": map[string]interface{}{
			"appointment_id":   consult.AppointmentID,
			"session_id":       req.SessionID,
			"duration_minutes": durationMins,
		},
	}
	publishToRoom(ctx, fmt.Sprintf("room:user:%d", consult.PatientID), endPayload)
	publishToRoom(ctx, fmt.Sprintf("room:user:%d", consult.DoctorID), endPayload)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"session_id":       req.SessionID,
		"duration_minutes": durationMins,
		"message":          "Call ended",
	}, "Call ended successfully")
}

// ─── Recording ────────────────────────────────────────────────────────────────

func StartRecording(c *fiber.Ctx) error {
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	consult, err := loadConsultationBySession(req.SessionID)
	if err != nil {
		if req.SessionID == "" {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "session_id required")
		}
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Session not found")
	}
	if !canRecordConsultation(c, *consult) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Only the assigned doctor or admin can record this session")
	}
	svc := livekit.GetService()
	if svc == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "LIVEKIT_UNAVAILABLE", "LiveKit not configured")
	}
	recID, err := svc.StartRecording(req.SessionID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "RECORD_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"recording_id": recID}, "Recording started")
}

func StopRecording(c *fiber.Ctx) error {
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	consult, err := loadConsultationBySession(req.SessionID)
	if err != nil {
		if req.SessionID == "" {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "session_id required")
		}
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Session not found")
	}
	if !canRecordConsultation(c, *consult) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Only the assigned doctor or admin can stop recording this session")
	}
	svc := livekit.GetService()
	if svc == nil {
		return response.Error(c, fiber.StatusServiceUnavailable, "LIVEKIT_UNAVAILABLE", "LiveKit not configured")
	}
	_ = svc.StopRecording(req.SessionID)
	return response.JSON(c, fiber.StatusOK, fiber.Map{"recording_url": ""}, "Recording stopped")
}

// ─── History & Admin ──────────────────────────────────────────────────────────

func GetCallHistory(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	var consultations []communication.VideoConsultation
	database.DB.Where("doctor_id = ? OR patient_id = ?", userID, userID).
		Order("created_at DESC").Limit(limit).Find(&consultations)
	return response.JSON(c, fiber.StatusOK, consultations, "Call history fetched")
}

func GetActiveCalls(c *fiber.Ctx) error {
	userID, role := currentLiveKitUser(c)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	var consultations []communication.VideoConsultation
	query := database.DB.Where("status = ?", "in_progress")
	if !isLiveKitAdmin(role) {
		query = query.Where("doctor_id = ? OR patient_id = ?", userID, userID)
	}
	query.Find(&consultations)
	return response.JSON(c, fiber.StatusOK, consultations, "Active calls fetched")
}

func AdminGetActiveCalls(c *fiber.Ctx) error {
	_, role := currentLiveKitUser(c)
	if !isLiveKitAdmin(role) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Admin access required")
	}
	var consultations []communication.VideoConsultation
	database.DB.Where("status = ?", "in_progress").Find(&consultations)
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"total_active_calls": len(consultations),
		"calls":              consultations,
	}, "Active calls fetched")
}

func AdminGetCallStats(c *fiber.Ctx) error {
	_, role := currentLiveKitUser(c)
	if !isLiveKitAdmin(role) {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "Admin access required")
	}
	days, _ := strconv.Atoi(c.Query("days", "7"))
	if days <= 0 {
		days = 7
	}
	since := time.Now().AddDate(0, 0, -days)

	// Basic consultation counts
	var total, completed, failed, active int64
	database.DB.Model(&communication.VideoConsultation{}).Where("created_at >= ?", since).Count(&total)
	database.DB.Model(&communication.VideoConsultation{}).Where("created_at >= ? AND status = ?", since, "completed").Count(&completed)
	database.DB.Model(&communication.VideoConsultation{}).Where("created_at >= ? AND status = ?", since, "failed").Count(&failed)
	database.DB.Model(&communication.VideoConsultation{}).Where("status = ?", "in_progress").Count(&active)

	// Completion rate as percentage (frontend expects a number 0–100)
	completionRate := 0.0
	if total > 0 {
		completionRate = (float64(completed) / float64(total)) * 100.0
	}

	// Average duration in minutes — pull from call_quality_reports joined on appointment_id
	type avgRow struct {
		AvgSeconds float64
	}
	var dur avgRow
	database.DB.Raw(`
		SELECT COALESCE(AVG(cqr.call_duration_sec), 0) AS avg_seconds
		FROM call_quality_reports cqr
		JOIN video_consultations vc ON vc.appointment_id = cqr.appointment_id
		WHERE vc.created_at >= ? AND cqr.call_duration_sec > 0
	`, since).Scan(&dur)
	avgDurationMinutes := dur.AvgSeconds / 60.0

	// Quality breakdown — count reports by grade, scoped to consultations in window
	type gradeRow struct {
		QualityGrade string
		N            int64
	}
	var gradeRows []gradeRow
	database.DB.Raw(`
		SELECT cqr.quality_grade, COUNT(*) AS n
		FROM call_quality_reports cqr
		JOIN video_consultations vc ON vc.appointment_id = cqr.appointment_id
		WHERE vc.created_at >= ?
		GROUP BY cqr.quality_grade
	`, since).Scan(&gradeRows)
	quality := map[string]int64{"excellent": 0, "good": 0, "fair": 0, "poor": 0}
	for _, r := range gradeRows {
		// Coalesce "bad" into "poor" — frontend only has 4 buckets
		key := r.QualityGrade
		if key == "bad" {
			key = "poor"
		}
		if _, ok := quality[key]; ok {
			quality[key] += r.N
		}
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"period_days":              days,
		"total_calls":              total,
		"completed_calls":          completed,
		"failed_calls":             failed,
		"active_calls":             active,
		"completion_rate":          completionRate,
		"average_duration_minutes": avgDurationMinutes,
		"quality_breakdown":        quality,
	}, "Call stats fetched")
}

// ─── Routes ───────────────────────────────────────────────────────────────────

func RegisterLiveKitRoutes(router fiber.Router) {
	lkGroup := router.Group("/livekit", middleware.Protected())

	// Token endpoints
	lkGroup.Post("/token", GetLiveKitToken)
	lkGroup.Get("/token/:appointmentId", GetTokenForAppointment)

	// Call lifecycle
	lkGroup.Post("/call/start", StartCall)
	lkGroup.Post("/call/join/:sessionId", JoinCall)
	lkGroup.Post("/call/end", EndCall)

	// Session lookup
	lkGroup.Get("/sessions/appointment/:appointmentId", GetSessionByAppointment)

	// Recording
	lkGroup.Post("/recording/start", StartRecording)
	lkGroup.Post("/recording/stop", StopRecording)

	// History
	lkGroup.Get("/calls/history", GetCallHistory)
	lkGroup.Get("/calls/active", GetActiveCalls)

	// Admin
	lkGroup.Get("/admin/active-calls", AdminGetActiveCalls)
	lkGroup.Get("/admin/call-stats", AdminGetCallStats)

	// /video/* aliases — redirect legacy frontend calls to the correct livekit handlers
	videoGroup := router.Group("/video", middleware.Protected())
	videoGroup.Post("/sessions", func(c *fiber.Ctx) error {
		// createSession(appointmentId) → StartCall
		return StartCall(c)
	})
	videoGroup.Get("/sessions/appointment/:appointmentId", GetSessionByAppointment)
	videoGroup.Post("/sessions/:sessionId/end", func(c *fiber.Ctx) error {
		// Rewrite body to include session_id from path param, then call EndCall
		sessionID := c.Params("sessionId")
		body := map[string]interface{}{"session_id": sessionID}
		if cq := c.Query("call_quality"); cq != "" {
			body["call_quality"] = cq
		}
		c.Request().SetBody([]byte(`{"session_id":"` + sessionID + `"}`))
		return EndCall(c)
	})
	videoGroup.Get("/history", GetCallHistory)
}
