package chat

import (
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/middleware"
	"drayya-core/internal/services/chat"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// CreateConsultationRoom handler
func CreateConsultationRoom(c *fiber.Ctx) error {
	type Request struct {
		AppointmentID uint `json:"appointment_id"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	// Fetch appointment to get doctor/patient IDs
	var appt appointment.Appointment
	if err := database.DB.First(&appt, req.AppointmentID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "APPOINTMENT_NOT_FOUND", "Appointment not found")
	}

	// Note: In real app, check permissions (current user must be doctor or patient)
	// userID := c.Locals("user_id").(uint)

	if appt.DoctorID == nil {
		return response.Error(c, fiber.StatusBadRequest, "NO_DOCTOR_ASSIGNED", "Appointment has no assigned doctor")
	}

	room, err := chat.GetOrCreateConsultationRoom(req.AppointmentID, *appt.DoctorID, appt.PatientID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ROOM_CREATION_FAILED", "Failed to create room")
	}

	return response.JSON(c, fiber.StatusOK, room, "Fetched")
}

// CreateDirectChat handler
func CreateDirectChat(c *fiber.Ctx) error {
	type Request struct {
		TargetUserID uint `json:"target_user_id"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	currentUserID := c.Locals("user_id").(uint)

	room, err := chat.GetOrCreateDirectRoom(currentUserID, req.TargetUserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "DIRECT_CHAT_FAILED", "Failed to create direct chat")
	}

	return response.JSON(c, fiber.StatusOK, room, "Fetched")
}

// GetUserRooms handler — returns rooms with unread message count per room.
func GetUserRooms(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	rooms, err := chat.GetUserRooms(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_ROOMS_FAILED", "Failed to fetch rooms")
	}

	db := database.DB
	type roomWithMeta struct {
		communication.ChatRoom
		UnreadCount int64                     `json:"unread_count"`
		LastMessage *communication.ChatMessage `json:"last_message,omitempty"`
	}

	result := make([]roomWithMeta, 0, len(rooms))
	for _, r := range rooms {
		meta := roomWithMeta{ChatRoom: r}
		// Unread = messages in the room sent by someone else that aren't marked read.
		// Using sender_id != userID instead of receiver_id = userID because receiver_id
		// can be 0 when it wasn't set correctly (e.g., dispatch-assigned appointments).
		var count int64
		db.Model(&communication.ChatMessage{}).
			Where("room_id = ? AND sender_id != ? AND is_read = ?", r.ID, userID, false).
			Count(&count)
		meta.UnreadCount = count

		var lastMsg communication.ChatMessage
		if err := db.Where("room_id = ?", r.ID).Order("created_at desc").First(&lastMsg).Error; err == nil {
			meta.LastMessage = &lastMsg
		}
		result = append(result, meta)
	}

	return response.JSON(c, fiber.StatusOK, result, "Fetched")
}

// GetRoomMessages handler
func GetRoomMessages(c *fiber.Ctx) error {
	roomID, _ := strconv.Atoi(c.Params("room_id"))
	limit := 50
	offset := 0

	if l := c.Query("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}
	if o := c.Query("skip"); o != "" { // Python used 'skip'
		offset, _ = strconv.Atoi(o)
	}

	msgs, err := chat.GetRoomMessages(uint(roomID), limit, offset)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_MESSAGES_FAILED", "Failed to fetch messages")
	}

	// Reverse messages? Python `reversed(messages)` suggests frontend expects logic order?
	// Python: `.order_by(desc).offset.limit` then `reversed()`.
	// This means API returns oldest first (chronological) even though DB fetch is newest first (pagination).
	// Let's replicate this or frontend might break.

	// Quick reverse in Go
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"room_id":  roomID,
		"messages": msgs,
	}, "Fetched")
}

// CloseRoom handler
func CloseRoom(c *fiber.Ctx) error {
	roomID, _ := strconv.Atoi(c.Params("room_id"))
	userID := c.Locals("user_id").(uint)

	if err := chat.CloseRoom(uint(roomID), userID); err != nil {
		return response.Error(c, fiber.StatusForbidden, "CLOSE_ROOM_FORBIDDEN", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Room closed")
}

// UpdateRoom handler
func UpdateRoom(c *fiber.Ctx) error {
	roomID, _ := strconv.Atoi(c.Params("room_id"))

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request body")
	}

	if err := chat.UpdateRoom(uint(roomID), updates); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Update failed")
	}

	return response.JSON(c, fiber.StatusOK, nil, "Room updated")
}

// DeleteMessage handler
func DeleteMessage(c *fiber.Ctx) error {
	msgID, _ := strconv.Atoi(c.Params("id"))
	userID := c.Locals("user_id").(uint)

	if err := chat.DeleteMessage(uint(msgID), userID); err != nil {
		return response.Error(c, fiber.StatusForbidden, "DELETE_MESSAGE_FORBIDDEN", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Message deleted")
}

// NotifyTyping handler
func NotifyTyping(c *fiber.Ctx) error {
	roomID, _ := strconv.Atoi(c.Params("room_id"))
	userID := c.Locals("user_id").(uint)

	// In a real app, you'd publish this to Redis for the WS Hub
	fmt.Printf("User %d is typing in room %d\n", userID, roomID)

	return c.SendStatus(200)
}

// GetRoomFiles handler
func GetRoomFiles(c *fiber.Ctx) error {
	roomID, _ := strconv.Atoi(c.Params("room_id"))

	var messages []communication.ChatMessage
	database.DB.Where("room_id = ? AND file_url IS NOT NULL", roomID).
		Order("created_at desc").
		Find(&messages)

	return response.JSON(c, fiber.StatusOK, messages, "Fetched")
}

// GetCommandCenterChats for Admin Command Center
func GetCommandCenterChats(c *fiber.Ctx) error {
	db := database.DB
	var rooms []communication.ChatRoom

	query := db.Preload("Doctor").Preload("Patient")

	if urgency := c.Query("urgency_filter"); urgency != "" {
		query = query.Where("urgency = ?", urgency)
	}
	if status := c.Query("status_filter"); status == "active" {
		query = query.Where("is_active = ?", true)
	}

	query.Order("updated_at desc").Find(&rooms)

	// Silent session: "no last message" is the normal state for empty rooms.
	// Without this, every command-center page load logs N "record not found"
	// errors (one per chat room).
	silentDB := db.Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})

	var resp []fiber.Map
	for _, room := range rooms {
		// Get last message
		var lastMsg communication.ChatMessage
		silentDB.Where("room_id = ?", room.ID).Order("created_at desc").First(&lastMsg)

		participants := []fiber.Map{}
		if room.Doctor != nil {
			participants = append(participants, fiber.Map{
				"id":   room.Doctor.ID,
				"name": room.Doctor.FullName,
				"role": "doctor",
			})
		}
		if room.Patient != nil {
			participants = append(participants, fiber.Map{
				"id":   room.Patient.ID,
				"name": room.Patient.FullName,
				"role": "patient",
			})
		}

		var lastMessageMap interface{} = nil
		if lastMsg.ID != 0 {
			lastMessageMap = fiber.Map{
				"message": lastMsg.Message,
			}
		}

		resp = append(resp, fiber.Map{
			"room_id":      room.ID,
			"chat_type":    room.ChatType,
			"urgency":      room.Urgency,
			"updated_at":   room.UpdatedAt,
			"participants": participants,
			"last_message": lastMessageMap,
		})
	}

	// Make sure it returns an empty array instead of null
	if len(resp) == 0 {
		return response.JSON(c, fiber.StatusOK, []fiber.Map{}, "Fetched")
	}

	return response.JSON(c, fiber.StatusOK, resp, "Fetched")
}

func RegisterChatRoutes(router fiber.Router) {
	chatGroup := router.Group("/chat", middleware.Protected())
	chatGroup.Post("/rooms", CreateConsultationRoom)
	chatGroup.Put("/rooms/:room_id", UpdateRoom)
	chatGroup.Post("/rooms/direct", CreateDirectChat)
	chatGroup.Get("/rooms", GetUserRooms)
	chatGroup.Get("/rooms/:room_id/messages", GetRoomMessages)
	chatGroup.Get("/rooms/:room_id/files", GetRoomFiles)
	chatGroup.Post("/rooms/:room_id/typing", NotifyTyping)
	chatGroup.Post("/rooms/:room_id/close", CloseRoom)
	chatGroup.Delete("/messages/:id", DeleteMessage)
	chatGroup.Get("/command-center/chats", GetCommandCenterChats)
}
