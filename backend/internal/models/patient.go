//go:build ignore
// +build ignore

package models

import (
	"time"

	"gorm.io/gorm"
)

type PatientProfile struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            uint           `gorm:"uniqueIndex" json:"user_id"`
	DateOfBirth       *time.Time     `json:"date_of_birth"` // Use pointer for nullable date
	Gender            string         `json:"gender"`
	BloodGroup        string         `json:"blood_group"`
	Height            float64        `json:"height"`                     // in cm
	Weight            float64        `json:"weight"`                     // in kg
	Allergies         string         `gorm:"type:text" json:"allergies"` // Comma-separated or JSON
	ChronicConditions string         `gorm:"type:text" json:"chronic_conditions"`
	EmergencyContact  string         `json:"emergency_contact"`
	Address           string         `json:"address"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (PatientProfile) TableName() string {
	return "patient_profiles"
}

// Age helper method could be added but GORM models are data structs.
