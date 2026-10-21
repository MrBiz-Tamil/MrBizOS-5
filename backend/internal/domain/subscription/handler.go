package subscription

import (
	"encoding/json"
	"os"

	"github.com/gofiber/fiber/v2"

	"drayya-core/internal/middleware"
	"drayya-core/internal/services/razorpay"
)

func RegisterRoutes(router fiber.Router) {
	subscriptionGroup := router.Group("/subscriptions")
	subAlias := router.Group("/subscription") // Alias for frontend compatibility

	// Define routes on both groups
	for _, g := range []fiber.Router{subscriptionGroup, subAlias} {
		g.Get("/plans", getPlans)
		g.Post("/webhook", handleWebhookEndpoint)

		// Protected routes
		protected := g.Group("", middleware.Protected())
		protected.Post("/subscribe", createSubscriptionCheckout)
		protected.Get("/status", getSubscriptionStatus)
		protected.Get("/mine", getSubscriptionStatus) // frontend calls /mine
		protected.Post("/cancel", cancelSubscription)
		protected.Post("/pause", pauseSubscription)
		protected.Post("/resume", resumeSubscription)
		protected.Post("/upgrade", upgradeSubscription)
		protected.Get("/invoices", getInvoiceHistory)
		protected.Get("/history", getSubscriptionHistory)
	}

	// Admin routes
	admin := router.Group("/admin/subscriptions", middleware.Protected(), middleware.RequireRoles("admin", "super_admin"))
	admin.Get("/overview", adminOverview)
}

func getPlans(c *fiber.Ctx) error {
	plans, err := GetAllPlans()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch plans"})
	}
	return c.JSON(plans)
}

type SubscribeRequest struct {
	Tier string `json:"tier"`
}

func createSubscriptionCheckout(c *fiber.Ctx) error {
	var req SubscribeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	userID := c.Locals("user_id").(uint)

	// Check if user already has an active subscription
	activeSub, err := GetActiveSubscriptionForUser(userID)
	if err == nil && activeSub != nil && activeSub.Plan.Tier == req.Tier {
		return c.Status(400).JSON(fiber.Map{"error": "already subscribed to this tier"})
	}

	subID, err := CreateCheckout(userID, req.Tier)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"subscription_id": subID})
}

func getSubscriptionStatus(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	activeSub, err := GetActiveSubscriptionForUser(userID)
	if err != nil {
		return c.JSON(fiber.Map{"tier": "free", "status": "active"})
	}
	return c.JSON(activeSub)
}

func cancelSubscription(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	if err := CancelSubscription(userID); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Subscription cancelled successfully"})
}

func pauseSubscription(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	if err := PauseSubscription(userID); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Subscription paused"})
}

func resumeSubscription(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	if err := ResumeSubscription(userID); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Subscription resumed"})
}

func upgradeSubscription(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var req struct {
		Tier string `json:"tier"`
	}
	if err := c.BodyParser(&req); err != nil || req.Tier == "" {
		return c.Status(400).JSON(fiber.Map{"error": "tier is required"})
	}
	sub, err := UpgradePlan(userID, req.Tier)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(200).JSON(sub)
}

func getInvoiceHistory(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	invoices, err := GetInvoicesByUserID(userID, 20)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(invoices)
}

func getSubscriptionHistory(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	history, err := GetUserSubscriptionHistory(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(history)
}

func adminOverview(c *fiber.Ctx) error {
	overview, err := GetAdminSubscriptionOverview()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(overview)
}

func handleWebhookEndpoint(c *fiber.Ctx) error {
	// Verify signature
	signature := c.Get("X-Razorpay-Signature")
	secret := os.Getenv("RAZORPAY_WEBHOOK_SECRET")

	body := c.Body()

	if !razorpay.VerifyWebhookSignature(body, signature, secret) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid signature"})
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload format"})
	}

	event, ok := payload["event"].(string)
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": "missing event type"})
	}

	if err := HandleWebhook(payload, event); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}
