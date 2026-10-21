package store

import (
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/services/order"
	"drayya-core/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func CreateOrder(c *fiber.Ctx) error {
	// UserID MUST come from the verified JWT — never trust the request body.
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required to place an order")
	}

	var req struct {
		Lat             float64              `json:"lat"`
		Lng             float64              `json:"lng"`
		City            string               `json:"city"`
		Items           []order.CartItem     `json:"items"`
		DeliveryAddress string               `json:"delivery_address"`
		DeliveryPhone   string               `json:"delivery_phone"`
		DestPincode     string               `json:"dest_pincode"`
		DeliveryMode    pharmacy.DeliveryMode `json:"delivery_mode"` // "QUICK" | "STANDARD"; auto-detected if blank
	}

	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST_BODY", "invalid request body")
	}

	orderReq := order.OrderRequest{
		UserID:          userID,
		Items:           req.Items,
		Lat:             req.Lat,
		Lng:             req.Lng,
		City:            req.City,
		DeliveryAddress: req.DeliveryAddress,
		DeliveryPhone:   req.DeliveryPhone,
		DestPincode:     req.DestPincode,
		DeliveryMode:    req.DeliveryMode,
	}

	masterOrder, err := order.CreateOrder(c.Context(), orderReq)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ORDER_CREATION_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusCreated, masterOrder, "Created")
}

func GetOrders(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	_ = strconv.Itoa // keep import used by CheckDelivery below
	orders, err := order.ListOrders(userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "ORDERS_FETCH_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, orders, "Fetched")
}

func CheckDelivery(c *fiber.Ctx) error {
	pincode := c.Query("pincode")
	// Mock delivery check logic
	if len(pincode) == 6 && pincode[0] != '0' {
		return response.JSON(c, fiber.StatusOK, fiber.Map{
			"pincode":        pincode,
			"serviceable":    true,
			"estimated_days": 2,
		}, "Fetched")
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"pincode":     pincode,
		"serviceable": false,
		"error":       "Invalid or unserviceable pincode",
	}, "Fetched")
}
