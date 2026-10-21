package finance

import (
	"context"
	"log"
	"strings"

	"drayya-core/internal/tokens"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

// MountWebSocket attaches the live finance stream WebSocket at /ws/finance.
//
// Connection flow (mirrors the audit stream):
//  1. Client opens wss://<host>/ws/finance?token=<jwt>
//  2. Upgrade handler validates the JWT and requires an admin/super_admin role.
//  3. On success the client receives {type, event} JSON frames for every ledger
//     transaction the moment it commits.
//
// Auth via query-param token is the standard browser-WebSocket pattern (they
// cannot set Authorization headers on the upgrade request).
func MountWebSocket(app *fiber.App) {
	app.Use("/ws/finance", func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return c.Status(fiber.StatusUpgradeRequired).SendString("WebSocket upgrade required")
		}
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
		// Financial data is highly sensitive — admins only.
		if claims.Role != "admin" && claims.Role != "super_admin" {
			return c.Status(fiber.StatusForbidden).SendString("Admin role required")
		}
		c.Locals("allowed", true)
		c.Locals("admin_id", claims.UserID)
		return c.Next()
	})

	app.Get("/ws/finance", websocket.New(func(conn *websocket.Conn) {
		defer conn.Close()
		adminID, _ := conn.Locals("admin_id").(string)
		log.Printf("[finance:stream] admin %s connected (subs=%d)", adminID, SubscriberCount()+1)

		ctx, cancel := context.WithCancel(context.Background())
		// Detect client disconnect: any read error → done.
		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					cancel()
					return
				}
			}
		}()

		LiveStreamSession(ctx, func(b []byte) error {
			return conn.WriteMessage(websocket.TextMessage, b)
		})
		log.Printf("[finance:stream] admin %s disconnected (subs=%d)", adminID, SubscriberCount()-1)
	}))
}
