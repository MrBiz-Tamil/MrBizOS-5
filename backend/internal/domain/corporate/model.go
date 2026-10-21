package corporate

import (
	"time"

	"gorm.io/gorm"
)

type Company struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"size:200;not null" json:"name"`
	Domain    string         `gorm:"size:100;uniqueIndex" json:"domain"`
	WalletBalance float64    `json:"wallet_balance"` // Corporate allowance balance
}

type Employee struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	CompanyID uint           `gorm:"index" json:"company_id"`
	Company   Company        `gorm:"foreignKey:CompanyID" json:"company"`
	UserID    uint           `gorm:"index;unique" json:"user_id"` // Link to Identity user
	EmployeeID string        `gorm:"size:100" json:"employee_id"` // Company specific ID
	WalletAllowance float64  `json:"wallet_allowance"` // Individual allowance allocated by company
	IsActive  bool           `gorm:"default:true" json:"is_active"`
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Company{}, &Employee{}, &CompanyCreditLedger{}, &CorporateInvoice{})
}

// CompanyCreditLedger tracks top-ups and usage of corporate wallet.
type CompanyCreditLedger struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CompanyID   uint      `gorm:"index;not null" json:"company_id"`
	AmountPaise int64     `json:"amount_paise"` // positive = top-up, negative = deduction
	Description string    `json:"description"`
	RefType     string    `json:"ref_type"` // topup, consultation, medicine, lab
	RefID       uint      `json:"ref_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func (CompanyCreditLedger) TableName() string { return "company_credit_ledger" }

// CorporateInvoice is a consolidated monthly invoice for the company.
type CorporateInvoice struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	CompanyID           uint       `gorm:"index;not null" json:"company_id"`
	Month               string     `json:"month"` // "2026-03"
	TotalConsultations  int        `json:"total_consultations"`
	TotalMedicineOrders int        `json:"total_medicine_orders"`
	TotalLabOrders      int        `json:"total_lab_orders"`
	TotalAmountPaise    int64      `json:"total_amount_paise"`
	Status              string     `gorm:"default:'pending'" json:"status"` // pending, paid, overdue
	DueDate             time.Time  `json:"due_date"`
	PaidAt              *time.Time `json:"paid_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (CorporateInvoice) TableName() string { return "corporate_invoices" }
