package handlers

import (
	"strconv"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/events"
	"drayya-core/internal/services/automation"
	"drayya-core/internal/tokens"
	"drayya-core/internal/ws"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

var Hub *ws.Hub

// EventBus is the package-level event bus shared by all handlers in this package.
// Set via SetEventBus from server/app.go after the bus is initialized.
var EventBus *events.Bus

// SetEventBus injects the domain event bus into the handlers package.
func SetEventBus(bus *events.Bus) {
	EventBus = bus
}

func InitWebSocketHub(svc *automation.Service) {
	database.InitRedis()
	Hub = ws.NewHub(svc)
	go Hub.Run()
}

// SetHubCommSvc injects the communication service into the WebSocket hub.
// Must be called after both Hub and commSvc are constructed (in app.go).
func SetHubCommSvc(svc communication.Service) {
	if Hub != nil {
		Hub.SetCommSvc(svc)
	}
}

// WebSocketUpgrade handles the upgrade to WebSocket protocol
func WebSocketUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return c.Status(fiber.StatusUpgradeRequired).SendString("WebSocket upgrade required")
}

// WebSocketHandler handles the ongoing connection
func WebSocketHandler(c *websocket.Conn) {
	// Extract user info from query params or context (from middleware)
	token := c.Query("token")
	userID := "guest"
	role := "guest"

	if token != "" {
		claims, err := tokens.ValidateToken(token)
		if err == nil {
			userID = claims.UserID
			role = claims.Role
		}
	} else {
		// Fallback to direct query params for legacy support
		if qID := c.Query("user_id"); qID != "" {
			userID = qID
		}
		if qRole := c.Query("role"); qRole != "" {
			role = qRole
		}
	}

	// Optional Devi context — passed as query params by the frontend
	var familyMemberID *uint
	if fmStr := c.Query("family_member_id"); fmStr != "" {
		if fmN, err := strconv.ParseUint(fmStr, 10, 64); err == nil && fmN > 0 {
			n := uint(fmN)
			familyMemberID = &n
		}
	}
	guestBrowserID := c.Query("guest_browser_id") // UUID from localStorage (unauthenticated callers)

	client := &ws.Client{
		Hub:            Hub,
		Conn:           c,
		Send:           make(chan []byte, 256),
		UserID:         userID,
		Role:           role,
		Rooms:          make(map[string]bool),
		FamilyMemberID: familyMemberID,
		GuestBrowserID: guestBrowserID,
	}

	// Join personal room for targeted alerts
	client.Rooms["room:user:"+userID] = true

	// Doctors auto-join the instant-call broadcast room so they receive
	// instant_call_request events as soon as a patient creates an unassigned instant/SOS
	// appointment — before the dispatch worker has matched a specific doctor.
	if role == "doctor" || role == "chief_doctor" || role == "hospital_head_doctor" {
		client.Rooms["room:doctors:instant"] = true
	}

	// AUTO-JOIN: Support joining a specific room via query param (e.g., Devi Chat)
	if qRoom := c.Query("room_id"); qRoom != "" {
		client.Rooms[qRoom] = true
	} else if qRoomLegacy := c.Query("room"); qRoomLegacy != "" {
		client.Rooms[qRoomLegacy] = true
	}

	client.Hub.Register <- client

	// Start WritePump in a goroutine
	go client.WritePump()

	// ReadPump executes in the current goroutine (blocking until disconnect)
	client.ReadPump()
}
