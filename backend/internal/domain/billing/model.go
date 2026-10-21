package billing

import (
	"time"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Billing{}, &Payment{})
}

type Billing struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	OrganizationID uint      `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID uint      `gorm:"index;default:2" json:"business_unit_id"`
	AppointmentID *uint     `json:"appointment_id"`
	PatientID     uint      `json:"patient_id"`
	Amount        int64     `json:"amount"`                                  // In paise (1 INR = 100 paise)
	Currency      string    `gorm:"default:'INR'" json:"currency"`
	PaymentStatus string    `gorm:"default:'pending'" json:"payment_status"` // pending, paid, failed, refunded
	PaymentMethod string    `json:"payment_method"`
	InvoiceDate   time.Time `json:"invoice_date"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Billing) TableName() string {
	return "billing"
}

type Payment struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	OrganizationID        uint      `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID        uint      `gorm:"index;default:2" json:"business_unit_id"`
	AppointmentID         *uint     `json:"appointment_id"`
	InvoiceID             *uint     `json:"invoice_id"`
	PatientID             uint      `json:"patient_id"`
	Amount                int64     `json:"amount"`
	ConsultationFee       int64     `json:"consultation_fee"`
	PlatformFeePercentage float64   `json:"platform_fee_percentage"`
	PlatformCommission    int64     `json:"platform_commission"`
	DoctorEarnings        int64     `json:"doctor_earnings"`
	PaymentMethod         string    `json:"payment_method"`
	RazorpayPaymentID     string    `json:"razorpay_payment_id"`
	RazorpayOrderID       string    `json:"razorpay_order_id"`
	Status                string    `json:"status"` // completed, failed
	PaidAt                time.Time `json:"paid_at"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (Payment) TableName() string {
	return "payment"
}
