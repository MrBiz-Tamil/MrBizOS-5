package budget

import (
	"time"

	"gorm.io/gorm"
)

// BudgetPeriod is a named financial period for budgeting.
type BudgetPeriod struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"size:100;not null" json:"name"` // e.g. "FY 2026-27"
	StartDate time.Time      `json:"start_date"`
	EndDate   time.Time      `json:"end_date"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
}

// BudgetLine is one line item in a budget period.
type BudgetLine struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	PeriodID        uint      `gorm:"index;not null" json:"period_id"`
	Category        string    `gorm:"size:100;not null" json:"category"`   // e.g. "Marketing", "Infrastructure"
	SubCategory     string    `gorm:"size:100" json:"sub_category,omitempty"`
	AllocatedPaise  int64     `json:"allocated_paise"`
	SpentPaise      int64     `json:"spent_paise"`
	CommittedPaise  int64     `json:"committed_paise"` // approved but not yet spent
	OwnerUserID     uint      `json:"owner_user_id,omitempty"`
	Notes           string    `gorm:"type:text" json:"notes,omitempty"`
}

// RemainingPaise returns free budget.
func (b *BudgetLine) RemainingPaise() int64 {
	return b.AllocatedPaise - b.SpentPaise - b.CommittedPaise
}

// UtilizationPct returns % of budget consumed.
func (b *BudgetLine) UtilizationPct() float64 {
	if b.AllocatedPaise == 0 {
		return 0
	}
	return float64(b.SpentPaise) / float64(b.AllocatedPaise) * 100
}

// BudgetExpense is an actual expenditure record posted against a budget line.
type BudgetExpense struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	LineID      uint      `gorm:"index;not null" json:"line_id"`
	AmountPaise int64     `json:"amount_paise"`
	Description string    `gorm:"size:300" json:"description"`
	VendorName  string    `gorm:"size:200" json:"vendor_name,omitempty"`
	InvoiceNo   string    `gorm:"size:100" json:"invoice_no,omitempty"`
	ExpenseDate time.Time `json:"expense_date"`
	SubmittedBy uint      `json:"submitted_by"`
	ApprovedBy  *uint     `json:"approved_by,omitempty"`
	Status      string    `gorm:"size:20;default:'pending'" json:"status"` // pending, approved, rejected
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&BudgetPeriod{}, &BudgetLine{}, &BudgetExpense{})
}
