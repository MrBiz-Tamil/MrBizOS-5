package pharmacy

// counter_sale.go — POST /pharmacy/orders/counter-sale
//
// Counter (walk-in) sales bypass the reservation and online-payment flows:
// the pharmacist is physically present, stock is decremented immediately
// inside a transaction, and the order is created with status "delivered" /
// payment_status "paid".  FEFO (first-expiry-first-out) batch selection is
// applied automatically so older stock ships first.

import (
	"encoding/json"
	"fmt"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// ModeCounter is the delivery mode used for over-the-counter walk-in sales.
const ModeCounter DeliveryMode = "COUNTER"

// CounterSaleRequest is the POST body for /pharmacy/orders/counter-sale.
type CounterSaleRequest struct {
	Items         []ReservationItem `json:"items"`          // medicine_id + quantity
	PaymentMethod string            `json:"payment_method"` // "cash" | "card" | "upi"
	DiscountPct   float64           `json:"discount_pct"`   // 0–50%; applied as flat paise discount
}

// CounterSaleResponse wraps the created order + full bill breakdown.
type CounterSaleResponse struct {
	Order  *PharmacyOrder `json:"order"`
	Totals *CartTotals    `json:"totals"`
}

// CounterSaleHandler creates a walk-in sale in one transaction:
//  1. Price the cart (GST-inclusive).
//  2. FEFO-pick inventory batches and decrement atomically.
//  3. Create PharmacyOrder (status=delivered, payment_status=paid).
//  4. Create PharmacyOrderItems with batch/expiry traceability.
//
// POST /pharmacy/orders/counter-sale
// Auth: PharmacyManagerOnly (pharmacist, pharmacy_admin, admin)
func (h *Handler) CounterSaleHandler(c *fiber.Ctx) error {
	pharmacistID, ok := c.Locals("user_id").(uint)
	if !ok || pharmacistID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	// pharmacy_id is set by Protected() when the JWT contains one.
	pharmacyID, _ := c.Locals("pharmacy_id").(uint)

	var req CounterSaleRequest
	if err := c.BodyParser(&req); err != nil || len(req.Items) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "items[] is required")
	}
	if req.DiscountPct < 0 || req.DiscountPct > 50 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_DISCOUNT", "discount_pct must be 0–50")
	}
	validMethods := map[string]bool{"cash": true, "card": true, "upi": true}
	if req.PaymentMethod == "" {
		req.PaymentMethod = "cash"
	}
	if !validMethods[req.PaymentMethod] {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_PAYMENT_METHOD", "payment_method must be cash, card, or upi")
	}

	// Price the cart — returns per-line GST + grand total.
	totals, err := PriceCart(c.Context(), h.db, PricingRequest{
		Items:         req.Items,
		PaymentMethod: PaymentPrepaid, // counter sales are always prepaid (collected in person)
	})
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "PRICING_FAILED", err.Error())
	}

	// Apply flat discount from discount_pct.
	if req.DiscountPct > 0 {
		discountPaise := int64(float64(totals.SubtotalPaise) * req.DiscountPct / 100)
		totals.DiscountPaise = discountPaise
		grand := totals.SubtotalPaise + totals.TotalGSTPaise - discountPaise
		if grand < 0 {
			grand = 0
		}
		totals.GrandTotalPaise = grand
	}

	// Build the order items list (populated inside transaction with batch info).
	var orderItems []PharmacyOrderItem

	now := time.Now()
	var order *PharmacyOrder

	txErr := database.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		for _, it := range req.Items {
			// FEFO: pick batches for this medicine ordered by earliest expiry.
			var batches []PharmacyInventory
			q := tx.Where(
				"medicine_id = ? AND stock_quantity > 0 AND is_active = ? AND expiry_date > ?",
				it.MedicineID, true, now,
			).Order("expiry_date ASC")
			if pharmacyID > 0 {
				q = q.Where("pharmacy_id = ?", pharmacyID)
			}
			if err := q.Find(&batches).Error; err != nil {
				return fmt.Errorf("failed to fetch inventory for medicine %d: %w", it.MedicineID, err)
			}

			remaining := it.Quantity
			for _, batch := range batches {
				if remaining <= 0 {
					break
				}
				take := remaining
				if batch.StockQuantity < take {
					take = batch.StockQuantity
				}
				res := tx.Exec(
					`UPDATE pharmacyinventory
					    SET stock_quantity = stock_quantity - ?
					  WHERE id = ? AND stock_quantity >= ?`,
					take, batch.ID, take,
				)
				if res.Error != nil {
					return fmt.Errorf("failed to decrement batch %d: %w", batch.ID, res.Error)
				}
				if res.RowsAffected == 0 {
					// Race condition — another sale grabbed this batch; try next.
					continue
				}
				remaining -= take
			}
			if remaining > 0 {
				return fmt.Errorf("insufficient stock for medicine %d (%d units short)", it.MedicineID, remaining)
			}

			// Find the pricing line for this medicine.
			var unitPrice int64
			for _, line := range totals.Lines {
				if line.MedicineID == it.MedicineID {
					unitPrice = line.UnitMRPPaise
					break
				}
			}

			// Pick the first active batch for traceability (FEFO already sorted).
			var traceBatch PharmacyInventory
			var batchNum string
			var expiryPtr *time.Time
			q2 := tx.Where("medicine_id = ? AND is_active = ?", it.MedicineID, true)
			if pharmacyID > 0 {
				q2 = q2.Where("pharmacy_id = ?", pharmacyID)
			}
			if err := q2.Order("expiry_date ASC").First(&traceBatch).Error; err == nil {
				batchNum = traceBatch.BatchNumber
				t := traceBatch.ExpiryDate
				expiryPtr = &t
			}

			var medName string
			var med Medicine
			if err := tx.First(&med, it.MedicineID).Error; err == nil {
				medName = med.Name
			}

			orderItems = append(orderItems, PharmacyOrderItem{
				MedicineName: medName,
				MedicineID:   it.MedicineID,
				Quantity:     it.Quantity,
				UnitPrice:    unitPrice,
				TotalPrice:   unitPrice * int64(it.Quantity),
				BatchNumber:  batchNum,
				ExpiryDate:   expiryPtr,
			})
		}

		// Serialise order items to JSON.
		itemsJSON, err := json.Marshal(orderItems)
		if err != nil {
			return fmt.Errorf("failed to serialise order items: %w", err)
		}

		orderNum := fmt.Sprintf("CTR-%s", generateOrderToken())
		order = &PharmacyOrder{
			OrderNumber:   orderNum,
			UserID:        pharmacistID,
			PharmacyID:    pharmacyID,
			Items:         itemsJSON,
			Amount:        totals.GrandTotalPaise,
			Currency:      "INR",
			Status:        "delivered",
			PaymentStatus: "paid",
			PaymentMethod: req.PaymentMethod,
			DeliveryMode:  ModeCounter,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := tx.Create(order).Error; err != nil {
			return fmt.Errorf("failed to create order: %w", err)
		}

		// Persist line items with batch traceability.
		for i := range orderItems {
			orderItems[i].OrderID = order.ID
		}
		if err := tx.Create(&orderItems).Error; err != nil {
			return fmt.Errorf("failed to create order items: %w", err)
		}

		return nil
	})
	if txErr != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "SALE_FAILED", txErr.Error())
	}

	return response.JSON(c, fiber.StatusCreated, CounterSaleResponse{
		Order:  order,
		Totals: totals,
	}, "Counter sale completed")
}
