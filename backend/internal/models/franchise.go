//go:build ignore
// +build ignore

package models

import (
	"drayya-core/internal/domain/identity"
	"time"
)

type Franchise struct {
	ID          uint          `gorm:"primaryKey" json:"id"`
	OwnerID     uint          `gorm:"uniqueIndex;not null" json:"owner_id"`
	Owner       identity.User `gorm:"foreignKey:OwnerID" json:"owner"`
	Name        string        `gorm:"not null" json:"name"`
	RoyaltyRate float64       `gorm:"default:10.0" json:"royalty_rate"` // Percentage of consultation fee
	Region      string        `json:"region"`
	IsActive    bool          `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

func (Franchise) TableName() string {
	return "franchises"
}

type FranchiseRoyaltyLog struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	FranchiseID   uint      `gorm:"index" json:"franchise_id"`
	AppointmentID uint      `gorm:"index" json:"appointment_id"`
	TotalAmount   int64     `json:"total_amount"`
	RoyaltyAmount int64     `json:"royalty_amount"`
	TransactionID string    `json:"transaction_id"` // Ledger reference
	CreatedAt     time.Time `json:"created_at"`
}

func (FranchiseRoyaltyLog) TableName() string {
	return "franchise_royalty_logs"
}
