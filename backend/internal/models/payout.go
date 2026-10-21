package models

import (
	"time"

	"github.com/google/uuid"
)

type PayoutRequest struct {
	ID                  uint           `gorm:"primaryKey" json:"id"`
	DoctorID            uint           `gorm:"index;not null" json:"doctor_id"`
	Amount              int64          `gorm:"not null" json:"amount"` // In paise
	Currency            string         `gorm:"default:'INR'" json:"currency"`
	Status              string         `gorm:"default:'pending'" json:"status"` // pending, approved, paid, rejected
	LedgerTransactionID *uuid.UUID     `gorm:"type:uuid" json:"ledger_transaction_id"`
	AdminComment        string         `json:"admin_comment"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

func (PayoutRequest) TableName() string {
	return "payout_requests"
}
