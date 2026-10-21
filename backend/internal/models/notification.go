//go:build ignore
// +build ignore

package models

import (
	"drayya-core/internal/domain/identity"
	"time"

	"gorm.io/gorm"
)

type NotificationTemplate struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	TemplateType   string         `gorm:"uniqueIndex;not null" json:"template_type"`
	MessageContent string         `gorm:"type:text;not null" json:"message_content"`
	Variables      string         `json:"variables"` // Comma separated list of vars e.g. "name,date"
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (NotificationTemplate) TableName() string {
	return "notification_templates"
}

type NotificationLog struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"index" json:"user_id"`
	AppointmentID    *uint     `gorm:"index" json:"appointment_id,omitempty"`
	NotificationType string    `json:"notification_type"` // sms, whatsapp, email
	TemplateType     string    `json:"template_type"`
	Status           string    `json:"status"` // sent, failed
	ErrorMessage     *string   `json:"error_message,omitempty"`
	SentAt           time.Time `json:"sent_at"`

	// Relations
	User *identity.User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (NotificationLog) TableName() string {
	return "notification_logs"
}

// Notification represents a user notification
type Notification struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Title     string    `gorm:"not null" json:"title"`
	Message   string    `gorm:"type:text;not null" json:"message"`
	Type      string    `gorm:"not null" json:"type"` // info, success, warning, error
	IsRead    bool      `gorm:"default:false" json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	User *identity.User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Notification) TableName() string {
	return "notifications"
}
