package communication

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/career"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handler struct {
	svc Service
	db  *gorm.DB
}

type ChatRoomRequest struct {
	DoctorID      uint   `json:"doctor_id"`
	PatientID     uint   `json:"patient_id"`
	ChatType      string `json:"chat_type"`
	AppointmentID uint   `json:"appointment_id"`
}

type SendMessageRequest struct {
	RoomID      uint    `json:"room_id"`
	SenderID    uint    `json:"sender_id"`
	ReceiverID  uint    `json:"receiver_id"`
	Message     string  `json:"message"`
	MessageType string  `json:"message_type"`
	FileURL     *string `json:"file_url,omitempty"`
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc, db: database.DB}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	commGroup := router.Group("/communication", middleware.Protected())

	// Chat rooms
	commGroup.Post("/chat/rooms", h.GetOrCreateChatRoom)
	commGroup.Get("/chat/rooms", h.GetUserRooms)
	commGroup.Get("/chat/rooms/:roomID", h.GetRoom)
	commGroup.Post("/chat/rooms/:roomID/read", h.MarkRoomRead)
	commGroup.Get("/chat/rooms/:roomID/search", h.SearchMessages)

	// Chat messages
	commGroup.Post("/chat/messages", h.SendMessage)
	commGroup.Delete("/chat/messages/:id", h.DeleteMessage)
	commGroup.Get("/chat/history/:roomID", h.GetChatHistory)
	commGroup.Get("/chat/rooms/:roomID/messages", h.GetChatHistory) // Alias

	// Active consultations
	commGroup.Post("/chat/rooms/:roomID/messages", h.SendMessage) // Alias
	commGroup.Post("/chat/rooms/direct", h.GetOrCreateChatRoom)   // Alias

	// Active consultations
	commGroup.Get("/chat/consultations", h.GetActiveConsultations)

	// Notifications
	commGroup.Get("/notifications", h.GetNotifications)
	commGroup.Put("/notifications/:id/read", h.MarkNotificationRead)

	// Enhanced Notifications (Admin/Config)
	router.Get("/notifications-enhanced/templates", func(c *fiber.Ctx) error {
		var templates []career.RecruitmentTemplate
		if err := h.db.Find(&templates).Error; err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "DB_ERROR", err.Error())
		}
		
		var mapped []fiber.Map
		for _, t := range templates {
			channel := "email"
			if t.Type == "WhatsApp" {
				channel = "whatsapp"
			} else if t.Type == "SMS" {
				channel = "sms"
			}
			mapped = append(mapped, fiber.Map{
				"id":                t.ID,
				"template_type":     t.Title,
				"template_name":     t.Title,
				"name":              t.Title,
				"notification_type": t.Title,
				"channel":           channel,
				"subject":           t.Title,
				"message_content":   t.Content,
				"body_template":     t.Content,
				"variables":         t.Variables,
				"language":          "en",
				"is_active":         true,
				"created_at":        t.CreatedAt,
				"updated_at":        t.UpdatedAt,
			})
		}
		if mapped == nil {
			mapped = []fiber.Map{}
		}
		return response.JSON(c, fiber.StatusOK, mapped, "Templates fetched")
	})
	
	router.Get("/notifications-enhanced/templates/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var t career.RecruitmentTemplate
		if err := h.db.First(&t, id).Error; err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Template not found")
		}
		
		channel := "email"
		if t.Type == "WhatsApp" {
			channel = "whatsapp"
		} else if t.Type == "SMS" {
			channel = "sms"
		}
		mapped := fiber.Map{
			"id":                t.ID,
			"template_type":     t.Title,
			"template_name":     t.Title,
			"name":              t.Title,
			"notification_type": t.Title,
			"channel":           channel,
			"subject":           t.Title,
			"message_content":   t.Content,
			"body_template":     t.Content,
			"variables":         t.Variables,
			"language":          "en",
			"is_active":         true,
			"created_at":        t.CreatedAt,
			"updated_at":        t.UpdatedAt,
		}
		return response.JSON(c, fiber.StatusOK, mapped, "Template fetched")
	})

	router.Post("/notifications-enhanced/templates", func(c *fiber.Ctx) error {
		var req struct {
			Name             string `json:"name" query:"name"`
			TemplateName     string `json:"template_name" query:"template_name"`
			NotificationType string `json:"notification_type" query:"notification_type"`
			Channel          string `json:"channel" query:"channel"`
			Subject          string `json:"subject" query:"subject"`
			BodyTemplate     string `json:"body_template" query:"body_template"`
			MessageContent   string `json:"message_content" query:"message_content"`
			Variables        string `json:"variables" query:"variables"`
		}

		if err := c.QueryParser(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_QUERY", err.Error())
		}
		if req.Name == "" && req.BodyTemplate == "" {
			if err := c.BodyParser(&req); err != nil {
				return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
			}
		}

		title := req.Name
		if title == "" {
			title = req.TemplateName
		}
		if title == "" {
			title = req.NotificationType
		}
		if title == "" {
			title = req.Subject
		}

		content := req.BodyTemplate
		if content == "" {
			content = req.MessageContent
		}

		channel := req.Channel
		if strings.ToLower(channel) == "whatsapp" {
			channel = "WhatsApp"
		} else if strings.ToLower(channel) == "email" {
			channel = "Email"
		} else if strings.ToLower(channel) == "sms" {
			channel = "SMS"
		} else if channel == "" {
			channel = "Email"
		}

		t := career.RecruitmentTemplate{
			Role:      "All Users",
			Type:      channel,
			Title:     title,
			Content:   content,
			Variables: req.Variables,
		}

		if err := h.db.Create(&t).Error; err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "DB_ERROR", err.Error())
		}

		return response.JSON(c, fiber.StatusCreated, fiber.Map{
			"id":                t.ID,
			"template_type":     t.Title,
			"template_name":     t.Title,
			"name":              t.Title,
			"notification_type": t.Title,
			"channel":           strings.ToLower(t.Type),
			"subject":           t.Title,
			"message_content":   t.Content,
			"body_template":     t.Content,
			"variables":         t.Variables,
			"language":          "en",
			"is_active":         true,
			"created_at":        t.CreatedAt,
			"updated_at":        t.UpdatedAt,
		}, "Template created successfully")
	})

	router.Put("/notifications-enhanced/templates/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var t career.RecruitmentTemplate
		if err := h.db.First(&t, id).Error; err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Template not found")
		}

		var req struct {
			Name             string `json:"name" query:"name"`
			TemplateName     string `json:"template_name" query:"template_name"`
			NotificationType string `json:"notification_type" query:"notification_type"`
			Channel          string `json:"channel" query:"channel"`
			Subject          string `json:"subject" query:"subject"`
			BodyTemplate     string `json:"body_template" query:"body_template"`
			MessageContent   string `json:"message_content" query:"message_content"`
			Variables        string `json:"variables" query:"variables"`
		}

		if err := c.QueryParser(&req); err != nil {
			return response.Error(c, fiber.StatusBadRequest, "INVALID_QUERY", err.Error())
		}
		if req.Name == "" && req.BodyTemplate == "" {
			if err := c.BodyParser(&req); err != nil {
				return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
			}
		}

		title := req.Name
		if title == "" {
			title = req.TemplateName
		}
		if title == "" {
			title = req.NotificationType
		}
		if title == "" {
			title = req.Subject
		}
		if title != "" {
			t.Title = title
		}

		content := req.BodyTemplate
		if content == "" {
			content = req.MessageContent
		}
		if content != "" {
			t.Content = content
		}

		if req.Channel != "" {
			channel := req.Channel
			if strings.ToLower(channel) == "whatsapp" {
				channel = "WhatsApp"
			} else if strings.ToLower(channel) == "email" {
				channel = "Email"
			} else if strings.ToLower(channel) == "sms" {
				channel = "SMS"
			}
			t.Type = channel
		}

		if req.Variables != "" {
			t.Variables = req.Variables
		}

		if err := h.db.Save(&t).Error; err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "DB_ERROR", err.Error())
		}

		return response.JSON(c, fiber.StatusOK, fiber.Map{
			"id":                t.ID,
			"template_type":     t.Title,
			"template_name":     t.Title,
			"name":              t.Title,
			"notification_type": t.Title,
			"channel":           strings.ToLower(t.Type),
			"subject":           t.Title,
			"message_content":   t.Content,
			"body_template":     t.Content,
			"variables":         t.Variables,
			"language":          "en",
			"is_active":         true,
			"created_at":        t.CreatedAt,
			"updated_at":        t.UpdatedAt,
		}, "Template updated successfully")
	})

	router.Delete("/notifications-enhanced/templates/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var t career.RecruitmentTemplate
		if err := h.db.First(&t, id).Error; err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Template not found")
		}

		if err := h.db.Delete(&t).Error; err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "DB_ERROR", err.Error())
		}

		return response.JSON(c, fiber.StatusOK, nil, "Template deleted successfully")
	})

	router.Get("/notifications-enhanced/logs", func(c *fiber.Ctx) error {
		var logs []NotificationLog
		
		q := h.db.Preload("User").Order("sent_at desc")
		
		if limit := c.Query("limit"); limit != "" {
			if l, err := strconv.Atoi(limit); err == nil && l > 0 {
				q = q.Limit(l)
			}
		} else {
			q = q.Limit(50)
		}
		
		if channel := c.Query("channel"); channel != "" {
			q = q.Where("notification_type = ?", channel)
		}
		
		if status := c.Query("status"); status != "" {
			q = q.Where("status = ?", status)
		}
		
		if err := q.Find(&logs).Error; err != nil {
			return response.Error(c, fiber.StatusInternalServerError, "DB_ERROR", err.Error())
		}
		
		var mapped []fiber.Map
		for _, l := range logs {
			userName := ""
			if l.User != nil {
				userName = l.User.FullName
			}
			
			mapped = append(mapped, fiber.Map{
				"id":                l.ID,
				"user_id":           l.UserID,
				"user_name":         userName,
				"notification_type": l.TemplateType,
				"channel":           l.NotificationType,
				"status":            l.Status,
				"error_message":     l.ErrorMessage,
				"sent_at":           l.SentAt,
				"created_at":        l.SentAt,
			})
		}
		if mapped == nil {
			mapped = []fiber.Map{}
		}
		return response.JSON(c, fiber.StatusOK, mapped, "Logs fetched")
	})
}

// GetOrCreateChatRoom returns an existing room or creates a new one for a doctor-patient pair
// @Summary Get or Create Chat Room
// @Description Fetch an existing chat room or initialize a new one between a doctor and a patient
// @Tags Communication
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body ChatRoomRequest true "Room Request"
// @Success 200 {object} response.StandardResponse{data=map[string]interface{}} "Room fetched"
// @Router /api/v1/communication/chat/rooms [post]
func (h *Handler) GetOrCreateChatRoom(c *fiber.Ctx) error {
	var req ChatRoomRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Support appointment_id lookup: derive doctor/patient from the appointment
	if req.AppointmentID != 0 && (req.DoctorID == 0 || req.PatientID == 0) {
		// Use a minimal struct to avoid import cycle with the appointment package
		var appt struct {
			DoctorID  *uint `gorm:"column:doctor_id"`
			PatientID uint  `gorm:"column:patient_id"`
		}
		if err := h.db.Table("appointments").Select("doctor_id, patient_id").First(&appt, req.AppointmentID).Error; err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Appointment not found")
		}
		if appt.DoctorID == nil {
			return response.Error(c, fiber.StatusBadRequest, "NO_DOCTOR", "Appointment has no assigned doctor yet")
		}
		req.DoctorID = *appt.DoctorID
		req.PatientID = appt.PatientID
		if req.ChatType == "" {
			req.ChatType = "consultation"
		}
	}

	if req.DoctorID == 0 || req.PatientID == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "doctor_id and patient_id (or appointment_id) required")
	}

	// Verify caller is a participant in this room
	callerID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	if role != "admin" && role != "super_admin" && callerID != req.DoctorID && callerID != req.PatientID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You are not a participant in this consultation")
	}

	room, err := h.svc.GetOrCreateChatRoom(c.Context(), req.DoctorID, req.PatientID, req.ChatType)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ROOM_ERROR", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, room, "Chat room fetched/created")
}

// SendMessage records a new message in a chat room
// @Summary Send Chat Message
// @Description Send a text or file message to a specific chat room
// @Tags Communication
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body SendMessageRequest true "Message Request"
// @Success 201 {object} response.StandardResponse{data=map[string]interface{}} "Message sent"
// @Router /api/v1/communication/chat/messages [post]
func (h *Handler) SendMessage(c *fiber.Ctx) error {
	var req SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Try to get roomID from URL if not in body
	if req.RoomID == 0 {
		roomIDStr := c.Params("roomID")
		if roomIDStr != "" {
			rid, _ := strconv.ParseUint(roomIDStr, 10, 32)
			req.RoomID = uint(rid)
		}
	}

	// Force SenderID from token — never trust client-supplied value (prevents impersonation)
	senderID, _ := c.Locals("user_id").(uint)
	if senderID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	req.SenderID = senderID

	msg, err := h.svc.SendMessage(c.Context(), req.RoomID, req.SenderID, req.ReceiverID, req.Message, req.MessageType, req.FileURL)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SEND_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, msg, "Message sent successfully")
}

// GetChatHistory returns messages for a specific room
// @Summary Get Chat History
// @Description Fetch past messages for a chat room with pagination
// @Tags Communication
// @Produce json
// @Security ApiKeyAuth
// @Param roomID path string true "Room ID"
// @Param limit query int false "Number of messages (default 50)"
// @Success 200 {object} response.StandardResponse{data=[]map[string]interface{}} "History fetched"
// @Router /api/v1/communication/chat/history/{roomID} [get]
func (h *Handler) GetChatHistory(c *fiber.Ctx) error {
	roomIDStr := c.Params("roomID")
	roomID, _ := strconv.ParseUint(roomIDStr, 10, 32)
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	// Membership check: caller must be the doctor or patient in this room
	callerID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	if role != "admin" && role != "super_admin" {
		var room ChatRoom
		if err := h.db.First(&room, roomID).Error; err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Chat room not found")
		}
		if room.DoctorID != callerID && room.PatientID != callerID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You are not a participant in this chat room")
		}
	}

	msgs, err := h.svc.GetChatHistory(c.Context(), uint(roomID), limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, msgs, "Chat history fetched")
}

// GetActiveConsultations returns all active chat rooms for a doctor
// @Summary Get Active Consultations
// @Description Fetch all ongoing consultation chat rooms for the current doctor
// @Tags Communication
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse{data=[]map[string]interface{}} "Consultations fetched"
// @Router /api/v1/communication/chat/consultations [get]
func (h *Handler) GetActiveConsultations(c *fiber.Ctx) error {
	doctorID, ok := c.Locals("user_id").(uint)
	if !ok || doctorID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	rooms, err := h.svc.GetActiveConsultations(c.Context(), doctorID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, rooms, "Active consultations fetched")
}

// GetUserRooms returns all chat rooms for the authenticated user, with unread counts.
func (h *Handler) GetUserRooms(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	rooms, err := h.svc.GetRoomsForUser(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, rooms, "Rooms fetched")
}

// GetRoom returns details of a single chat room.
func (h *Handler) GetRoom(c *fiber.Ctx) error {
	roomID, _ := strconv.ParseUint(c.Params("roomID"), 10, 32)
	callerID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	room, err := h.svc.GetRoomByID(c.Context(), uint(roomID))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Chat room not found")
	}
	if role != "admin" && role != "super_admin" && room.DoctorID != callerID && room.PatientID != callerID {
		return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You are not a participant in this chat room")
	}
	return response.JSON(c, fiber.StatusOK, room, "Room fetched")
}

// MarkRoomRead marks all messages in a room as read for the authenticated user.
func (h *Handler) MarkRoomRead(c *fiber.Ctx) error {
	roomID, _ := strconv.ParseUint(c.Params("roomID"), 10, 32)
	userID, _ := c.Locals("user_id").(uint)
	if userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	if err := h.svc.MarkMessagesRead(c.Context(), uint(roomID), userID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Messages marked as read")
}

// SearchMessages searches messages in a room by keyword.
func (h *Handler) SearchMessages(c *fiber.Ctx) error {
	roomID, _ := strconv.ParseUint(c.Params("roomID"), 10, 32)
	callerID, _ := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	query := c.Query("q")
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if role != "admin" && role != "super_admin" {
		var room ChatRoom
		if err := h.db.First(&room, roomID).Error; err != nil {
			return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Chat room not found")
		}
		if room.DoctorID != callerID && room.PatientID != callerID {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", "You are not a participant in this chat room")
		}
	}

	msgs, err := h.svc.SearchMessages(c.Context(), uint(roomID), query, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SEARCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, msgs, "Search results")
}

// DeleteMessage soft-deletes a message (only sender can delete their own).
func (h *Handler) DeleteMessage(c *fiber.Ctx) error {
	msgID, _ := strconv.ParseUint(c.Params("id"), 10, 32)
	callerID, _ := c.Locals("user_id").(uint)
	if callerID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	if err := h.svc.DeleteMessage(c.Context(), uint(msgID), callerID); err != nil {
		if err.Error() == "forbidden: you can only delete your own messages" {
			return response.Error(c, fiber.StatusForbidden, "FORBIDDEN", err.Error())
		}
		return response.Error(c, fiber.StatusInternalServerError, "DELETE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Message deleted")
}

// GetNotifications returns notifications for the authenticated user
// @Summary Get User Notifications
// @Description Fetch all notifications (read and unread) for the current user
// @Tags Communication
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} response.StandardResponse{data=[]map[string]interface{}} "Notifications fetched"
// @Router /api/v1/communication/notifications [get]
func (h *Handler) GetNotifications(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	ns, err := h.svc.GetUserNotifications(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, ns, "Notifications fetched")
}

// MarkNotificationRead marks a notification as read
// @Summary Mark Notification as Read
// @Description Update the status of a specific notification to 'read'
// @Tags Communication
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Notification ID"
// @Success 200 {object} response.StandardResponse "Updated"
// @Router /api/v1/communication/notifications/{id}/read [put]
func (h *Handler) MarkNotificationRead(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)

	if err := h.svc.MarkNotificationRead(c.Context(), uint(id)); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Notification marked as read")
}
