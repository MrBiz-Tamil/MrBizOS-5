package ecommerce

import (
	"strconv"

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
	ec := r.Group("/ecommerce")
	ec.Get("/orders", h.ListPODOrders)
	ec.Post("/orders", h.CreatePODOrder)
	ec.Get("/orders/:id", h.GetPODOrder)
	ec.Get("/listings", h.ListListings)
	ec.Post("/listings", h.CreateListing)
}

func (h *Handler) ListPODOrders(c *fiber.Ctx) error {
	orders, err := h.svc.ListPODOrders()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, orders, "E-commerce orders fetched successfully")
}

func (h *Handler) CreatePODOrder(c *fiber.Ctx) error {
	var req struct {
		CustomerID     uint    `json:"customer_id"`
		SKU            string  `json:"sku"`
		Quantity       int     `json:"quantity"`
		Revenue        float64 `json:"revenue"`
		ProductCost    float64 `json:"product_cost"`
		ShippingFee    float64 `json:"shipping_fee"`
		MarketplaceFee float64 `json:"marketplace_fee"`
		PaymentFee     float64 `json:"payment_fee"`
		AdSpend        float64 `json:"ad_spend"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	if req.CustomerID == 0 {
		req.CustomerID = 1
	}

	order, err := h.svc.CreatePODOrder(req.CustomerID, req.SKU, req.Quantity, req.Revenue, req.ProductCost, req.ShippingFee, req.MarketplaceFee, req.PaymentFee, req.AdSpend)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, order, "E-commerce order created successfully")
}

func (h *Handler) GetPODOrder(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Order ID must be an integer")
	}

	order, err := h.svc.GetPODOrder(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}
	return response.JSON(c, fiber.StatusOK, order, "E-commerce order fetched successfully")
}

func (h *Handler) ListListings(c *fiber.Ctx) error {
	marketplace := c.Query("marketplace")
	listings, err := h.svc.ListListings(marketplace)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, listings, "Marketplace listings fetched successfully")
}

func (h *Handler) CreateListing(c *fiber.Ctx) error {
	var req struct {
		Marketplace string  `json:"marketplace"`
		SKU         string  `json:"sku"`
		Title       string  `json:"title"`
		Price       float64 `json:"price"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_BODY", err.Error())
	}

	listing, err := h.svc.CreateListing(req.Marketplace, req.SKU, req.Title, req.Price)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "CREATE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusCreated, listing, "Marketplace listing created successfully")
}
