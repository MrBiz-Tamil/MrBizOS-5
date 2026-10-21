package handlers

import (
	"drayya-core/internal/middleware"
	"drayya-core/internal/shipping"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type ShippingHandler struct {
	service *shipping.Service
}

func NewShippingHandler(service *shipping.Service) *ShippingHandler {
	return &ShippingHandler{service: service}
}

func (h *ShippingHandler) GetAllShipments(c *fiber.Ctx) error {
	shipments, err := h.service.GetAllShipments()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch shipments", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, shipments, "Shipments fetched")
}

func (h *ShippingHandler) GetShipment(c *fiber.Ctx) error {
	id := c.Params("id")
	shipment, err := h.service.GetShipment(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Shipment not found", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, shipment, "Shipment fetched")
}

func (h *ShippingHandler) SubmitTracking(c *fiber.Ctx) error {
	var req struct {
		ShipmentID uint   `json:"shipment_id"`
		Status     string `json:"status"`
		Location   string `json:"location"`
		Notes      string `json:"notes"`
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request", err.Error())
	}

	// Override ShipmentID if in params
	if idRaw := c.Params("id"); idRaw != "" {
		id, _ := strconv.Atoi(idRaw)
		req.ShipmentID = uint(id)
	}

	if err := h.service.AddShipmentUpdate(req.ShipmentID, req.Status, req.Location, req.Notes); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to update tracking", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, nil, "Tracking updated")
}

func (h *ShippingHandler) GetCouriers(c *fiber.Ctx) error {
	couriers, err := h.service.GetAvailableCouriers()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch couriers", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, couriers, "Couriers fetched")
}

func (h *ShippingHandler) PublicTrack(c *fiber.Ctx) error {
	id := c.Params("id")
	shipment, err := h.service.GetShipment(id)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Shipment not found")
	}
	return response.JSON(c, fiber.StatusOK, shipment, "Tracking data fetched")
}

func (h *ShippingHandler) GetPatientActiveDeliveries(c *fiber.Ctx) error {
	patientID, ok := c.Locals("user_id").(uint)
	if !ok || patientID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "User ID not found")
	}

	deliveries, err := h.service.GetPatientDeliveries(patientID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch patient deliveries", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, deliveries, "Patient deliveries fetched")
}

func (h *ShippingHandler) RegisterRoutes(router fiber.Router) {
	// Public tracking
	router.Get("/retail/track/:id", h.PublicTrack)

	s := router.Group("/shipping", middleware.Protected())
	s.Get("/deliveries", h.GetAllShipments)
	s.Get("/track/:id", h.GetShipment)
	s.Get("/shipments", h.GetAllShipments)
	s.Get("/shipments/:id", h.GetShipment)
	s.Post("/shipments/:id/tracking", h.SubmitTracking)
	s.Post("/tracking/submit", h.SubmitTracking)
	s.Get("/couriers/available", h.GetCouriers)
	s.Get("/couriers", h.GetCouriers)
	s.Get("/patient/active", h.GetPatientActiveDeliveries)
}
