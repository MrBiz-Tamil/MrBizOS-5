package ws

import (
	"context"
	"encoding/json"
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/communication"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/presence"
	"drayya-core/internal/services/automation"
	"sync"
	"time"
)



// InitRedis moved to database package

// WebSocketMessage moved to types.go

type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Broadcast channel
	Broadcast chan []byte

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client

	// Bot Integration
	AutomationSvc *automation.Service

	// CommSvc is the communication domain service — used by the WS client
	// to persist messages and verify room membership. Set via SetCommSvc
	// after construction (it's created later than the Hub in app.go).
	CommSvc communication.Service

	mu sync.RWMutex
}

func NewHub(svc *automation.Service) *Hub {
	return &Hub{
		Broadcast:     make(chan []byte),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		clients:       make(map[*Client]bool),
		AutomationSvc: svc,
	}
}

// SetCommSvc injects the communication service after construction.
func (h *Hub) SetCommSvc(svc communication.Service) {
	h.CommSvc = svc
}

// BroadcastToRoom sends a message to all clients subscribed to the given roomID.
// It is a thin wrapper around BroadcastMessage for clarity.
func (h *Hub) BroadcastToRoom(roomID string, data interface{}) {
	h.BroadcastMessage(roomID, data)
}

func (h *Hub) Run() {
	// Subscribe to global Redis channel only if Redis is available
	if database.Rdb != nil {
		ctx := context.Background()
		pubsub := database.Rdb.Subscribe(ctx, "chat:global")
		ch := pubsub.Channel()

		go func() {
			for msg := range ch {
				h.Broadcast <- []byte(msg.Payload)
			}
		}()
		log.Println("    WebSocket Hub subscribed to Redis")
	} else {
		log.Println("       WebSocket Hub running without Redis (local mode only)")
	}

	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("     Client connected: User %s (Role: %s)", client.UserID, client.Role)
			// Track Presence
			presence.SetOnline(context.Background(), client.UserID, client.Role)
			if client.Role == "doctor" {
				database.DB.Model(&identity.DoctorProfile{}).
					Where("user_id = ?", client.UserID).
					Updates(map[string]interface{}{
						"is_online":      true,
						"current_status": "available",
						"last_ping":      time.Now(),
					})

				if presence.GetActiveSessions(context.Background(), client.UserID) == 1 && database.Rdb != nil {
					var user identity.User
					database.DB.Select("full_name").First(&user, client.UserID)
					jsonMsg, _ := json.Marshal(map[string]interface{}{
						"type":        "doctor_status_update",
						"sender_id":   client.UserID,
						"sender_name": user.FullName,
						"sender_role": "doctor",
						"timestamp":   time.Now().Format(time.RFC3339),
						"metadata": map[string]interface{}{
							"doctor_id": client.UserID,
							"status":    "online",
						},
					})
					database.Rdb.Publish(context.Background(), "chat:global", jsonMsg)
				}
			}


		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				// Remove Presence
				presence.SetOffline(context.Background(), client.UserID, client.Role)
				if client.Role == "doctor" && presence.GetActiveSessions(context.Background(), client.UserID) == 0 {
					database.DB.Model(&identity.DoctorProfile{}).
						Where("user_id = ?", client.UserID).
						Updates(map[string]interface{}{
							"is_online":      false,
							"current_status": "offline",
						})

					if database.Rdb != nil {
						var user identity.User
						database.DB.Select("full_name").First(&user, client.UserID)
						jsonMsg, _ := json.Marshal(map[string]interface{}{
							"type":        "doctor_status_update",
							"sender_id":   client.UserID,
							"sender_name": user.FullName,
							"sender_role": "doctor",
							"timestamp":   time.Now().Format(time.RFC3339),
							"metadata": map[string]interface{}{
								"doctor_id": client.UserID,
								"status":    "offline",
							},
						})
						database.Rdb.Publish(context.Background(), "chat:global", jsonMsg)
					}
				}
			}

			h.mu.Unlock()
			log.Printf("     Client disconnected: User %s", client.UserID)


		case message := <-h.Broadcast:
			// Unmarshal to check target room. If JSON fails, drop the message — do NOT
			// broadcast to all clients (that would leak messages across rooms).
			var msg WSMsg
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("[Hub] Dropped unparseable message: %v", err)
				continue
			}

			h.mu.Lock()
			for client := range h.clients {
				// Filter by room if specified
				if msg.RoomID != "" && !client.Rooms[msg.RoomID] {
					continue
				}

				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) BroadcastMessage(roomID string, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling broadcast message: %v", err)
		return
	}

	h.Broadcast <- payload
}

