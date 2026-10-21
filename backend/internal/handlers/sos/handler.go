package sos

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/queue"
	chatSvc "drayya-core/internal/services/chat"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// publishRoomMsg sends a JSON message to a Redis room channel so the WS hub
// routes it to the correct connected client(s).
func publishRoomMsg(room string, payload map[string]interface{}) {
	if database.Rdb == nil {
		return
	}
	payload["room_id"] = room
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[SOS] publishRoomMsg marshal error: %v", err)
		return
	}
	database.Rdb.Publish(context.Background(), "chat:global", data)
}

type TriggerRequest struct {
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	ConsultationMode string  `json:"consultation_mode"`
}

// TriggerSOS creates a new emergency appointment
// @Summary Trigger Emergency SOS
// @Description Create a high-priority emergency appointment with location data
// @Tags SOS
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body TriggerRequest true "SOS Trigger Request"
// @Success 201 {object} response.StandardResponse{data=appointment.Appointment} "SOS triggered successfully"
// @Failure 400 {object} response.StandardError "Invalid Request"
// @Router /api/v1/sos/trigger [post]
func TriggerSOS(c *fiber.Ctx) error {
	var req TriggerRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	// Extract patient ID from context. The route is guarded by Protected() +
	// PatientOnly(), so user_id must be present — refuse rather than default to
	// user #1 if a future middleware change ever removes those guards.
	patientID, ok := c.Locals("user_id").(uint)
	if !ok || patientID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// 1. Check if patient is blocked in DB
	var block identity.SOSBlock
	if err := database.DB.Where("patient_id = ? AND expires_at > ?", patientID, time.Now()).First(&block).Error; err == nil {
		return response.Error(c, fiber.StatusForbidden, "SOS_BLOCKED", fmt.Sprintf("Access denied: %s", block.Reason))
	}

	appt := appointment.Appointment{
		PatientID:        patientID,
		StartTime:        time.Now(),
		EndTime:          time.Now().Add(30 * time.Minute),
		Status:           appointment.StatusPending,
		AppointmentType:  "emergency",
		ConsultationMode: req.ConsultationMode,
		IsPriority:       true,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		Notes:            "SOS Triggered",
	}

	if err := database.DB.Create(&appt).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", "Failed to create SOS appointment")
	}

	// Enqueue to dispatch stream so the autopilot assigns the nearest available doctor
	queue.EnqueueStream(context.Background(), queue.AppointmentMetadata{
		AppointmentID:   int(appt.ID),
		PatientID:       int(patientID),
		AppointmentType: "emergency",
		AISeverityScore: 1.0, // highest severity for SOS
	})

	// Broadcast to all online doctors immediately so they see the alert without waiting for dispatch
	publishRoomMsg("room:doctors:instant", map[string]interface{}{
		"type": "instant_call_request",
		"metadata": map[string]interface{}{
			"appointment_id":    appt.ID,
			"consultation_mode": req.ConsultationMode,
			"type":              "emergency",
			"patient_id":        patientID,
		},
	})

	return response.JSON(c, fiber.StatusCreated, appt, "Emergency SOS triggered")
}

// GetPendingSOSRequests returns all pending emergency appointments
// @Summary Get Pending SOS Requests
// @Description Fetch all emergency appointments with 'pending' status for staff/dispatchers
// @Tags SOS
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Pending requests fetched"
// @Router /api/v1/sos/pending [get]
func GetPendingSOSRequests(c *fiber.Ctx) error {
	var appointments []appointment.Appointment

	// Fetch appointments where type is 'emergency' and status is 'pending'
	if err := database.DB.Where("appointment_type = ? AND status = ?", "emergency", appointment.StatusPending).
		Find(&appointments).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch pending requests")
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{"requests": appointments}, "Pending SOS requests fetched")
}

// AcceptSOSRequest allows a doctor to accept an SOS request
// @Summary Accept SOS Request
// @Description Transition an emergency appointment from pending to confirmed by an assigned doctor
// @Tags SOS
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Appointment ID"
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "SOS accepted"
// @Router /api/v1/sos/{id}/accept [post]
func AcceptSOSRequest(c *fiber.Ctx) error {
	id := c.Params("id")

	// Get accepting doctor's ID from JWT token
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Doctor ID not found in token")
	}

	var appt appointment.Appointment
	if err := database.DB.First(&appt, id).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
	}

	if appt.Status != appointment.StatusPending {
		// If already assigned to THIS doctor (e.g. by autopilot), treat as success to avoid 400 error race condition
		if appt.Status == appointment.StatusConfirmed && appt.DoctorID != nil && *appt.DoctorID == doctorID {
			return response.JSON(c, fiber.StatusOK, fiber.Map{
				"appointment_id":    appt.ID,
				"consultation_mode": appt.ConsultationMode,
				"patient_id":        appt.PatientID,
				"message":           "Already assigned via autopilot",
			}, "Appointment already assigned to you")
		}
		return response.Error(c, fiber.StatusBadRequest, "INVALID_STATUS", "Appointment is not pending or already taken by another doctor")
	}

	// Use column-level Updates (not Save) to avoid GORM upsert-ing the zero-value
	// Patient/Doctor association structs which causes a constraint violation → 500.
	if err := database.DB.Model(&appt).Updates(map[string]interface{}{
		"status":    appointment.StatusConfirmed,
		"doctor_id": doctorID,
	}).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to accept appointment")
	}

	// Pre-create chat room so both parties can join immediately after accept
	if appt.ConsultationMode == "chat" {
		if _, err := chatSvc.GetOrCreateConsultationRoom(appt.ID, doctorID, appt.PatientID); err != nil {
			log.Printf("[SOS] failed to create chat room for appointment %d: %v", appt.ID, err)
			// Non-fatal — frontend will retry via createOrGetRoom
		}
	}

	// Notify patient via WebSocket so their emergency page can redirect immediately
	publishRoomMsg(fmt.Sprintf("room:user:%d", appt.PatientID), map[string]interface{}{
		"type": "appointment_confirmed",
		"metadata": map[string]interface{}{
			"appointment_id":    appt.ID,
			"consultation_mode": appt.ConsultationMode,
			"doctor_id":         doctorID,
		},
	})

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"appointment_id":    appt.ID,
		"consultation_mode": appt.ConsultationMode,
		"patient_id":        appt.PatientID,
	}, "Appointment accepted successfully")
}

// GetRecentSOSRequests returns recent SOS alerts for the history widget
// @Summary Get Recent SOS History
// @Description Fetch last 5 emergency appointments for historical tracking
// @Tags SOS
// @Produce json
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "SOS history fetched"
// @Router /api/v1/sos/recent-requests [get]
func GetRecentSOSRequests(c *fiber.Ctx) error {
	var appointments []appointment.Appointment

	// Fetch last 5 emergency appointments, regardless of status
	if err := database.DB.Where("appointment_type = ?", "emergency").
		Order("created_at DESC").
		Limit(5).
		Find(&appointments).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch SOS history")
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{"requests": appointments}, "SOS history fetched")
}

// sosHistoryItem is the response shape used by the doctor dashboard history widget.
type sosHistoryItem struct {
	ID               uint      `json:"id"`
	PatientName      string    `json:"patient_name"`
	PatientID        uint      `json:"patient_id"`
	Status           string    `json:"status"` // accepted | missed | pending
	CreatedAt        time.Time `json:"created_at"`
	ConsultationMode string    `json:"consultation_mode"`
}

// patientNames fetches full names for a list of user IDs, returning a map[userID]name.
func patientNames(ids []uint) map[uint]string {
	m := make(map[uint]string, len(ids))
	if len(ids) == 0 || database.DB == nil {
		return m
	}
	var users []identity.User
	database.DB.Select("id, full_name").Where("id IN ?", ids).Find(&users)
	for _, u := range users {
		m[u.ID] = u.FullName
	}
	return m
}

// GetSOSHistory returns the last 20 emergency appointments with patient names and
// missed/accepted counts — used by SOSHistoryWidget.tsx at GET /sos/history.
func GetSOSHistory(c *fiber.Ctx) error {
	var appts []appointment.Appointment
	if err := database.DB.Where("appointment_type = ?", "emergency").
		Order("created_at DESC").Limit(20).Find(&appts).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch SOS history")
	}

	ids := make([]uint, 0, len(appts))
	for _, a := range appts {
		ids = append(ids, a.PatientID)
	}
	names := patientNames(ids)

	items := make([]sosHistoryItem, 0, len(appts))
	missed, accepted := 0, 0
	for _, a := range appts {
		name := names[a.PatientID]
		if name == "" {
			name = fmt.Sprintf("Patient #%d", a.PatientID)
		}
		status := string(a.Status)
		if a.Status == appointment.StatusPending && time.Since(a.CreatedAt) > 5*time.Minute {
			status = "missed"
			missed++
		} else if a.DoctorID != nil {
			accepted++
		}
		items = append(items, sosHistoryItem{
			ID:               a.ID,
			PatientName:      name,
			PatientID:        a.PatientID,
			Status:           status,
			CreatedAt:        a.CreatedAt,
			ConsultationMode: a.ConsultationMode,
		})
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"requests":       items,
		"missed_count":   missed,
		"accepted_count": accepted,
	}, "SOS history fetched")
}

// GetSOSLogs is an admin-level view of all recent SOS events (last 50).
// GET /sos/logs
func GetSOSLogs(c *fiber.Ctx) error {
	var appts []appointment.Appointment
	if err := database.DB.Where("appointment_type = ?", "emergency").
		Order("created_at DESC").Limit(50).Find(&appts).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch SOS logs")
	}

	ids := make([]uint, 0, len(appts))
	for _, a := range appts {
		ids = append(ids, a.PatientID)
	}
	names := patientNames(ids)

	items := make([]sosHistoryItem, 0, len(appts))
	for _, a := range appts {
		name := names[a.PatientID]
		if name == "" {
			name = fmt.Sprintf("Patient #%d", a.PatientID)
		}
		status := string(a.Status)
		if a.Status == appointment.StatusPending && time.Since(a.CreatedAt) > 5*time.Minute {
			status = "missed"
		}
		items = append(items, sosHistoryItem{
			ID:               a.ID,
			PatientName:      name,
			PatientID:        a.PatientID,
			Status:           status,
			CreatedAt:        a.CreatedAt,
			ConsultationMode: a.ConsultationMode,
		})
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{"logs": items}, "SOS logs fetched")
}

// BlockPatient marks a patient as blocked from the SOS system for 60 days.
// POST /sos/block-patient/:id
func BlockPatient(c *fiber.Ctx) error {
	idParam := c.Params("id")
	patientID, _ := strconv.ParseUint(idParam, 10, 32)
	if patientID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_ID", "Patient ID required")
	}

	doctorID, _ := c.Locals("user_id").(uint)

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.BodyParser(&body)
	if body.Reason == "" {
		body.Reason = "Abuse of emergency system"
	}

	expiresAt := time.Now().Add(60 * 24 * time.Hour)

	// Persist in DB
	block := identity.SOSBlock{
		PatientID: uint(patientID),
		Reason:    body.Reason,
		BlockedBy: doctorID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
	if err := database.DB.Create(&block).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DB_ERROR", "Failed to persist block")
	}

	// Also store in Redis for fast lookup in high-traffic scenarios
	if database.Rdb != nil {
		key := fmt.Sprintf("sos:blocked:%d", patientID)
		ctx := context.Background()
		database.Rdb.Set(ctx, key, body.Reason, 60*24*time.Hour)
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"patient_id": patientID,
		"blocked":    true,
		"expires_at": expiresAt,
		"reason":     body.Reason,
	}, "Patient blocked from SOS system")
}

func RegisterSOSRoutes(router fiber.Router) {
	sosGroup := router.Group("/sos")
	sosGroup.Post("/trigger", middleware.Protected(), middleware.PatientOnly(), TriggerSOS)
	// PHI: emergency feeds leak patient names + coordinates. Restrict to staff
	// (admins, doctors, BPO, dispatchers) — never unauthenticated public access.
	sosGroup.Get("/recent-requests", middleware.Protected(), middleware.StaffOnly(), GetRecentSOSRequests)
	sosGroup.Get("/history", middleware.Protected(), middleware.StaffOnly(), GetSOSHistory)
	sosGroup.Get("/logs", middleware.Protected(), GetSOSLogs)
	sosGroup.Post("/block-patient/:id", middleware.Protected(), middleware.DoctorOnly(), BlockPatient)

	// Staff/Doctor only routes
	sosGroup.Get("/pending", middleware.Protected(), middleware.StaffOnly(), GetPendingSOSRequests)
	sosGroup.Post("/:id/accept", middleware.Protected(), middleware.DoctorOnly(), AcceptSOSRequest)
}
