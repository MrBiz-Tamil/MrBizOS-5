package journey

import (
	"strconv"

	"drayya-core/internal/middleware"
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
	journeys := r.Group("/journeys")
	journeys.Get("/templates", h.ListTemplates)
	journeys.Post("/templates", h.CreateTemplate)
	journeys.Post("/instances", h.StartJourney)
	journeys.Get("/instances", h.ListInstances)
	journeys.Get("/instances/:id", h.GetInstance)
	journeys.Post("/instances/:id/transition", h.TransitionStage)
}

func (h *Handler) ListTemplates(c *fiber.Ctx) error {
	buID := middleware.GetBusinessUnitID(c)
	templates, err := h.svc.ListTemplates(buID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, templates, "Journey templates fetched successfully")
}

func (h *Handler) CreateTemplate(c *fiber.Ctx) error {
	var tmpl JourneyTemplate
	if err := c.BodyParser(&tmpl); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}
	tmpl.BusinessUnitID = middleware.GetBusinessUnitID(c)
	tmpl.OrganizationID = middleware.GetOrganizationID(c)

	created, err := h.svc.CreateTemplate(&tmpl)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, created, "Journey template created successfully")
}

func (h *Handler) StartJourney(c *fiber.Ctx) error {
	var req struct {
		TemplateID uint  `json:"template_id"`
		CustomerID uint  `json:"customer_id"`
		LeadID     *uint `json:"lead_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	buID := middleware.GetBusinessUnitID(c)
	orgID := middleware.GetOrganizationID(c)

	instance, err := h.svc.StartJourney(req.TemplateID, req.CustomerID, req.LeadID, buID, orgID)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "START_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, instance, "Journey started successfully")
}

func (h *Handler) ListInstances(c *fiber.Ctx) error {
	buID := middleware.GetBusinessUnitID(c)
	var custID uint
	if cidStr := c.Query("customer_id"); cidStr != "" {
		if id, err := strconv.ParseUint(cidStr, 10, 32); err == nil {
			custID = uint(id)
		}
	}

	instances, err := h.svc.ListInstances(buID, custID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, instances, "Journey instances fetched successfully")
}

func (h *Handler) GetInstance(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Instance ID must be an integer")
	}

	instance, err := h.svc.GetInstance(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Journey instance not found")
	}
	return response.JSON(c, fiber.StatusOK, instance, "Journey instance fetched successfully")
}

func (h *Handler) TransitionStage(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Instance ID must be an integer")
	}

	var req struct {
		NextStageID uint   `json:"next_stage_id"`
		Notes       string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	actorID := uint(1)
	if uid, ok := c.Locals("user_id").(uint); ok {
		actorID = uid
	}

	updated, err := h.svc.TransitionStage(uint(id), req.NextStageID, actorID, req.Notes)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "TRANSITION_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, updated, "Journey stage transitioned successfully")
}
