package finance

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Account Types
const (
	TypeAsset     = "ASSET"
	TypeLiability = "LIABILITY"
	TypeEquity    = "EQUITY"
	TypeRevenue   = "REVENUE"
	TypeExpense   = "EXPENSE"
)

// Direction Constants
const (
	DirectionDebit  = "DEBIT"
	DirectionCredit = "CREDIT"
)

// EntryData for Recording Transactions
type EntryData struct {
	AccountID   uuid.UUID
	AccountName string
	Direction   string
	Amount      int64
}

// Revenue Leg Structs
type RevenueLeg struct {
	Account string
	Type    string
	Amount  int64
}

type RevenueInput struct {
	FundingAccount string
	FundingType    string
	Total          int64
	Credits        []RevenueLeg
}

type RecurringExpense struct {
	ID          uint
	Description string
	Amount      int64
	Category    string
	Department  string
	CreatedBy   uint
	Frequency   string
	DayOfMonth  int
	DayOfWeek   int
}

type Expense struct {
	Description string
	Amount      int64
	Category    string
	Department  string
	SourceID    string
	Status      string
	CreatedBy   uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func RecurringExpenseComputeNextDue(frequency string, dom int, dow int) *time.Time {
	now := time.Now()
	return &now
}



type Investor struct {
	ID        uint
	Name      string
	Email     string
	Amount    int64
	Equity    float64
	CreatedAt time.Time
}

// Ledger Entry Types (Phase 9)
const (
	EntryRevenue     = "REVENUE"
	EntryExpense     = "EXPENSE"
	EntryCOGS        = "COGS"
	EntryTax         = "TAX"
	EntryAdvertising = "ADVERTISING"
)

// LedgerAccount represents a double-entry ledger account.
type LedgerAccount struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string         `gorm:"size:255;uniqueIndex;not null" json:"name"`
	Type      string         `gorm:"size:32;not null" json:"type"`
	Balance   int64          `gorm:"default:0" json:"balance"` // in paise/cents
	Currency  string         `gorm:"size:8;default:'INR'" json:"currency"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// LedgerTransaction represents a multi-entry transaction batch.
type LedgerTransaction struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Type         string         `gorm:"size:64;index;not null" json:"type"`
	Description  string         `gorm:"type:text" json:"description"`
	RefID        string         `gorm:"size:128;index" json:"ref_id,omitempty"`
	ReferenceID  string         `gorm:"size:128;index" json:"reference_id,omitempty"`
	Status       string         `gorm:"size:32;default:'POSTED'" json:"status"`
	Hash         string         `gorm:"size:64" json:"hash,omitempty"`
	PreviousHash string         `gorm:"size:64" json:"previous_hash,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Entries      []LedgerEntry  `gorm:"foreignKey:TransactionID" json:"entries,omitempty"`
}

// LedgerEntry represents a single debit/credit leg of a ledger transaction.
type LedgerEntry struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	TransactionID uuid.UUID      `gorm:"type:uuid;index;not null" json:"transaction_id"`
	AccountID     uuid.UUID      `gorm:"type:uuid;index;not null" json:"account_id"`
	AccountName   string         `gorm:"size:255" json:"account_name,omitempty"`
	Direction     string         `gorm:"size:8;not null" json:"direction"` // DEBIT / CREDIT
	Amount        int64          `gorm:"not null" json:"amount"`          // in paise/cents
	BalanceAfter  int64          `gorm:"default:0" json:"balance_after"`
	Hash          string         `gorm:"size:64" json:"hash,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// PayoutRequest tracks doctor/vendor payout requests.
type PayoutRequest struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	Amount    int64          `gorm:"not null" json:"amount"`
	Status    string         `gorm:"size:32;default:'PENDING';index" json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TaxRule represents tax rules (GST/TDS).
type TaxRule struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Code       string    `gorm:"size:64;uniqueIndex" json:"code"`
	Percentage float64   `gorm:"not null" json:"percentage"`
	Region     string    `gorm:"size:64" json:"region"`
	Category   string    `gorm:"size:64" json:"category"`
	TaxType    string    `gorm:"size:64" json:"tax_type"`
	Rate       float64   `gorm:"not null" json:"rate"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// SeedTaxRules inserts default tax rules into DB.
func SeedTaxRules(db *gorm.DB) {
	var count int64
	db.Model(&TaxRule{}).Count(&count)
	if count == 0 {
		db.Create(&TaxRule{Code: "GST", Percentage: 18.0})
		db.Create(&TaxRule{Code: "TDS", Percentage: 10.0})
	}
}

// --- Phase 9 Group Intelligence & Profitability Structures ---

// CostCenter tracks budget allocation per business unit department.
type CostCenter struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	OrganizationID   uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID   uint           `gorm:"index;not null" json:"business_unit_id"`
	Code             string         `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Name             string         `gorm:"size:128;not null" json:"name"`
	BudgetAllocation float64        `gorm:"default:0" json:"budget_allocation"`
	YTDExpenses      float64        `gorm:"default:0" json:"ytd_expenses"`
	Status           string         `gorm:"size:32;default:'ACTIVE';index" json:"status"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// FinancialLedgerEntry records double-entry items for multi-BU cost-center accounting.
type FinancialLedgerEntry struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	OrganizationID uint           `gorm:"index;default:1" json:"organization_id"`
	BusinessUnitID uint           `gorm:"index;not null" json:"business_unit_id"`
	EntryType      string         `gorm:"size:32;index;not null" json:"entry_type"` // REVENUE, EXPENSE, COGS, ADVERTISING
	Amount         float64        `gorm:"not null" json:"amount"`
	Currency       string         `gorm:"size:8;default:'USD'" json:"currency"`
	ReferenceType  string         `gorm:"size:64" json:"reference_type,omitempty"`
	ReferenceID    uint           `gorm:"index" json:"reference_id,omitempty"`
	Description    string         `gorm:"type:text" json:"description"`
	EntryDate      time.Time      `gorm:"index;not null" json:"entry_date"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// BusinessUnitFinancialSummary summarizes financial performance for a single BU.
type BusinessUnitFinancialSummary struct {
	BusinessUnitID     uint    `json:"business_unit_id"`
	BusinessUnitName   string  `json:"business_unit_name"`
	GrossRevenue       float64 `json:"gross_revenue"`
	COGS               float64 `json:"cogs"`
	AdSpend            float64 `json:"ad_spend"`
	OperatingExpenses  float64 `json:"operating_expenses"`
	ContributionMargin float64 `json:"contribution_margin"`
	MarginPct          float64 `json:"margin_pct"`
}

// GroupProfitabilityReport compiles consolidated P&L across all business group units.
type GroupProfitabilityReport struct {
	ReportDate              time.Time                      `json:"report_date"`
	Currency                string                         `json:"currency"`
	TotalGrossRevenue       float64                        `json:"total_gross_revenue"`
	TotalGroupExpenses      float64                        `json:"total_group_expenses"`
	NetGroupProfit          float64                        `json:"net_group_profit"`
	NetGroupProfitMargin    float64                        `json:"net_group_profit_margin_pct"`
	BusinessUnitSummaries   []BusinessUnitFinancialSummary `json:"business_unit_summaries"`
	CustomerAcquisitionCost float64                        `json:"average_cac"`
	CustomerLifetimeValue   float64                        `json:"average_ltv"`
}

// AutoMigrate migrates financial domain models.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&LedgerAccount{},
		&LedgerTransaction{},
		&LedgerEntry{},
		&PayoutRequest{},
		&TaxRule{},
		&CostCenter{},
		&FinancialLedgerEntry{},
	)
}
