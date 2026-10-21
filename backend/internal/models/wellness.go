//go:build ignore
// +build ignore

package models

import (
	"time"

	"gorm.io/gorm"
)

type WellnessProduct struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	Name               string         `json:"name"`
	Brand              string         `json:"brand"`
	Category           string         `json:"category"`
	Description        string         `json:"description"`
	MRP                int64          `json:"mrp"`
	SellingPrice       int64          `json:"selling_price"`
	Unit               string         `json:"unit"` // e.g., "500g", "100ml"
	IsOrganicCertified bool           `json:"is_organic_certified"`
	Ingredients        string         `json:"ingredients"`      // Text or JSON
	NutritionalInfo    string         `json:"nutritional_info"` // Text or JSON
	ShelfLifeDays      int            `json:"shelf_life_days"`
	ImageURL           string         `json:"image_url"`
	IsActive           bool           `gorm:"default:true" json:"is_active"`
	StockQuantity      int            `json:"stock_quantity"` // Simple inventory tracking
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

type WellnessOrder struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	OrderNumber     string    `gorm:"uniqueIndex" json:"order_number"`
	UserID          uint      `gorm:"index" json:"user_id"`
	Subtotal        int64     `json:"subtotal"`
	TotalAmount     int64     `json:"total_amount"`
	Status          string    `json:"status"` // pending, processing, shipped, delivered, cancelled
	DeliveryAddress string    `json:"delivery_address"`
	OrderedAt       time.Time `json:"ordered_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Items (stored as JSON for simplicity in this module, or separate table)
	// For full normalization we'd use WellnessOrderItem table, but sticking to simple for now as requested by user speed.
	// Actually, let's do it properly with a relation if possible, but the frontend sends `items: any[]`.
	// I'll stick to a related table for better querying later.
	Items []WellnessOrderItem `gorm:"foreignKey:OrderID" json:"items"`
}

type WellnessOrderItem struct {
	ID        uint  `gorm:"primaryKey" json:"id"`
	OrderID   uint  `gorm:"index" json:"order_id"`
	ProductID uint  `json:"product_id"`
	Quantity  int   `json:"quantity"`
	UnitPrice int64 `json:"unit_price"`

	Product WellnessProduct `gorm:"foreignKey:ProductID" json:"product"`
}
