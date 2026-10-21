package pharmacy

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// This file holds pharmacy models whose Fiber HTTP handlers were removed with
// the REST server. The models remain because the repository, migrations and
// (future) ConnectRPC services depend on them.

// DispenseRecord is one line of a fulfilled/dispensed order item.
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

// ProductReview is a patient review of a medicine or wellness product.
type ProductReview struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	ProductID        uint           `gorm:"index;not null" json:"product_id"`
	ProductType      string         `gorm:"index;size:16;not null" json:"product_type"` // medicine | wellness
	PatientID        uint           `gorm:"index;not null" json:"patient_id"`
	Rating           int            `gorm:"not null;check:rating BETWEEN 1 AND 5" json:"rating"`
	Body             string         `gorm:"type:text" json:"body"`
	VerifiedPurchase bool           `gorm:"default:false" json:"verified_purchase"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ProductReview) TableName() string { return "product_reviews" }

// Coupon is a discount code applied at checkout.
type Coupon struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	Code             string         `gorm:"uniqueIndex;size:32;not null" json:"code"`
	Description      string         `json:"description"`
	Type             string         `gorm:"size:16;not null" json:"type"` // percent | fixed
	Value            int64          `gorm:"not null" json:"value"`        // % (0-100) or paise
	MinOrderPaise    int64          `json:"min_order_paise"`
	MaxDiscountPaise int64          `json:"max_discount_paise"` // 0 = no cap
	MaxUses          int            `json:"max_uses"`           // 0 = unlimited
	UsedCount        int            `gorm:"default:0" json:"used_count"`
	IsActive         bool           `gorm:"default:true;index" json:"is_active"`
	ExpiresAt        *time.Time     `gorm:"index" json:"expires_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Coupon) TableName() string { return "coupons" }

// ProductBundle is a curated set of products sold together at a bundle price.
type ProductBundle struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	Name             string         `gorm:"not null" json:"name"`
	Slug             string         `gorm:"uniqueIndex;size:80" json:"slug"`
	ConditionTag     string         `gorm:"index;size:64" json:"condition_tag"` // diabetes | heart | immunity | …
	Description      string         `json:"description"`
	ImageURL         string         `json:"image_url"`
	Items            datatypes.JSON `gorm:"type:jsonb" json:"items"` // [{product_id, product_type, qty}]
	BundlePricePaise int64          `gorm:"not null" json:"bundle_price_paise"`
	IsActive         bool           `gorm:"default:true;index" json:"is_active"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ProductBundle) TableName() string { return "product_bundles" }
