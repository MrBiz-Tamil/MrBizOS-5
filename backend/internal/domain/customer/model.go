package customer

import (
	"time"

	"gorm.io/gorm"
)

// Lead represents a potential customer inquiry across any of the 5 Business Units.
type Lead struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	OrganizationID uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID uint           `gorm:"index;not null" json:"business_unit_id"`
	Name           string         `gorm:"size:128;not null" json:"name"`
	Email          string         `gorm:"size:128;index" json:"email"`
	Phone          string         `gorm:"size:32;index" json:"phone"`
	Source         string         `gorm:"size:64" json:"source,omitempty"` // Web, Campaign, Referral, WhatsApp
	Status         string         `gorm:"size:32;default:'NEW';index" json:"status"` // NEW, QUALIFIED, UNQUALIFIED, CONVERTED
	Stage          string         `gorm:"size:64;default:'Inquiry'" json:"stage"`
	Value          float64        `gorm:"default:0" json:"value"`
	OwnerID        *uint          `gorm:"index" json:"owner_id,omitempty"`
	Notes          string         `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// Customer360Profile represents a unified relationship record across the group.
type Customer360Profile struct {
	ID                    uint                   `gorm:"primaryKey" json:"id"`
	OrganizationID        uint                   `gorm:"index;default:1" json:"organization_id"`
	UserID                *uint                  `gorm:"uniqueIndex" json:"user_id,omitempty"`
	Name                  string                 `gorm:"size:128;not null" json:"name"`
	Email                 string                 `gorm:"size:128;index" json:"email"`
	Phone                 string                 `gorm:"size:32;index" json:"phone"`
	PrimaryBusinessUnitID uint                   `gorm:"default:2" json:"primary_business_unit_id"`
	ActiveRelationships   []BusinessRelationship `json:"active_relationships,omitempty"`
	Timeline              []TimelineEvent        `json:"timeline,omitempty"`
	TotalInvoiced         float64                `json:"total_invoiced"`
	TotalPaid             float64                `json:"total_paid"`
	OutstandingBalance    float64                `json:"outstanding_balance"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// BusinessRelationship tracks a customer's active engagement with a specific Business Unit.
type BusinessRelationship struct {
	BusinessUnitID   uint   `json:"business_unit_id"`
	BusinessUnitName string `json:"business_unit_name"`
	Status           string `json:"status"` // ACTIVE, INACTIVE, PROSPECT
	RelationshipID   string `json:"relationship_id"` // e.g. Patient ID, Client Account #, Buyer ID
}

// TimelineEvent is a unified interaction record in Customer 360.
type TimelineEvent struct {
	ID             string    `json:"id"`
	BusinessUnitID uint      `json:"business_unit_id"`
	EventType      string    `json:"event_type"` // CALL, EMAIL, APPOINTMENT, PAYMENT, TASK, JOURNEY
	Summary        string    `json:"summary"`
	Timestamp      time.Time `json:"timestamp"`
	IsClinical     bool      `json:"is_clinical"` // If true, requires healthcare.clinical.read permission
}

// AutoMigrate migrates customer & lead models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Lead{})
}
