package middleware

import (
	"drayya-core/internal/database"

	"github.com/gofiber/fiber/v2"
)

// RequirePrime blocks access unless the user has an active paid subscription.
// Use after Protected() so user_id is already in Locals.
func RequirePrime() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(uint)
		if !ok || userID == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "authentication required",
			})
		}

		var tier string
		if err := database.DB.Table("user").
			Select("tier").
			Where("id = ?", userID).
			Scan(&tier).Error; err != nil || tier == "" || tier == "free" {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"error":       "Prime subscription required",
				"upgrade_url": "/membership",
				"tier":        "free",
			})
		}

		return c.Next()
	}
}

// RequireSubscriptionTier blocks access unless the user has at least the given tier.
// Tier hierarchy: free < prime < enterprise
func RequireSubscriptionTier(minTier string) fiber.Handler {
	tierRank := map[string]int{
		"free":       0,
		"prime":      1,
		"enterprise": 2,
	}
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(uint)
		if !ok || userID == 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
		}

		var tier string
		database.DB.Table("user").Select("tier").Where("id = ?", userID).Scan(&tier)
		if tier == "" {
			tier = "free"
		}

		if tierRank[tier] < tierRank[minTier] {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"error":        "Subscription upgrade required",
				"current_tier": tier,
				"required_tier": minTier,
				"upgrade_url":  "/membership",
			})
		}

		c.Locals("subscription_tier", tier)
		return c.Next()
	}
}
