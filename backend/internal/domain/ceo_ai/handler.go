package ceo_ai

import (
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r fiber.Router) {
	ai := r.Group("/ceo-ai")
	ai.Post("/query", h.QueryExecutiveAssistant)
}

func (h *Handler) QueryExecutiveAssistant(c *fiber.Ctx) error {
	var req ExecutiveQueryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	if req.UserID == 0 {
		if uid, ok := c.Locals("user_id").(uint); ok {
			req.UserID = uid
		} else {
			req.UserID = 1
		}
	}

	resp, err := h.svc.QueryExecutiveAssistant(&req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "AI_QUERY_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, resp, "Executive query processed successfully")
}
