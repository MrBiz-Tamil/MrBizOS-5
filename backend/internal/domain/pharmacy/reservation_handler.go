package pharmacy

import (
	"drayya-core/internal/database"
	"drayya-core/internal/utils/response"
	"time"

	"github.com/gofiber/fiber/v2"
)

// ReserveStockRequest is the body for POST /pharmacy/reservations.
type ReserveStockRequest struct {
	Latitude  float64           `json:"latitude"`
	Longitude float64           `json:"longitude"`
	TTLMinutes int              `json:"ttl_minutes,omitempty"`
	Items     []ReservationItem `json:"items"`
}

// ReserveStock picks the nearest pharmacy that can fulfill the cart and locks
// inventory for the patient with a TTL. Two concurrent carts cannot both reserve
// the same packs — the second sees the first's hold and either falls back to a
// further pharmacy with stock or fails cleanly.
//
// POST /pharmacy/reservations
// body: { latitude, longitude, items: [{ medicine_id, quantity }], ttl_minutes? }
func ReserveStock(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	var req ReserveStockRequest
	if err := c.BodyParser(&req); err != nil || len(req.Items) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "items[] is required")
	}
	ttl := DefaultReservationTTL
	if req.TTLMinutes > 0 {
		ttl = time.Duration(req.TTLMinutes) * time.Minute
	}
	res, err := PickNearestPharmacyAndReserve(c.Context(), database.DB, userID,
		req.Latitude, req.Longitude, req.Items, ttl)
	if err != nil {
		// Out-of-stock is the most common branch; surface it as a 409 so the
		// frontend can show "We couldn't reserve all your items" without treating
		// it as a server error.
		return response.Error(c, fiber.StatusConflict, "RESERVATION_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, res, "Stock reserved")
}

// ConfirmCartReservation is called by the order-checkout handler once payment
// clears — decrements actual inventory atomically.
//
// POST /pharmacy/reservations/:cart_id/confirm
func ConfirmCartReservation(c *fiber.Ctx) error {
	if _, ok := c.Locals("user_id").(uint); !ok {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	cartID := c.Params("cart_id")
	if cartID == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "cart_id is required")
	}
	if err := ConfirmReservations(c.Context(), database.DB, cartID); err != nil {
		return response.Error(c, fiber.StatusConflict, "CONFIRM_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"cart_id": cartID, "status": "confirmed"}, "Inventory committed")
}

// ReleaseCartReservation frees the hold when the patient abandons checkout.
//
// POST /pharmacy/reservations/:cart_id/release
func ReleaseCartReservation(c *fiber.Ctx) error {
	if _, ok := c.Locals("user_id").(uint); !ok {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	cartID := c.Params("cart_id")
	if cartID == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "cart_id is required")
	}
	if err := ReleaseReservations(c.Context(), database.DB, cartID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "RELEASE_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{"cart_id": cartID, "status": "released"}, "Reservation released")
}
