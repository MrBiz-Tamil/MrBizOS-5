package v2

import (
	"context"
	"errors"
	"time"

	"drayya-core/internal/database"
)

// addToCartTool adds a medicine to the patient's pharmacy cart. Reversible
// (the user can remove items at checkout), so we don't gate it behind a
// confirmation handshake.
//
// Patient-only. Quantity capped at 10 per call to prevent runaway tool use.
var addToCartTool = Tool{
	Name: "add_to_cart",
	Description: "Add a medicine product to the user's pharmacy cart. Use after " +
		"the user agrees to buy a product surfaced by search_products. The user " +
		"can review and remove items at checkout, so this is reversible.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"product_id": map[string]any{
				"type":        "integer",
				"description": "The medicine ID from search_products results.",
			},
			"quantity": map[string]any{
				"type":        "integer",
				"description": "How many units to add. Defaults to 1. Max 10.",
			},
		},
		"required": []string{"product_id"},
	},
	Allowed:       []PersonaRole{RolePatient},
	Confirm:       false, // reversible — user reviews cart at checkout
	MaxPerSession: 10,
	MaxPerMinute:  5,
	Execute:       executeAddToCart,
}

// devi_v2_cart_items is a lightweight per-session staging table. The actual
// PharmacyOrder is created at checkout; this just records intent. We don't
// touch the real cart/order tables here — avoids cross-domain coupling and
// keeps the tool idempotent.
type cartStagingRow struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"index"`
	SessionID string    `gorm:"index"`
	ProductID uint
	Quantity  int
	CreatedAt time.Time
}

func (cartStagingRow) TableName() string { return "devi_v2_cart_items" }

func executeAddToCart(ctx context.Context, args map[string]any) (any, error) {
	persona := PersonaFromContext(ctx)
	if persona.UserID == nil {
		return map[string]any{"error": "must be signed in to add to cart"}, nil
	}
	if database.DB == nil {
		return nil, errors.New("database unavailable")
	}

	var productID uint
	if v, ok := args["product_id"].(float64); ok && v > 0 {
		productID = uint(v)
	}
	if productID == 0 {
		return map[string]any{"error": "product_id is required"}, nil
	}
	qty := 1
	if v, ok := args["quantity"].(float64); ok && v >= 1 && v <= 10 {
		qty = int(v)
	}

	// Verify the product exists and is OTC.
	var ok bool
	database.DB.Raw(`SELECT EXISTS(
		SELECT 1 FROM medicine
		WHERE id = ? AND is_active = ? AND deleted_at IS NULL
		  AND is_prescription_required = ?
	)`, productID, true, false).Scan(&ok)
	if !ok {
		return map[string]any{"error": "product not found or requires a prescription"}, nil
	}

	// AutoMigrate the staging table on first use — keeps the table tied to
	// V2 without touching the pharmacy migration set.
	_ = database.DB.AutoMigrate(&cartStagingRow{})

	sid := SessionIDFromContext(ctx)
	row := cartStagingRow{
		UserID:    *persona.UserID,
		SessionID: sid,
		ProductID: productID,
		Quantity:  qty,
		CreatedAt: time.Now(),
	}
	if err := database.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}

	// Count current items so the LLM can summarise.
	var total int64
	database.DB.Model(&cartStagingRow{}).
		Where("user_id = ? AND session_id = ?", *persona.UserID, sid).
		Count(&total)

	return map[string]any{
		"product_id":  productID,
		"quantity":    qty,
		"cart_total":  total,
		"checkout_url": "/pharmacy/cart",
		"message":     "Added to cart. Review and checkout at /pharmacy/cart.",
	}, nil
}
