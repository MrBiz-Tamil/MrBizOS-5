package refill

import (
	"drayya-core/internal/domain/identity"
	"time"
)

// RefillSchedule represents an auto-refill configuration for a patient's medication
type RefillSchedule struct {
	ID               uint          `gorm:"primaryKey" json:"id"`
	UserID           uint          `gorm:"index;not null" json:"user_id"`
	MedicineName     string        `gorm:"not null" json:"medicine_name"`
	Dosage           string        `json:"dosage"`
	Frequency        string        `json:"frequency"` // Daily, Weekly, Monthly, etc.
	QuantityPerRefill int          `gorm:"default:30" json:"quantity_per_refill"`
	NextRefillDate   time.Time     `json:"next_refill_date"`
	AutoOrder        bool          `gorm:"default:true" json:"auto_order"`
	Status           string        `gorm:"default:'active'" json:"status"` // active, paused, cancelled
	LastFilledAt     *time.Time    `json:"last_filled_at,omitempty"`
	PrescriptionID   string        `json:"prescription_id,omitempty"`
	PharmacyID       *uint         `json:"pharmacy_id,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`

	User identity.User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (RefillSchedule) TableName() string {
	return "refill_schedules"
}

// RefillOrder represents an auto-generated pharmacy order from a refill schedule
type RefillOrder struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ScheduleID uint      `gorm:"index;not null" json:"schedule_id"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`
	OrderID    *uint     `json:"order_id,omitempty"` // linked pharmacy order
	Status     string    `gorm:"default:'pending'" json:"status"` // pending, placed, failed, cancelled
	Quantity   int       `json:"quantity"`
	Note       string    `json:"note,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (RefillOrder) TableName() string {
	return "refill_orders"
}
