package travel

import (
	"time"

	"gorm.io/gorm"
)

// Destination represents a travel location.
type Destination struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Country     string         `gorm:"size:128;not null;index" json:"country"`
	CountryCode string         `gorm:"size:8;index" json:"country_code"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	ImageURL    string         `gorm:"size:512" json:"image_url,omitempty"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TourPackage represents a curated tour itinerary.
type TourPackage struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	DestinationID uint           `gorm:"index;not null" json:"destination_id"`
	Destination   *Destination   `gorm:"foreignKey:DestinationID" json:"destination,omitempty"`
	Title         string         `gorm:"size:255;not null" json:"title"`
	Slug          string         `gorm:"size:255;uniqueIndex" json:"slug"`
	Days          int            `gorm:"not null" json:"days"`
	Nights        int            `gorm:"not null" json:"nights"`
	Price         float64        `gorm:"not null" json:"price"`
	Highlights    string         `gorm:"type:text" json:"highlights,omitempty"`
	Inclusions    string         `gorm:"type:text" json:"inclusions,omitempty"`
	Exclusions    string         `gorm:"type:text" json:"exclusions,omitempty"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TravelBooking represents a customer tour booking.
type TravelBooking struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	OrganizationID    uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID    uint           `gorm:"index;default:1" json:"business_unit_id"` // 1: Travel
	CustomerID        uint           `gorm:"index;not null" json:"customer_id"`
	TourPackageID     uint           `gorm:"index;not null" json:"tour_package_id"`
	TourPackage       *TourPackage   `gorm:"foreignKey:TourPackageID" json:"tour_package,omitempty"`
	JourneyInstanceID *uint          `gorm:"index" json:"journey_instance_id,omitempty"`
	TravelDate        time.Time      `json:"travel_date"`
	Passengers        int            `gorm:"default:1" json:"passengers"`
	TotalPrice        float64        `gorm:"not null" json:"total_price"`
	PaidAmount        float64        `gorm:"default:0" json:"paid_amount"`
	Status            string         `gorm:"size:32;default:'CONFIRMED';index" json:"status"` // CONFIRMED, CANCELLED, COMPLETED
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// VisaChecklist defines required documents & fees per country/visa type.
type VisaChecklist struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	CountryCode       string    `gorm:"size:8;index;not null" json:"country_code"`
	CountryName       string    `gorm:"size:128;not null" json:"country_name"`
	VisaType          string    `gorm:"size:64;index;not null" json:"visa_type"` // TOURIST, BUSINESS, WORK, STUDENT
	FeeAmount         float64   `gorm:"default:0" json:"fee_amount"`
	ProcessingDays    int       `gorm:"default:5" json:"processing_days"`
	RequiredDocuments string    `gorm:"type:text" json:"required_documents"` // Comma-separated or JSON list
	CreatedAt         time.Time `json:"created_at"`
}

// VisaApplication tracks a customer's visa processing workflow.
type VisaApplication struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	OrganizationID    uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID    uint           `gorm:"index;default:1" json:"business_unit_id"` // 1: Travel & Visa
	ApplicantID       uint           `gorm:"index;not null" json:"applicant_id"`       // Customer ID
	CountryCode       string         `gorm:"size:8;index;not null" json:"country_code"`
	VisaType          string         `gorm:"size:64;not null" json:"visa_type"`
	PassportNumber    string         `gorm:"size:64;index" json:"passport_number,omitempty"`
	JourneyInstanceID *uint          `gorm:"index" json:"journey_instance_id,omitempty"`
	Status            string         `gorm:"size:32;default:'SUBMITTED';index" json:"status"` // SUBMITTED, IN_REVIEW, APPROVED, REJECTED
	AppointmentDate   *time.Time     `json:"appointment_date,omitempty"`
	DecisionDate      *time.Time     `json:"decision_date,omitempty"`
	Notes             string         `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// AutoMigrate migrates travel & visa domain models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Destination{}, &TourPackage{}, &TravelBooking{}, &VisaChecklist{}, &VisaApplication{})
}
