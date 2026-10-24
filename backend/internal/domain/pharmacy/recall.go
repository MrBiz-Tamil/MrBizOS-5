package pharmacy

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// RecallAffectedPatient is one row in the recall report — a patient who
// received the recalled batch on a given order. Pharmacy ops uses this list to
// pull stock, refund, and notify.
type RecallAffectedPatient struct {
	OrderID     uint      `json:"order_id"`
	OrderNumber string    `json:"order_number"`
	OrderItemID uint      `json:"order_item_id"`
	UserID      uint      `json:"user_id"`
	PatientName string    `json:"patient_name"`
	Phone       string    `json:"phone,omitempty"`
	Email       string    `json:"email,omitempty"`
	Quantity    int       `json:"quantity"`
	ShippedAt   time.Time `json:"shipped_at"`
}

// LookupOrderItemsByBatch returns every order item dispensed from the given
// batch (optionally further narrowed by medicine_id). Joins the patient's name
// and contact info from the identity user table so the recall workflow has
// everything it needs in one response.
func LookupOrderItemsByBatch(ctx context.Context, db *gorm.DB, batchNo string, medicineID uint) ([]RecallAffectedPatient, error) {
	batchNo = strings.TrimSpace(batchNo)
	if batchNo == "" {
		return nil, errors.New("batch_number is required")
	}

	type row struct {
		OrderID     uint
		OrderNumber string
		OrderItemID uint
		UserID      uint
		PatientName string
		Phone       string
		Email       string
		Quantity    int
		ShippedAt   time.Time
	}
	q := db.WithContext(ctx).
		Table("pharmacy_order_items oi").
		Select(`oi.order_id, o.order_number, oi.id AS order_item_id, o.user_id,
		         u.full_name AS patient_name, u.phone_number AS phone, u.email,
		         oi.quantity, o.created_at AS shipped_at`).
		Joins(`JOIN pharmacy_orders o ON o.id = oi.order_id`).
		Joins(`LEFT JOIN "user" u ON u.id = o.user_id`).
		Where(`oi.batch_number = ?`, batchNo)
	if medicineID > 0 {
		q = q.Where(`oi.medicine_id = ?`, medicineID)
	}

	var rows []row
	if err := q.Order("o.created_at DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]RecallAffectedPatient, 0, len(rows))
	for _, r := range rows {
		out = append(out, RecallAffectedPatient(r))
	}
	return out, nil
}

// AttachBatchToOrderItem is the helper a fulfillment / dispatch handler calls
// when it picks an inventory batch for an order line. Centralising this here
// means we never have an order shipped without batch traceability.
func AttachBatchToOrderItem(ctx context.Context, db *gorm.DB, orderItemID, medicineID uint, batchNo string, expiry *time.Time) error {
	batchNo = strings.TrimSpace(batchNo)
	if orderItemID == 0 || batchNo == "" {
		return errors.New("order_item_id and batch_no are required")
	}
	updates := map[string]interface{}{
		"batch_number": batchNo,
	}
	if medicineID != 0 {
		updates["medicine_id"] = medicineID
	}
	if expiry != nil {
		updates["expiry_date"] = *expiry
	}
	return db.WithContext(ctx).Table("pharmacy_order_items").
		Where("id = ?", orderItemID).Updates(updates).Error
}
