//go:build ignore
// +build ignore

package models

import (
	"time"
)

type StaffProfile struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	User             User      `gorm:"foreignKey:UserID" json:"user"`
	EmployeeID       string    `gorm:"uniqueIndex;not null" json:"employee_id"`
	Department       string    `json:"department"`
	Designation      string    `json:"designation"`
	JoiningDate      time.Time `json:"joining_date"`
	BaseSalary       int64     `json:"base_salary"` // In cents
	ManagerID        *uint     `json:"manager_id"`
	Manager          *User     `gorm:"foreignKey:ManagerID" json:"manager,omitempty"`
	EmploymentStatus string    `gorm:"default:'active'" json:"employment_status"` // active, on_leave, terminated
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (StaffProfile) TableName() string {
	return "staff_profiles"
}

type StaffShift struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	HospitalID uint      `gorm:"index" json:"hospital_id"`
	Name       string    `json:"name"`       // e.g. "Morning Shift", "Night Shift"
	StartTime  string    `json:"start_time"` // "08:00"
	EndTime    string    `json:"end_time"`   // "16:00"
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (StaffShift) TableName() string {
	return "staff_shifts"
}

type StaffAttendance struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"user_id"`
	ShiftID    uint       `gorm:"index" json:"shift_id"`
	Date       time.Time  `gorm:"type:date" json:"date"`
	CheckIn    *time.Time `json:"check_in"`
	CheckOut   *time.Time `json:"check_out"`
	Status     string     `gorm:"default:'present'" json:"status"` // present, late, absent, half_day
	IsVerified bool       `gorm:"default:false" json:"is_verified"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (StaffAttendance) TableName() string {
	return "staff_attendance"
}

type PayrollRecord struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"index;not null" json:"user_id"`
	Month         int        `json:"month"`
	Year          int        `json:"year"`
	BaseAmount    int64      `json:"base_amount"`
	Bonus         int64      `json:"bonus"`
	Deductions    int64      `json:"deductions"`
	NetAmount     int64      `json:"net_amount"`
	Status        string     `gorm:"default:'pending'" json:"status"` // pending, processed, paid, failed
	PaymentDate   *time.Time `json:"payment_date"`
	TransactionID string     `json:"transaction_id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (PayrollRecord) TableName() string {
	return "payroll_records"
}
