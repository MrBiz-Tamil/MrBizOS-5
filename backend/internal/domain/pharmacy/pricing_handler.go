package pharmacy

import (
	"drayya-core/internal/database"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// PriceCartRequest is the HTTP body for POST /pharmacy/pricing/price-cart.
type PriceCartRequest struct {
	Items                 []ReservationItem `json:"items"`
	DeliveryPincode       string            `json:"delivery_pincode"`
	PaymentMethod         string            `json:"payment_method"` // "prepaid" | "cod"
	InsuranceCoveredPaise int64             `json:"insurance_covered_paise"`
	DiscountPaise         int64             `json:"discount_paise"`
}

// PriceCartHandler returns a full itemised bill for a draft cart so the patient
// can review charges (MRP, GST, shipping, COD surcharge, insurance) before
// committing. Pure read-only — does not write to the DB.
//
// POST /pharmacy/pricing/price-cart
func PriceCartHandler(c *fiber.Ctx) error {
	var req PriceCartRequest
	if err := c.BodyParser(&req); err != nil || len(req.Items) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "items[] is required")
	}
	totals, err := PriceCart(c.Context(), database.DB, PricingRequest{
		Items:                 req.Items,
		DeliveryPincode:       req.DeliveryPincode,
		PaymentMethod:         PaymentMethod(req.PaymentMethod),
		InsuranceCoveredPaise: req.InsuranceCoveredPaise,
		DiscountPaise:         req.DiscountPaise,
	})
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "PRICING_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, totals, "Cart priced")
}
