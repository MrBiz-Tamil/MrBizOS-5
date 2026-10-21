//go:build ignore
// +build ignore

package models

import (
	"drayya-core/internal/domain/identity"
	"time"

	"gorm.io/gorm"
)

// Blog represents a Siddha Medical Journal or health article written by a doctor.
type Blog struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Title      string         `gorm:"not null" json:"title"`
	Excerpt    string         `json:"excerpt"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	Category   string         `gorm:"index;not null" json:"category"`
	AuthorID   uint           `gorm:"index;not null" json:"author_id"`
	Author     identity.User  `gorm:"foreignKey:AuthorID" json:"author"`
	Status     string         `gorm:"default:'pending';index" json:"status"` // 'pending', 'approved', 'rejected'
	ApproverID *uint          `json:"approver_id"`
	ImageURL   string         `json:"image_url"`
	ReadTime   string         `json:"read_time"`
	IsFeatured bool           `gorm:"default:false" json:"is_featured"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Blog) TableName() string {
	return "blogs"
}
