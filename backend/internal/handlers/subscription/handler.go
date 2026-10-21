package subscription

import (
	"drayya-core/internal/services/subscription"

	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

func GetPlans(c *fiber.Ctx) error {
	return response.JSON(c, fiber.StatusOK, subscription.GetPlans(), "Subscription plans fetched")
}

func Upgrade(c *fiber.Ctx) error {
	type Request struct {
		Tier string `json:"tier"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	userID := c.Locals("user_id").(uint)

	user, err := subscription.UpgradeSubscriptionWithPayment(userID, req.Tier)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPGRADE_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, user, "Subscription upgraded successfully")
}
func Cancel(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	if err := subscription.CancelSubscription(userID); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CANCEL_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Auto-renewal cancelled. Your benefits remain active until the end of the billing period.")
}

func GetMySubscription(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)

	status, err := subscription.GetSubscriptionStatus(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, status, "Subscription status fetched")
}

func RegisterSubscriptionRoutes(router fiber.Router) {
	subGroup := router.Group("/subscription")
	subGroup.Get("/plans", GetPlans)

	protected := subGroup.Group("/", middleware.Protected())
	protected.Post("/upgrade", Upgrade)
	protected.Post("/cancel", Cancel)
	protected.Get("/mine", GetMySubscription)
}
