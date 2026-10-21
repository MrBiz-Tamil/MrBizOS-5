//go:build ignore

package models

import (
	"time"
)

// Ticket represents a customer support ticket
type Ticket struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	UserID    uint            `gorm:"index" json:"user_id"`
	Subject   string          `json:"subject"`
	Status    string          `gorm:"default:'open'" json:"status"`     // open, in_progress, resolved, closed
	Priority  string          `gorm:"default:'normal'" json:"priority"` // low, normal, high, urgent
	Messages  []TicketMessage `gorm:"foreignKey:TicketID" json:"messages"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (Ticket) TableName() string {
	return "tickets"
}

// TicketMessage represents a single message within a support ticket
type TicketMessage struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TicketID  uint      `gorm:"index" json:"ticket_id"`
	SenderID  uint      `json:"sender_id"`
	Message   string    `json:"message"`
	IsStaff   bool      `gorm:"default:false" json:"is_staff"`
	CreatedAt time.Time `json:"created_at"`
}

func (TicketMessage) TableName() string {
	return "ticket_messages"
}
