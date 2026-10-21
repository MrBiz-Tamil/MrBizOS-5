package audit

import (
	"context"
	"log"
	"strings"

	"drayya-core/internal/tokens"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// MountWebSocket attaches the live audit stream WebSocket at /ws/audit.
//
// Connection flow:
//   1. Client opens wss://api.medihub.mrbiz.in/ws/audit?token=<jwt>
//   2. Upgrade handler validates the JWT and ensures the role is admin/super_admin.
//   3. On success, the client receives a stream of {type, event} JSON frames.
//
// Authentication via query-param token is the standard pattern for browser
// WebSockets — they cannot set Authorization headers on the upgrade request.
func MountWebSocket(app *fiber.App) {
	app.Use("/ws/audit", func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return c.Status(fiber.StatusUpgradeRequired).SendString("WebSocket upgrade required")
		}
		// Authenticate at upgrade time so unauthenticated sockets can't even
		// open. The token may be in `Authorization: Bearer X` or `?token=X`.
		token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).SendString("Missing token")
		}
		claims, err := tokens.ValidateToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Invalid token")
		}
		// Only admins can subscribe — audit data is highly sensitive
		if claims.Role != "admin" && claims.Role != "super_admin" {
			return c.Status(fiber.StatusForbidden).SendString("Admin role required")
		}
		c.Locals("allowed", true)
		c.Locals("admin_id", claims.UserID)
		return c.Next()
	})

	app.Get("/ws/audit", websocket.New(func(conn *websocket.Conn) {
		defer conn.Close()
		adminID, _ := conn.Locals("admin_id").(string)
		log.Printf("[audit:stream] admin %s connected (subs=%d)", adminID, SubscriberCount()+1)

		ctx, cancel := context.WithCancel(context.Background())
		// Detect client disconnect by trying to read — any read error → done.
		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					cancel()
					return
				}
			}
		}()

		// Stream events until ctx is cancelled (client disconnects).
		LiveStreamSession(ctx, func(b []byte) error {
			return conn.WriteMessage(websocket.TextMessage, b)
		})
		log.Printf("[audit:stream] admin %s disconnected (subs=%d)", adminID, SubscriberCount()-1)
	}))
}
