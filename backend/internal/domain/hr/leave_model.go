package hr

import (
	"time"

	"gorm.io/gorm"
)

// LeaveType defines categories of leave available in the organisation.
type LeaveType struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Name          string         `gorm:"size:100;not null;uniqueIndex" json:"name"`   // e.g. "Annual Leave"
	Code          string         `gorm:"size:20;not null;uniqueIndex" json:"code"`    // e.g. "AL"
	MaxDaysPerYear int           `json:"max_days_per_year"`
	IsPaid        bool           `gorm:"default:true" json:"is_paid"`
	CarryForward  bool           `gorm:"default:false" json:"carry_forward"`
	MaxCarryDays  int            `json:"max_carry_days"`
}

// LeaveBalance tracks accrued / used balances per employee per year.
type LeaveBalance struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	LeaveTypeID uint      `gorm:"index;not null" json:"leave_type_id"`
	LeaveType   LeaveType `gorm:"foreignKey:LeaveTypeID" json:"leave_type"`
	Year        int       `gorm:"not null" json:"year"` // e.g. 2026
	Allocated   float64   `json:"allocated"`
	Used        float64   `json:"used"`
	Pending     float64   `json:"pending"` // applied but not yet approved
	CarriedOver float64   `json:"carried_over"`
}

// Remaining returns remaining available days.
func (b *LeaveBalance) Remaining() float64 {
	return b.Allocated + b.CarriedOver - b.Used - b.Pending
}

// LeaveRequest is a single leave application by an employee.
type LeaveRequest struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	UserID      uint           `gorm:"index;not null" json:"user_id"`
	LeaveTypeID uint           `gorm:"index;not null" json:"leave_type_id"`
	LeaveType   LeaveType      `gorm:"foreignKey:LeaveTypeID" json:"leave_type"`
	StartDate   time.Time      `json:"start_date"`
	EndDate     time.Time      `json:"end_date"`
	TotalDays   float64        `json:"total_days"`
	HalfDay     bool           `json:"half_day"`
	Reason      string         `gorm:"type:text" json:"reason"`
	Status      string         `gorm:"size:20;default:'pending'" json:"status"` // pending, approved, rejected, cancelled
	ApprovedBy  *uint          `json:"approved_by,omitempty"`
	ApprovedAt  *time.Time     `json:"approved_at,omitempty"`
	RejectedBy  *uint          `json:"rejected_by,omitempty"`
	RejectReason string        `gorm:"type:text" json:"reject_reason,omitempty"`
}

// HolidayCalendar lists public / company holidays for a given year.
type HolidayCalendar struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Date        time.Time `gorm:"index;not null" json:"date"`
	Name        string    `gorm:"size:200;not null" json:"name"`
	IsOptional  bool      `gorm:"default:false" json:"is_optional"` // restricted holiday
	Year        int       `gorm:"index;not null" json:"year"`
}

func (HolidayCalendar) TableName() string { return "holiday_calendar" }
