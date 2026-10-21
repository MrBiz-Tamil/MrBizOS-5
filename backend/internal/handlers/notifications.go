package handlers

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// RegisterFCMToken stores or updates a user's FCM device push token.
// POST /notifications/register-token   body: { token: string }
func RegisterFCMToken(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	var req struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&req); err != nil || req.Token == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_TOKEN", "token is required")
	}

	if err := database.DB.Model(&identity.User{}).
		Where("id = ?", userID).
		Update("fcm_token", req.Token).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", "Failed to save FCM token")
	}

	return response.JSON(c, fiber.StatusOK, nil, "FCM token registered")
}

// RegisterNotificationRoutes wires notification endpoints under /notifications.
func RegisterNotificationRoutes(router fiber.Router) {
	notifGroup := router.Group("/notifications", middleware.Protected())
	notifGroup.Post("/register-token", RegisterFCMToken)
}
