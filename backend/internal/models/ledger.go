//go:build ignore
// +build ignore

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccountType string

const (
	TypeAsset     AccountType = "asset"
	TypeLiability AccountType = "liability"
	TypeEquity    AccountType = "equity"
	TypeRevenue   AccountType = "revenue"
	TypeExpense   AccountType = "expense"
)

type EntryDirection string

const (
	DirectionDebit  EntryDirection = "debit"
	DirectionCredit EntryDirection = "credit"
)

// LedgerAccount mirrors ledger_account_v2
type LedgerAccount struct {
	ID           uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string      `gorm:"index" json:"name"`
	Type         AccountType `gorm:"type:varchar(20)" json:"type"`
	Currency     string      `gorm:"default:'INR'" json:"currency"`
	Balance      int64       `gorm:"default:0" json:"balance"` // Stored in atomic units (e.g., paise)
	IsFrozen     bool        `gorm:"default:false" json:"is_frozen"`
	LastChecksum string      `json:"last_checksum"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

func (LedgerAccount) TableName() string {
	return "ledger_account_v2"
}

func (l *LedgerAccount) BeforeCreate(tx *gorm.DB) (err error) {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return
}

// LedgerTransaction mirrors ledger_transaction_v2
type LedgerTransaction struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ReferenceID  string     `gorm:"index" json:"reference_id"`
	Type         string     `gorm:"index" json:"type"` // TOPUP, CONSULTATION
	Description  string     `json:"description"`
	MetadataJSON string     `json:"metadata_json"`
	CreatedAt    time.Time  `json:"created_at"`
	PostedAt     *time.Time `json:"posted_at"`
}

func (LedgerTransaction) TableName() string {
	return "ledger_transaction_v2"
}

func (t *LedgerTransaction) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return
}

// LedgerEntry mirrors ledger_entry_v2
type LedgerEntry struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	TransactionID uuid.UUID      `gorm:"type:uuid;index" json:"transaction_id"`
	AccountID     uuid.UUID      `gorm:"type:uuid;index" json:"account_id"`
	Account       LedgerAccount  `gorm:"foreignKey:AccountID" json:"account"`
	Direction     EntryDirection `gorm:"type:varchar(10)" json:"direction"`
	Amount        int64          `gorm:"check:amount > 0" json:"amount"`
	BalanceAfter  int64          `json:"balance_after"`
	CreatedAt     time.Time      `json:"created_at"`
}

func (LedgerEntry) TableName() string {
	return "ledger_entry_v2"
}

func (e *LedgerEntry) BeforeCreate(tx *gorm.DB) (err error) {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return
}
