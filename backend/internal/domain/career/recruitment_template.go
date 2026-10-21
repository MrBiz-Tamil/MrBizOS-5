package career

import (
	"time"

	"gorm.io/gorm"
)

type RecruitmentTemplate struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Role      string         `gorm:"not null" json:"role"` // Doctors, HR, Finance, All Users
	Type      string         `gorm:"not null" json:"type"` // e.g. Offer Letter, Policy, SMS, WhatsApp
	Title     string         `gorm:"not null" json:"title"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	Variables string         `json:"variables"` // comma-separated e.g. "name,position"
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (RecruitmentTemplate) TableName() string {
	return "recruitment_templates"
}
