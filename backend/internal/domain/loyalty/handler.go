package loyalty

import (
	"github.com/gofiber/fiber/v2"
	"drayya-core/internal/middleware"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) RegisterRoutes(router fiber.Router) {
	rewards := router.Group("/rewards", middleware.Protected())

	rewards.Get("/balance", h.getBalance)
	rewards.Get("/ledger", h.getLedger)
	rewards.Post("/redeem", h.redeem)
	rewards.Get("/referral-code", h.getReferralCode)
	rewards.Post("/apply-referral", h.applyReferral)

	// Admin
	admin := router.Group("/admin/rewards", middleware.Protected(), middleware.RequireRoles("admin", "super_admin"))
	admin.Get("/rules", h.listRules)
	admin.Put("/rules", h.upsertRule)
}

func (h *Handler) getBalance(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	balance, err := GetBalance(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"points":      balance,
		"value_inr":   PointsValueRupees(balance),
	})
}

func (h *Handler) getLedger(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	entries, err := GetLedger(userID, 50)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(entries)
}

type RedeemRequest struct {
	Points      int64  `json:"points"`
	Description string `json:"description"`
}

func (h *Handler) redeem(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var req RedeemRequest
	if err := c.BodyParser(&req); err != nil || req.Points <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "points must be positive"})
	}
	if req.Description == "" {
		req.Description = "Points redeemed"
	}
	entry, err := RedeemPoints(userID, req.Points, req.Description)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"redeemed":    req.Points,
		"value_inr":   PointsValueRupees(req.Points),
		"new_balance": entry.Balance,
	})
}

func (h *Handler) getReferralCode(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	rc, err := GetOrCreateReferralCode(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rc)
}

func (h *Handler) applyReferral(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	var req struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&req); err != nil || req.Code == "" {
		return c.Status(400).JSON(fiber.Map{"error": "code is required"})
	}
	if err := ApplyReferralCode(userID, req.Code); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Referral applied successfully", "welcome_points": refereeWelcomePoints})
}

// Admin
func (h *Handler) listRules(c *fiber.Ctx) error {
	rules, err := GetAllRewardRules()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rules)
}

func (h *Handler) upsertRule(c *fiber.Ctx) error {
	var rule RewardRule
	if err := c.BodyParser(&rule); err != nil || rule.RefType == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ref_type is required"})
	}
	if err := UpsertRewardRule(&rule); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rule)
}
