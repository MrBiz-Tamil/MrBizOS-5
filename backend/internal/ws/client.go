package ws

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"time"

	"drayya-core/internal/domain/devi"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/presence"

	"github.com/gofiber/contrib/websocket"
)


const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096 // 4KB
)

// Client represents a connected websocket client
type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
	UserID string
	Role   string
	Rooms  map[string]bool

	// Optional Devi context fields — set on WS upgrade for authenticated users
	// so family-member sessions are routed correctly without re-parsing each message.
	FamilyMemberID *uint  // non-nil when chatting on behalf of a dependent
	GuestBrowserID string // non-empty for unauthenticated callers
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		var msg WSMsg
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Invalid JSON: %v", err)
			continue
		}

		// Refresh presence on any activity
		presence.Heartbeat(context.Background(), c.UserID)


		switch msg.Type {
		case "join_room":
			c.handleJoinRoom(msg)

		case "text", "message", "image", "file":
			c.handleChatMessage(msg)

		case "read_receipt":
			c.handleReadReceipt(msg)

		case "typing_start", "typing_stop":
			// Broadcast typing indicators to the room only
			if msg.RoomID != "" {
				c.Hub.Broadcast <- message
			}

		case "courier_location_update":
			// GPS telemetry from courier app. Relay to the specific order room so the patient can track.
			if msg.RoomID != "" {
				// We also persist this to Redis or DB in a real scenario (debounced).
				// For now, simple relay.
				c.Hub.Broadcast <- message
			}

		case "pong":
			// Handled by PongHandler — nothing to do
			continue

		default:
			log.Printf("[WS] Unhandled message type %q from user %s in room %q", msg.Type, c.UserID, msg.RoomID)
		}
	}
}

// handleJoinRoom validates that the caller is a participant before subscribing them.
func (c *Client) handleJoinRoom(msg WSMsg) {
	if msg.RoomID == "" {
		return
	}

	// Virtual rooms (devi-*, support, doctor broadcast) — always allow
	if strings.HasPrefix(msg.RoomID, "devi-") ||
		msg.RoomID == "support" ||
		strings.HasPrefix(msg.RoomID, "room:user:") ||
		msg.RoomID == "room:doctors:instant" {
		c.Rooms[msg.RoomID] = true
		return
	}

	// Admins may join any room
	if c.Role == "admin" || c.Role == "super_admin" {
		c.Rooms[msg.RoomID] = true
		return
	}

	// Numeric room ID — verify membership via domain service
	rid, err := strconv.ParseUint(msg.RoomID, 10, 32)
	if err != nil {
		log.Printf("[WS] join_room: invalid room ID %q from user %s", msg.RoomID, c.UserID)
		return
	}

	if c.Hub.CommSvc != nil {
		room, rerr := c.Hub.CommSvc.GetRoomByID(context.Background(), uint(rid))
		if rerr != nil {
			log.Printf("[WS] join_room: room %d not found (user %s): %v", rid, c.UserID, rerr)
			c.sendError("room_not_found", "Chat room not found")
			return
		}
		callerID, _ := strconv.ParseUint(c.UserID, 10, 32)
		if room.DoctorID != uint(callerID) && room.PatientID != uint(callerID) {
			log.Printf("[WS] join_room: user %s is not a participant in room %d", c.UserID, rid)
			c.sendError("forbidden", "You are not a participant in this room")
			return
		}
	}

	c.Rooms[msg.RoomID] = true
}

// handleChatMessage persists the message and broadcasts it to the room.
func (c *Client) handleChatMessage(msg WSMsg) {
	if msg.RoomID == "" {
		return
	}

	senderID, _ := strconv.ParseUint(c.UserID, 10, 32)
	isDeviRoom := strings.HasPrefix(msg.RoomID, "devi-") || msg.RoomID == "support"

	// ── Virtual room (Devi chatbot) ───────────────────────────────────────────
	rid, err := strconv.ParseUint(msg.RoomID, 10, 32)
	if err != nil {
		// Non-numeric room ID → treat as Devi session (no DB row, no persistence)
		userMsg := communication.ChatMessage{
			RoomID:      0,
			SenderID:    uint(senderID),
			Message:     msg.Message,
			MessageType: msg.Type,
			CreatedAt:   time.Now(),
		}

		outBytes, _ := json.Marshal(map[string]interface{}{
			"type":       msg.Type,
			"room_id":    msg.RoomID,
			"message":    msg.Message,
			"sender_id":  c.UserID,
			"created_at": userMsg.CreatedAt,
			"id":         0,
		})
		c.Hub.Broadcast <- outBytes

		if isDeviRoom && c.Hub.AutomationSvc != nil {
			uc := c.buildUserContext()
			go func(roomID string, uc *devi.UserContext, inMsg communication.ChatMessage) {
				ctx := context.Background()
				reply := c.Hub.AutomationSvc.ProcessMessageWithContext(ctx, &inMsg, uc)
				c.Hub.BroadcastToRoom(roomID, map[string]interface{}{
					"type":        "text",
					"room_id":     roomID,
					"message":     reply.Message,
					"actions":     reply.Actions,
					"components":  reply.Components,
					"sender_id":   "0",
					"sender_name": "Devi",
					"created_at":  time.Now(),
					"id":          0,
				})
			}(msg.RoomID, uc, userMsg)
		}
		return
	}

	// ── Real consultation room (numeric ID) ───────────────────────────────────
	roomID := uint(rid)

	// Determine receiver: the other participant in the room
	receiverID := uint(0)
	if c.Hub.CommSvc != nil {
		if room, rerr := c.Hub.CommSvc.GetRoomByID(context.Background(), roomID); rerr == nil {
			if room.PatientID == uint(senderID) {
				receiverID = room.DoctorID
			} else {
				receiverID = room.PatientID
			}
		} else {
			log.Printf("[WS] room %d not found for message from user %s: %v", roomID, c.UserID, rerr)
			// Don't save the message if room doesn't exist
			c.sendError("room_not_found", "Chat room not found or has been closed")
			return
		}
	}
	if receiverID == 0 {
		log.Printf("[WS] receiverID is 0 for room %d sender %s — room may have no assigned doctor yet", roomID, c.UserID)
	}

	// Persist via domain service (so REST history + unread counts stay in sync)
	var fileURL *string
	if msg.FileURL != "" {
		fileURL = &msg.FileURL
	}

	ctx := context.Background()
	var savedID uint
	var savedAt time.Time

	if c.Hub.CommSvc != nil {
		savedMsg, serr := c.Hub.CommSvc.SendMessage(ctx, roomID, uint(senderID), receiverID, msg.Message, msg.Type, fileURL)
		if serr != nil {
			log.Printf("[WS] Error saving message to room %d: %v", roomID, serr)
		} else {
			savedID = savedMsg.ID
			savedAt = savedMsg.CreatedAt
		}
	} else {
		savedAt = time.Now()
	}

	outBytes, _ := json.Marshal(map[string]interface{}{
		"type":       msg.Type,
		"room_id":    msg.RoomID,
		"message":    msg.Message,
		"sender_id":  c.UserID,
		"created_at": savedAt,
		"id":         savedID,
	})
	c.Hub.Broadcast <- outBytes

	// Devi also responds in named numeric support rooms
	if isDeviRoom && c.Hub.AutomationSvc != nil && savedID > 0 {
		savedMsg := &communication.ChatMessage{
			ID:          savedID,
			RoomID:      roomID,
			SenderID:    uint(senderID),
			Message:     msg.Message,
			MessageType: msg.Type,
			CreatedAt:   savedAt,
		}
		uc := c.buildUserContext()
		go func(roomID string, uc *devi.UserContext, inMsg *communication.ChatMessage) {
			reply := c.Hub.AutomationSvc.ProcessMessageWithContext(ctx, inMsg, uc)
			c.Hub.BroadcastToRoom(roomID, map[string]interface{}{
				"type":        "text",
				"room_id":     roomID,
				"message":     reply.Message,
				"actions":     reply.Actions,
				"components":  reply.Components,
				"sender_id":   "0",
				"sender_name": "Devi",
				"created_at":  time.Now(),
				"id":          0,
			})
		}(msg.RoomID, uc, savedMsg)
	}
}

// handleReadReceipt marks messages as read and broadcasts the receipt to the room.
func (c *Client) handleReadReceipt(msg WSMsg) {
	if msg.RoomID == "" || c.Hub.CommSvc == nil {
		return
	}
	rid, err := strconv.ParseUint(msg.RoomID, 10, 32)
	if err != nil {
		return
	}
	userID, err := strconv.ParseUint(c.UserID, 10, 32)
	if err != nil {
		return
	}

	ctx := context.Background()
	if err := c.Hub.CommSvc.MarkMessagesRead(ctx, uint(rid), uint(userID)); err != nil {
		log.Printf("[WS] MarkMessagesRead error for room %d user %s: %v", rid, c.UserID, err)
		return
	}

	// Broadcast read receipt to the room so the other side can update UI ticks
	c.Hub.BroadcastToRoom(msg.RoomID, map[string]interface{}{
		"type":     string(TypeMessagesRead),
		"room_id":  msg.RoomID,
		"user_id":  c.UserID,
		"read_at":  time.Now(),
	})
}

// buildUserContext creates a Devi UserContext from the Client's identity fields.
// Called once per incoming message so the engine always gets the right participant type.
func (c *Client) buildUserContext() *devi.UserContext {
	var uid *uint
	if n, err := strconv.ParseUint(c.UserID, 10, 64); err == nil && n > 0 {
		u := uint(n)
		uid = &u
	}
	return devi.BuildUserContext(uid, c.FamilyMemberID, c.GuestBrowserID)
}

// sendError sends a structured error frame to this client only.
func (c *Client) sendError(code, detail string) {
	payload, _ := json.Marshal(map[string]interface{}{
		"type":   "error",
		"code":   code,
		"detail": detail,
	})
	select {
	case c.Send <- payload:
	default:
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Flush any queued messages in the same write frame
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
			// Refresh presence
			presence.Heartbeat(context.Background(), c.UserID)

		}
	}
}
