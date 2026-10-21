package hr

import (
	"time"
)

type OnboardingRecord struct {
	ID      uint `gorm:"primaryKey" json:"id"`
	StaffID uint `gorm:"uniqueIndex;not null" json:"staff_id"`

	// Linked Profile
	StaffProfile StaffProfile `gorm:"foreignKey:StaffID" json:"staff_profile,omitempty"`

	// Required Dr.Ayya Documents (Boolean or Status Strings: pending, submitted, verified)
	Certificate   string `gorm:"default:'pending'" json:"certificate"`
	AadharID      string `gorm:"default:'pending'" json:"aadhar_id"`
	PanCard       string `gorm:"default:'pending'" json:"pan_card"`
	BankAccount   string `gorm:"default:'pending'" json:"bank_account"`
	PassportPhoto string `gorm:"default:'pending'" json:"passport_photo"`

	// Orientation Tracking
	SystemAccess     bool `gorm:"default:false" json:"system_access_granted"`
	HandbookRead     bool `gorm:"default:false" json:"handbook_read"`
	WelcomeEmailSent bool `gorm:"default:false" json:"welcome_email_sent"`

	// Timeline
	ProbationEndDate time.Time `json:"probation_end_date"`
	Status           string    `gorm:"default:'pending_docs'" json:"status"` // pending_docs, in_training, completed
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (OnboardingRecord) TableName() string {
	return "hr_onboarding_records"
}
