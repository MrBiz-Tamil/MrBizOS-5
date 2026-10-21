package billing

import (
	"drayya-core/internal/database"

	"gorm.io/gorm"
)

type Repository interface {
	CreateBilling(billing *Billing) error
	GetBillingByAppointment(appointmentID uint) (*Billing, error)
	GetBillingByID(id uint) (*Billing, error)
	UpdateBilling(billing *Billing) error
	CreatePayment(payment *Payment) error
	GetPaymentByRazorpayID(razorpayPaymentID string) (*Payment, error)
	ExecuteTransaction(fn func(txRepo Repository) error) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Default instance using global DB
func NewDefaultRepository() Repository {
	return &repository{db: database.DB}
}

func (r *repository) CreateBilling(billing *Billing) error {
	return r.db.Create(billing).Error
}

func (r *repository) GetBillingByAppointment(appointmentID uint) (*Billing, error) {
	var billing Billing
	if err := r.db.Where("appointment_id = ?", appointmentID).First(&billing).Error; err != nil {
		return nil, err
	}
	return &billing, nil
}

func (r *repository) GetBillingByID(id uint) (*Billing, error) {
	var billing Billing
	if err := r.db.First(&billing, id).Error; err != nil {
		return nil, err
	}
	return &billing, nil
}

func (r *repository) UpdateBilling(billing *Billing) error {
	return r.db.Save(billing).Error
}

func (r *repository) CreatePayment(payment *Payment) error {
	return r.db.Create(payment).Error
}

func (r *repository) GetPaymentByRazorpayID(razorpayPaymentID string) (*Payment, error) {
	var p Payment
	if err := r.db.Where("razorpay_payment_id = ?", razorpayPaymentID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) ExecuteTransaction(fn func(txRepo Repository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &repository{db: tx}
		return fn(txRepo)
	})
}
