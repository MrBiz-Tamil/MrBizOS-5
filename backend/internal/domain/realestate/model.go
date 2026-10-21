package realestate

import (
	"time"

	"gorm.io/gorm"
)

// Property Types
const (
	TypeApartment  = "APARTMENT"
	TypeVilla      = "VILLA"
	TypeCommercial = "COMMERCIAL"
	TypeLand       = "LAND"
)

// Transaction Types
const (
	TxSale = "SALE"
	TxRent = "RENT"
)

// Property represents a real estate listing.
type Property struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	OrganizationID  uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID  uint           `gorm:"index;default:3" json:"business_unit_id"` // 3: Real Estate
	Title           string         `gorm:"size:255;not null" json:"title"`
	PropertyType    string         `gorm:"size:64;index;not null" json:"property_type"`
	TransactionType string         `gorm:"size:32;index;not null" json:"transaction_type"`
	Price           float64        `gorm:"not null" json:"price"`
	City            string         `gorm:"size:128;index;not null" json:"city"`
	Location        string         `gorm:"size:255" json:"location"`
	Bedrooms        int            `gorm:"default:0" json:"bedrooms"`
	Bathrooms       int            `gorm:"default:0" json:"bathrooms"`
	AreaSqFt        float64        `gorm:"default:0" json:"area_sqft"`
	Amenities       string         `gorm:"type:text" json:"amenities,omitempty"`
	MediaURLs       string         `gorm:"type:text" json:"media_urls,omitempty"`
	Status          string         `gorm:"size:32;default:'AVAILABLE';index" json:"status"` // AVAILABLE, RESERVED, SOLD
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// SiteVisit represents a property viewing appointment with a prospective buyer.
type SiteVisit struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	OrganizationID uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID uint           `gorm:"index;default:3" json:"business_unit_id"` // 3: Real Estate
	PropertyID     uint           `gorm:"index;not null" json:"property_id"`
	Property       *Property      `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	BuyerID        uint           `gorm:"index;not null" json:"buyer_id"` // Customer ID
	AgentID        uint           `gorm:"index;not null" json:"agent_id"` // User ID
	ScheduledAt    time.Time      `gorm:"index;not null" json:"scheduled_at"`
	Status         string         `gorm:"size:32;default:'SCHEDULED';index" json:"status"` // SCHEDULED, COMPLETED, CANCELLED
	Notes          string         `gorm:"type:text" json:"notes,omitempty"`
	TaskID         *uint          `gorm:"index" json:"task_id,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// PropertyDeal tracks negotiation, deposit, and closing of a property.
type PropertyDeal struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	OrganizationID    uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID    uint           `gorm:"index;default:3" json:"business_unit_id"` // 3: Real Estate
	PropertyID        uint           `gorm:"index;not null" json:"property_id"`
	Property          *Property      `gorm:"foreignKey:PropertyID" json:"property,omitempty"`
	BuyerID           uint           `gorm:"index;not null" json:"buyer_id"`
	SellerID          uint           `gorm:"index;not null" json:"seller_id"`
	AgreedPrice       float64        `gorm:"not null" json:"agreed_price"`
	CommissionRate    float64        `gorm:"default:2.0" json:"commission_rate"` // e.g. 2%
	CommissionAmount  float64        `gorm:"default:0" json:"commission_amount"`
	JourneyInstanceID *uint          `gorm:"index" json:"journey_instance_id,omitempty"`
	Status            string         `gorm:"size:32;default:'OFFER';index" json:"status"` // OFFER, NEGOTIATION, BOOKED, CLOSED
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// AutoMigrate migrates real estate domain models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Property{}, &SiteVisit{}, &PropertyDeal{})
}
