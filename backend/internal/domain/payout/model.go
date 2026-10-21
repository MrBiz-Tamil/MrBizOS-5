package payout

import (
	"time"

	"gorm.io/gorm"
)

// AutoMigrate creates the payout_records table.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Record{})
}

// Record tracks a single Razorpay payout issued to a doctor.
// One record is created per doctor per payout run (or on-demand trigger).
type Record struct {
	ID               uint      `gorm:"primaryKey"        json:"id"`
	DoctorUserID     uint      `gorm:"index;not null"    json:"doctor_user_id"`
	AmountPaise      int64     `json:"amount_paise"`
	Currency         string    `gorm:"default:'INR'"     json:"currency"`
	RazorpayPayoutID string    `json:"razorpay_payout_id"`
	RefID            string    `gorm:"uniqueIndex"       json:"ref_id"` // idempotency key
	// Status: processing | processed | failed
	Status           string    `gorm:"default:'processing'" json:"status"`
	Mode             string    `json:"mode"`              // IMPS | UPI | NEFT
	AppointmentCount int       `json:"appointment_count"` // number of appointments batched
	Notes            string    `gorm:"type:text"         json:"notes"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Record) TableName() string { return "payout_records" }
