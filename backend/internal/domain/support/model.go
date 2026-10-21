package support

import (
	"time"

	"gorm.io/gorm"
)

// Ticket is the top-level support ticket.
type Ticket struct {
	ID          uint             `gorm:"primaryKey"        json:"id"`
	UserID      uint             `gorm:"index"             json:"user_id"`
	Subject     string           `json:"subject"`
	Description string           `gorm:"type:text"         json:"description"` // copy of first message
	Category    string           `gorm:"default:'general'" json:"category"`    // billing, medical, pharmacy, technical, general
	Department  string           `gorm:"default:'general'" json:"department"`  // set by AI classifier
	Priority    string           `gorm:"default:'normal'"  json:"priority"`    // low, normal, high, urgent
	Status      string           `gorm:"default:'open'"    json:"status"`      // open, in_progress, resolved, closed
	AssignedTo  *uint            `gorm:"index"             json:"assigned_to"` // staff user ID
	AIHandled   bool             `gorm:"default:false"     json:"ai_handled"`  // true = AI replied successfully
	Messages    []TicketMessage  `gorm:"foreignKey:TicketID" json:"messages,omitempty"`
	ClosedAt    *time.Time       `json:"closed_at,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

func (Ticket) TableName() string { return "tickets" }

// TicketMessage is a single message in a ticket thread.
type TicketMessage struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TicketID    uint      `gorm:"index"      json:"ticket_id"`
	SenderID    uint      `json:"sender_id"`
	SenderName  string    `json:"sender_name"` // "Dr.Ayya Support Bot", staff name, or blank
	Message     string    `gorm:"type:text"  json:"message"`
	IsStaff     bool      `gorm:"default:false" json:"is_staff"`
	IsAIReply   bool      `gorm:"default:false" json:"is_ai_reply"`
	CreatedAt   time.Time `json:"created_at"`
}

func (TicketMessage) TableName() string { return "ticket_messages" }

// Valid departments / categories
var ValidDepartments = map[string]bool{
	"billing":   true,
	"medical":   true,
	"pharmacy":  true,
	"technical": true,
	"general":   true,
	"hr":        true,
	"logistics": true,
}

// DeptEmail returns the env-var name for the department notification email.
func DeptEmail(dept string) string {
	switch dept {
	case "billing":
		return "SUPPORT_BILLING_EMAIL"
	case "medical":
		return "SUPPORT_MEDICAL_EMAIL"
	case "pharmacy":
		return "SUPPORT_PHARMACY_EMAIL"
	case "technical":
		return "SUPPORT_TECH_EMAIL"
	case "hr":
		return "SUPPORT_HR_EMAIL"
	case "logistics":
		return "SUPPORT_LOGISTICS_EMAIL"
	default:
		return "SUPPORT_GENERAL_EMAIL"
	}
}

// TicketAuditLog is an immutable record of a status or assignment change.
type TicketAuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TicketID  uint      `gorm:"index"      json:"ticket_id"`
	Action    string    `json:"action"`     // status_changed, assigned, department_changed, escalated
	OldValue  string    `json:"old_value"`  // prior state/id
	NewValue  string    `json:"new_value"`  // new state/id
	ActorID   uint      `json:"actor_id"`   // Who made the change (0 = system/AI)
	CreatedAt time.Time `json:"created_at"`
}

func (TicketAuditLog) TableName() string { return "ticket_audit_logs" }

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Ticket{}, &TicketMessage{}, &TicketAuditLog{})
}
