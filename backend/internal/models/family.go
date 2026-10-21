//go:build ignore
// +build ignore

package models

import (
	"time"

	"gorm.io/gorm"
)

type FamilyMember struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	UserID            uint           `gorm:"not null" json:"user_id"`
	FullName          string         `gorm:"not null" json:"full_name"`
	Relationship      string         `gorm:"not null" json:"relationship"`
	DateOfBirth       *time.Time     `json:"date_of_birth"`
	Gender            string         `json:"gender"`
	BloodGroup        string         `json:"blood_group"`
	Allergies         string         `gorm:"type:text" json:"allergies"`
	MedicalConditions string         `gorm:"type:text" json:"medical_conditions"`
	ProfilePhotoURL   string         `json:"profile_photo_url"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (FamilyMember) TableName() string {
	return "family_members"
}
