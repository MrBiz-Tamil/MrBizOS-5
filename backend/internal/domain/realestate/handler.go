package realestate

import (
	"strconv"
	"time"

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
	re := r.Group("/realestate")
	re.Get("/properties", h.ListProperties)
	re.Post("/properties", h.CreateProperty)
	re.Get("/properties/:id", h.GetProperty)
	re.Post("/site-visits", h.ScheduleSiteVisit)
	re.Get("/site-visits", h.ListSiteVisits)
	re.Post("/deals", h.CreateDeal)
	re.Get("/deals", h.ListDeals)
	re.Put("/deals/:id/status", h.UpdateDealStatus)
}

func (h *Handler) ListProperties(c *fiber.Ctx) error {
	city := c.Query("city")
	txType := c.Query("transaction_type")
	var maxPrice float64
	if p := c.Query("max_price"); p != "" {
		if val, err := strconv.ParseFloat(p, 64); err == nil {
			maxPrice = val
		}
	}

	props, err := h.svc.ListProperties(city, txType, maxPrice)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, props, "Properties fetched successfully")
}

func (h *Handler) CreateProperty(c *fiber.Ctx) error {
	var prop Property
	if err := c.BodyParser(&prop); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	created, err := h.svc.CreateProperty(&prop)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, created, "Property created successfully")
}

func (h *Handler) GetProperty(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Property ID must be an integer")
	}

	prop, err := h.svc.GetProperty(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Property not found")
	}
	return response.JSON(c, fiber.StatusOK, prop, "Property fetched successfully")
}

func (h *Handler) ScheduleSiteVisit(c *fiber.Ctx) error {
	var req struct {
		PropertyID  uint   `json:"property_id"`
		BuyerID     uint   `json:"buyer_id"`
		AgentID     uint   `json:"agent_id"`
		ScheduledAt string `json:"scheduled_at"`
		Notes       string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	scheduledTime := time.Now().Add(24 * time.Hour)
	if req.ScheduledAt != "" {
		if parsed, err := time.Parse(time.RFC3339, req.ScheduledAt); err == nil {
			scheduledTime = parsed
		}
	}

	if req.BuyerID == 0 {
		if uid, ok := c.Locals("user_id").(uint); ok {
			req.BuyerID = uid
		} else {
			req.BuyerID = 1
		}
	}
	if req.AgentID == 0 {
		req.AgentID = 1
	}

	visit, err := h.svc.ScheduleSiteVisit(req.PropertyID, req.BuyerID, req.AgentID, scheduledTime, req.Notes)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "SCHEDULE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, visit, "Site visit scheduled successfully")
}

func (h *Handler) ListSiteVisits(c *fiber.Ctx) error {
	var agentID, buyerID uint
	if a := c.Query("agent_id"); a != "" {
		if id, err := strconv.ParseUint(a, 10, 32); err == nil {
			agentID = uint(id)
		}
	}
	if b := c.Query("buyer_id"); b != "" {
		if id, err := strconv.ParseUint(b, 10, 32); err == nil {
			buyerID = uint(id)
		}
	}

	visits, err := h.svc.ListSiteVisits(agentID, buyerID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, visits, "Site visits fetched successfully")
}

func (h *Handler) CreateDeal(c *fiber.Ctx) error {
	var req struct {
		PropertyID uint    `json:"property_id"`
		BuyerID    uint    `json:"buyer_id"`
		SellerID   uint    `json:"seller_id"`
		Price      float64 `json:"agreed_price"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	if req.BuyerID == 0 {
		req.BuyerID = 1
	}
	if req.SellerID == 0 {
		req.SellerID = 2
	}

	deal, err := h.svc.CreateDeal(req.PropertyID, req.BuyerID, req.SellerID, req.Price)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "DEAL_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, deal, "Property deal created successfully")
}

func (h *Handler) ListDeals(c *fiber.Ctx) error {
	deals, err := h.svc.ListDeals()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, deals, "Property deals fetched successfully")
}

func (h *Handler) UpdateDealStatus(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Deal ID must be an integer")
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	updated, err := h.svc.UpdateDealStatus(uint(id), req.Status)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "UPDATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, updated, "Property deal status updated successfully")
}
