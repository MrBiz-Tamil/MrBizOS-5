package handlers

import (
	"drayya-core/internal/domain/referral"
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type ReferralHandler struct {
	service referral.Service
}

func NewReferralHandler(service referral.Service) *ReferralHandler {
	return &ReferralHandler{service: service}
}

func (h *ReferralHandler) GetMyReferralCode(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", "User ID not found")
	}

	code, err := h.service.GetMyCode(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch/generate referral code", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{"code": code}, "Referral code fetched")
}

func (h *ReferralHandler) GetMyReferrals(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", "User ID not found")
	}

	referrals, err := h.service.GetMyReferrals(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch referrals", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, referrals, "My referrals fetched")
}

func (h *ReferralHandler) GetReferralStats(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", "User ID not found")
	}

	stats, err := h.service.GetStats(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch referral stats", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, stats, "Referral stats fetched")
}

func (h *ReferralHandler) GenerateCode(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "Unauthorized", "User ID not found")
	}

	code, err := h.service.GenerateNewCode(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate referral code", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{"code": code}, "New referral code generated")
}

func (h *ReferralHandler) RegisterRoutes(router fiber.Router) {
	ref := router.Group("/referrals", middleware.Protected())
	ref.Get("/code", h.GetMyReferralCode)
	ref.Post("/code", h.GenerateCode)
	ref.Post("/generate", h.GenerateCode)
	ref.Get("/mine", h.GetMyReferrals)
	ref.Get("/stats", h.GetReferralStats)
}
