package consulting

import (
	"time"

	"gorm.io/gorm"
)

// ClientAccount represents a corporate B2B IT consulting client.
type ClientAccount struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	OrganizationID uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID uint           `gorm:"index;default:5" json:"business_unit_id"` // 5: Software Consulting
	CompanyName    string         `gorm:"size:128;not null" json:"company_name"`
	PrimaryContact string         `gorm:"size:128" json:"primary_contact"`
	Email          string         `gorm:"size:128;index" json:"email"`
	Phone          string         `gorm:"size:32" json:"phone"`
	Industry       string         `gorm:"size:64" json:"industry,omitempty"`
	Status         string         `gorm:"size:32;default:'ACTIVE';index" json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// SoftwareProject represents an SOW software development contract.
type SoftwareProject struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	OrganizationID uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID uint           `gorm:"index;default:5" json:"business_unit_id"` // 5: Software Consulting
	ClientID       uint           `gorm:"index;not null" json:"client_id"`
	Client         *ClientAccount `gorm:"foreignKey:ClientID" json:"client,omitempty"`
	Title          string         `gorm:"size:255;not null" json:"title"`
	Code           string         `gorm:"size:64;uniqueIndex" json:"code"` // e.g. "PRJ-MED-V2"
	Budget         float64        `gorm:"not null" json:"budget"`
	Spent          float64        `gorm:"default:0" json:"spent"`
	Status         string         `gorm:"size:32;default:'PLANNING';index" json:"status"` // PLANNING, IN_PROGRESS, UAT, COMPLETED
	StartDate      *time.Time     `json:"start_date,omitempty"`
	TargetDate     *time.Time     `json:"target_date,omitempty"`
	AMCExpiry      *time.Time     `json:"amc_expiry,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// ProjectMilestone represents a contractual deliverable milestone.
type ProjectMilestone struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	ProjectID   uint       `gorm:"index;not null" json:"project_id"`
	Title       string     `gorm:"size:255;not null" json:"title"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
	Amount      float64    `gorm:"not null" json:"amount"`
	Status      string     `gorm:"size:32;default:'PENDING';index" json:"status"` // PENDING, COMPLETED, INVOICED
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Timesheet tracks developer billable hours.
type Timesheet struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ProjectID   uint      `gorm:"index;not null" json:"project_id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	WorkDate    time.Time `gorm:"index;not null" json:"work_date"`
	Hours       float64   `gorm:"not null" json:"hours"`
	Description string    `gorm:"type:text" json:"description"`
	Billable    bool      `gorm:"default:true" json:"billable"`
	CreatedAt   time.Time `json:"created_at"`
}

// AutoMigrate migrates software consulting domain models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&ClientAccount{}, &SoftwareProject{}, &ProjectMilestone{}, &Timesheet{})
}
