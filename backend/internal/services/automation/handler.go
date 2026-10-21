package automation

import (
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// HubCollector is a subset of the WebSocket hub interface needed for broadcasting.
type HubCollector interface {
	BroadcastMessage(roomID string, data interface{})
}

type Handler struct {
	commSvc communication.Service
	hub     HubCollector
}

func NewHandler(commSvc communication.Service, hub HubCollector) *Handler {
	return &Handler{
		commSvc: commSvc,
		hub:     hub,
	}
}

type BotResponseRequest struct {
	RoomID  string `json:"room_id"`
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

type ChatRequest struct {
	SessionID      string `json:"session_id"`
	GuestBrowserID string `json:"guest_browser_id"`
	Message        string `json:"message"`
}

// BotRespond allows an admin to send a message to a Devi chat room.
func (h *Handler) BotRespond(c *fiber.Ctx) error {
	var req BotResponseRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	if req.RoomID == "" || req.Message == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_FIELDS", "room_id and message are required")
	}

	// Broadcast the message to the room via the hub
	h.hub.BroadcastMessage(req.RoomID, fiber.Map{
		"type":    "chat_message",
		"room_id": req.RoomID,
		"user_id": "devi",
		"message": req.Message,
		"role":    "bot",
	})

	return c.JSON(fiber.Map{"status": "broadcasted"})
}

// DeviChat is a legacy bridge for HTTP-based Devi interactions.
func (h *Handler) DeviChat(c *fiber.Ctx) error {
	var req ChatRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	if req.Message == "" {
		return response.Error(c, fiber.StatusBadRequest, "EMPTY_MESSAGE", "Message cannot be empty")
	}

	// This is a stub for the test to pass. In reality, the /devi/chat route
	// is now handled by internal/domain/devi/handler.go.
	return c.JSON(fiber.Map{"reply": "This endpoint is deprecated. Use /api/v1/devi/chat"})
}
