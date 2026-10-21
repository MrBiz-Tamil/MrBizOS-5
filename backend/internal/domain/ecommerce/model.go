package ecommerce

import (
	"time"

	"gorm.io/gorm"
)

// PODProvider represents a Print-On-Demand fulfillment vendor (e.g. Printify, Gelato).
type PODProvider struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Code        string         `gorm:"size:64;uniqueIndex;not null" json:"code"`
	APIEndpoint string         `gorm:"size:255" json:"api_endpoint,omitempty"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// MarketplaceListing represents an item listed on external channels like Etsy, Amazon, or Shopify.
type MarketplaceListing struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Marketplace string         `gorm:"size:64;index;not null" json:"marketplace"` // ETSY, SHOPIFY, AMAZON
	ExternalID  string         `gorm:"size:128;index" json:"external_id,omitempty"`
	SKU         string         `gorm:"size:128;index;not null" json:"sku"`
	Title       string         `gorm:"size:255;not null" json:"title"`
	Price       float64        `gorm:"not null" json:"price"`
	Status      string         `gorm:"size:32;default:'ACTIVE';index" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// PODOrder tracks e-commerce order fulfillment and exact unit profitability.
type PODOrder struct {
	ID                     uint           `gorm:"primaryKey" json:"id"`
	OrganizationID         uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID         uint           `gorm:"index;default:4" json:"business_unit_id"` // 4: E-Commerce & POD
	OrderNumber            string         `gorm:"size:64;uniqueIndex;not null" json:"order_number"`
	CustomerID             uint           `gorm:"index;not null" json:"customer_id"`
	SKU                    string         `gorm:"size:128;not null" json:"sku"`
	Quantity               int            `gorm:"default:1" json:"quantity"`
	Revenue                float64        `gorm:"not null" json:"revenue"`
	ProductCost            float64        `gorm:"default:0" json:"product_cost"`
	ShippingFee            float64        `gorm:"default:0" json:"shipping_fee"`
	MarketplaceFee         float64        `gorm:"default:0" json:"marketplace_fee"`
	PaymentFee             float64        `gorm:"default:0" json:"payment_fee"`
	AdSpend                float64        `gorm:"default:0" json:"ad_spend"`
	ContributionMargin     float64        `gorm:"not null" json:"contribution_margin"`
	ContributionMarginPct  float64        `gorm:"not null" json:"contribution_margin_pct"`
	ShippingTrackingNumber string         `gorm:"size:128;index" json:"shipping_tracking_number,omitempty"`
	Status                 string         `gorm:"size:32;default:'RECEIVED';index" json:"status"` // RECEIVED, FULFILLED, SHIPPED, DELIVERED
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`
}

// AutoMigrate migrates e-commerce & POD domain models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&PODProvider{}, &MarketplaceListing{}, &PODOrder{})
}
