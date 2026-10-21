package handlers

import (
	franModel "drayya-core/internal/domain/franchise"
	franchiseSvc "drayya-core/internal/services/franchise"
	"strconv"

	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

func GetFranchiseAgreements(c *fiber.Ctx) error {
	status := c.Query("status")
	s := franchiseSvc.NewService()
	agreements, err := s.GetAgreements(status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, agreements, "Franchise agreements fetched")
}

func GetFranchisePerformance(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	s := franchiseSvc.NewService()
	performance, err := s.GetPerformance(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, performance, "Franchise performance fetched")
}

func WithdrawFranchiseEarnings(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))

	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}

	s := franchiseSvc.NewService()
	amountCents := int64(req.Amount * 100)
	if err := s.WithdrawFranchiseEarnings(uint(id), amountCents); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "WITHDRAW_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Withdrawal processed successfully")
}

func CreateFranchise(c *fiber.Ctx) error {
	var f franModel.Franchise
	if err := c.BodyParser(&f); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request")
	}

	s := franchiseSvc.NewService()
	res, err := s.CreateFranchise(f)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, res, "Franchise created successfully")
}

func GetEnterpriseAnalytics(c *fiber.Ctx) error {
	s := franchiseSvc.NewService()
	stats, err := s.GetEnterpriseAnalytics()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, stats, "Enterprise analytics fetched")
}

func RegisterFranchiseRoutes(router fiber.Router) {
	franGroup := router.Group("/franchise")
	franGroup.Get("/agreements", GetFranchiseAgreements)
	franGroup.Get("/performance/:id", GetFranchisePerformance)
	franGroup.Post("/withdraw/:id", middleware.Protected(), WithdrawFranchiseEarnings)

	// Admin Enterprise routes
	adminGroup := router.Group("/admin", middleware.Protected(), middleware.AdminOnly())
	adminGroup.Post("/franchise", CreateFranchise)
	adminGroup.Get("/enterprise/analytics", GetEnterpriseAnalytics)
}
