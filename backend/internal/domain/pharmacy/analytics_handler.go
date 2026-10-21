package pharmacy

import (
	"drayya-core/internal/middleware"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type AnalyticsHandler struct {
	svc AnalyticsService
}

func NewAnalyticsHandler(db *gorm.DB) *AnalyticsHandler {
	repo := NewAnalyticsRepository(db)
	svc := NewAnalyticsService(repo)
	return &AnalyticsHandler{svc: svc}
}

func (h *AnalyticsHandler) TrackEvent(c *fiber.Ctx) error {
	var req ProductInteraction
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
	}

	// M-12: validate event_type before delegating to service
	if !validEventTypes[req.EventType] {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_EVENT_TYPE",
			"event_type must be one of: view, click, add_to_cart, remove_from_cart, purchase")
	}

	// Optional user ID
	userID, ok := c.Locals("user_id").(uint)
	if ok && userID != 0 {
		req.UserID = &userID
	}

	if err := h.svc.TrackEvent(c.Context(), &req); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "TRACKING_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Event tracked")
}

func (h *AnalyticsHandler) GetDashboard(c *fiber.Ctx) error {
	data, err := h.svc.GetDashboardData(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, data, "Dashboard data fetched")
}

func (h *AnalyticsHandler) GetAbandonedCarts(c *fiber.Ctx) error {
	carts, err := h.svc.GetAbandonedCarts(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, carts, "Abandoned carts fetched")
}

func (h *AnalyticsHandler) RecoverCart(c *fiber.Ctx) error {
	sessionID := c.Params("id")
	if err := h.svc.RecoverCart(c.Context(), sessionID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "RECOVERY_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, nil, "Recovery email initiated")
}

func RegisterAnalyticsRoutes(router fiber.Router, db *gorm.DB) {
	handler := NewAnalyticsHandler(db)

	analyticsGroup := router.Group("/analytics")
	// Allow public access to event tracking so guest interactions can be captured
	analyticsGroup.Post("/event", handler.TrackEvent)

	// Protected manager endpoints
	managerGroup := analyticsGroup.Group("/dashboard", middleware.Protected(), middleware.PharmacyManagerOnly())
	managerGroup.Get("/", handler.GetDashboard)
	managerGroup.Get("/abandoned-carts", handler.GetAbandonedCarts)
	managerGroup.Post("/abandoned-carts/:id/recover", handler.RecoverCart)
}
