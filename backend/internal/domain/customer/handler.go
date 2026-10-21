package customer

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
	crm := r.Group("/crm")
	crm.Get("/leads", h.ListLeads)
	crm.Post("/leads", h.CreateLead)

	c360 := r.Group("/customer360")
	c360.Get("/:id", h.GetCustomer360)
}

func (h *Handler) ListLeads(c *fiber.Ctx) error {
	buID := middleware.GetBusinessUnitID(c)
	status := c.Query("status")

	leads, err := h.svc.ListLeads(buID, status)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, leads, "Leads fetched successfully")
}

func (h *Handler) CreateLead(c *fiber.Ctx) error {
	var lead Lead
	if err := c.BodyParser(&lead); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}
	lead.BusinessUnitID = middleware.GetBusinessUnitID(c)
	lead.OrganizationID = middleware.GetOrganizationID(c)

	created, err := h.svc.CreateLead(&lead)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, created, "Lead created successfully")
}

func (h *Handler) GetCustomer360(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Customer ID must be an integer")
	}

	// Clinical permission check
	role, _ := c.Locals("role").(string)
	hasClinicalAccess := (role == "doctor" || role == "admin" || role == "super_admin")

	profile, err := h.svc.GetCustomer360(uint(id), hasClinicalAccess)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, profile, "Customer 360 profile fetched successfully")
}
