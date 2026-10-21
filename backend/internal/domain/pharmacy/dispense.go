package pharmacy

// dispense.go — Feature 3: Dispense step with per-batch records and dispensing label.
//
// A dispense action is distinct from pack:
//   pack   → staff assembles the order (no batch-level tracking)
//   dispense → staff selects exact inventory batches, decrements them atomically,
//              creates DispenseRecord entries, and generates a label.
//
// The DispenseRecord table satisfies regulatory batch-traceability requirements.

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"drayya-core/internal/database"
	"drayya-core/internal/utils/response"
)

// DispenseRecord tracks which inventory batch was used for each dispensed item.
type DispenseRecord struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	OrderID          uint      `gorm:"not null;index" json:"order_id"`
	OrderItemID      uint      `gorm:"not null;index" json:"order_item_id"`
	InventoryBatchID uint      `gorm:"not null" json:"inventory_batch_id"`
	MedicineID       uint      `gorm:"not null" json:"medicine_id"`
	Quantity         int       `gorm:"not null" json:"quantity"`
	DispensedByID    uint      `gorm:"not null" json:"dispensed_by_id"`
	DispensedAt      time.Time `json:"dispensed_at"`
	LabelPrinted     bool      `gorm:"default:false" json:"label_printed"`
	CreatedAt        time.Time `json:"created_at"`
}

func (DispenseRecord) TableName() string { return "dispense_records" }

// DispenseOrderItem is one line in a dispense request.
type DispenseOrderItem struct {
	OrderItemID      uint `json:"order_item_id"`
	InventoryBatchID uint `json:"inventory_batch_id"` // which specific batch to draw from
	Quantity         int  `json:"quantity"`
}

// DispenseOrderRequest is the POST /orders/:id/dispense body.
type DispenseOrderRequest struct {
	Items []DispenseOrderItem `json:"items"`
}

// dispensingLabel is the label payload returned after a successful dispense.
type dispensingLabel struct {
	OrderID      uint              `json:"order_id"`
	OrderNumber  string            `json:"order_number"`
	DispensedAt  time.Time         `json:"dispensed_at"`
	PharmacistID uint              `json:"pharmacist_id"`
	Items        []dispenseLabelItem `json:"items"`
}

type dispenseLabelItem struct {
	MedicineName string    `json:"medicine_name"`
	BatchNumber  string    `json:"batch_number"`
	ExpiryDate   time.Time `json:"expiry_date"`
	Quantity     int       `json:"quantity"`
}

// DispenseOrder processes the dispense step for a packed order.
//
// POST /pharmacy/orders/:id/dispense
//
// For each item in the request:
//  1. Verify the inventory batch exists and has enough stock.
//  2. Atomically decrement the batch: UPDATE WHERE quantity >= ?
//  3. Create a DispenseRecord.
//
// After all items are processed the order is advanced to "dispensed".
// Returns the dispense records and a printable label payload.
func (h *Handler) DispenseOrder(c *fiber.Ctx) error {
	pharmacistID, ok := c.Locals("user_id").(uint)
	if !ok || pharmacistID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	orderIDStr := c.Params("id")
	orderID, err := parseUintParam(orderIDStr)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid order ID")
	}

	var req DispenseOrderRequest
	if err := c.BodyParser(&req); err != nil || len(req.Items) == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "items array is required")
	}

	// Fetch the order — must be in "packed" state.
	var order PharmacyOrder
	if err := h.db.First(&order, orderID).Error; err != nil {
		return response.Error(c, fiber.StatusNotFound, "NOT_FOUND", "Order not found")
	}
	if order.Status != "packed" {
		return response.Error(c, fiber.StatusUnprocessableEntity, "INVALID_STATE",
			fmt.Sprintf("Order must be in 'packed' state to dispense; current state is '%s'", order.Status))
	}

	now := time.Now()
	var records []DispenseRecord
	var labelItems []dispenseLabelItem

	// Process each item inside a DB transaction.
	txErr := database.DB.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			if item.InventoryBatchID == 0 || item.Quantity <= 0 {
				return fmt.Errorf("item inventory_batch_id and quantity > 0 are required")
			}

			// Atomically decrement the batch — fail if insufficient stock.
			result := tx.Exec(
				`UPDATE pharmacyinventory
				    SET stock_quantity = stock_quantity - ?
				  WHERE id = ? AND stock_quantity >= ?`,
				item.Quantity, item.InventoryBatchID, item.Quantity,
			)
			if result.Error != nil {
				return fmt.Errorf("failed to decrement batch %d: %w", item.InventoryBatchID, result.Error)
			}
			if result.RowsAffected == 0 {
				return fmt.Errorf("insufficient stock in batch %d (requested %d units)", item.InventoryBatchID, item.Quantity)
			}

			// Fetch batch info for the label.
			var batch PharmacyInventory
			if err := tx.First(&batch, item.InventoryBatchID).Error; err != nil {
				return fmt.Errorf("batch %d not found: %w", item.InventoryBatchID, err)
			}

			// Create the dispense record.
			rec := DispenseRecord{
				OrderID:          orderID,
				OrderItemID:      item.OrderItemID,
				InventoryBatchID: item.InventoryBatchID,
				MedicineID:       batch.MedicineID,
				Quantity:         item.Quantity,
				DispensedByID:    pharmacistID,
				DispensedAt:      now,
			}
			if err := tx.Create(&rec).Error; err != nil {
				return fmt.Errorf("failed to create dispense record: %w", err)
			}
			records = append(records, rec)

			// Gather label info.
			var med Medicine
			_ = tx.First(&med, batch.MedicineID).Error
			labelItems = append(labelItems, dispenseLabelItem{
				MedicineName: med.Name,
				BatchNumber:  batch.BatchNumber,
				ExpiryDate:   batch.ExpiryDate,
				Quantity:     item.Quantity,
			})
		}

		// Advance order to "dispensed".
		return tx.Model(&PharmacyOrder{}).
			Where("id = ?", orderID).
			Update("status", "dispensed").Error
	})
	if txErr != nil {
		return response.Error(c, fiber.StatusUnprocessableEntity, "DISPENSE_FAILED", txErr.Error())
	}

	// Mark labels as printed (W15: log failure instead of silently discarding).
	if lpErr := database.DB.WithContext(c.Context()).Model(&DispenseRecord{}).
		Where("order_id = ?", orderID).
		Update("label_printed", true).Error; lpErr != nil {
		log.Printf("[dispense] failed to mark label_printed for order %d: %v", orderID, lpErr)
	}

	label := dispensingLabel{
		OrderID:      order.ID,
		OrderNumber:  order.OrderNumber,
		DispensedAt:  now,
		PharmacistID: pharmacistID,
		Items:        labelItems,
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"records": records,
		"label":   label,
	}, "Order dispensed successfully")
}

// parseUintParam is a small helper shared within the package.
func parseUintParam(s string) (uint, error) {
	var n uint
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// RegisterDispenseRoutes wires the dispense endpoint onto the pharmacy router group.
// Called from Handler.RegisterRoutes.
func (h *Handler) registerDispenseRoute(pharmacyGroup fiber.Router) {
	pharmacyGroup.Post("/orders/:id/dispense", h.DispenseOrder)
}

// GetDispenseRecords returns all dispense records for a given order.
// GET /pharmacy/orders/:id/dispense-records
func (h *Handler) GetDispenseRecords(c *fiber.Ctx) error {
	_, ok := c.Locals("user_id").(uint)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}

	orderID, err := parseUintParam(c.Params("id"))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid order ID")
	}

	var records []DispenseRecord
	if err := h.db.WithContext(c.Context()). // W16: use request context, not Background()
		Where("order_id = ?", orderID).
		Find(&records).Error; err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "FETCH_FAILED", err.Error())
	}
	return response.JSON(c, fiber.StatusOK, records, "Dispense records fetched")
}
