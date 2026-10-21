package task

import (
	"time"

	"gorm.io/gorm"
)

// Task Statuses
const (
	StatusTodo        = "TODO"
	StatusInProgress  = "IN_PROGRESS"
	StatusBlocked     = "BLOCKED"
	StatusReview      = "REVIEW"
	StatusCompleted   = "COMPLETED"
	StatusCancelled   = "CANCELLED"
)

// Task Priorities
const (
	PriorityLow    = "LOW"
	PriorityMedium = "MEDIUM"
	PriorityHigh   = "HIGH"
	PriorityUrgent = "URGENT"
)

// Task represents a central action item across all 5 Business Units.
type Task struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	OrganizationID uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID uint           `gorm:"index;default:2" json:"business_unit_id"` // 1:Travel, 2:Health, 3:RealEstate, 4:Ecomm, 5:Software
	CustomerID     *uint          `gorm:"index" json:"customer_id,omitempty"`
	LeadID         *uint          `gorm:"index" json:"lead_id,omitempty"`
	JourneyID      *uint          `gorm:"index" json:"journey_id,omitempty"`
	EntityResource string         `gorm:"size:64;index" json:"entity_resource,omitempty"` // e.g. "appointment", "property", "visa_application", "sow"
	EntityID       *uint          `gorm:"index" json:"entity_id,omitempty"`
	OwnerID        uint           `gorm:"index;not null" json:"owner_id"`
	AssigneeID     *uint          `gorm:"index" json:"assignee_id,omitempty"`
	Team           string         `gorm:"size:64;index" json:"team,omitempty"`
	Title          string         `gorm:"size:255;not null" json:"title"`
	Description    string         `gorm:"type:text" json:"description,omitempty"`
	Status         string         `gorm:"size:32;default:'TODO';index" json:"status"`
	Priority       string         `gorm:"size:32;default:'MEDIUM';index" json:"priority"`
	StartDate      *time.Time     `json:"start_date,omitempty"`
	DueDate        *time.Time     `gorm:"index" json:"due_date,omitempty"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	Tags           string         `gorm:"size:255" json:"tags,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// TaskComment represents a discussion comment on a task.
type TaskComment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    uint      `gorm:"index;not null" json:"task_id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Comment   string    `gorm:"type:text;not null" json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// AutoMigrate migrates the task domain models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Task{}, &TaskComment{})
}
