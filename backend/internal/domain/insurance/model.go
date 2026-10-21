package insurance

import (
	"time"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&InsuranceProvider{},
		&InsuranceClaim{},
	)
}

type ClaimStatus string

const (
	ClaimPending        ClaimStatus = "pending"
	ClaimPreAuthPending ClaimStatus = "pre_auth_pending"
	ClaimApproved       ClaimStatus = "approved"
	ClaimRejected       ClaimStatus = "rejected"
	ClaimEscrowed       ClaimStatus = "escrowed"
	ClaimSettled        ClaimStatus = "settled"
)

type InsuranceProvider struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Code      string    `gorm:"uniqueIndex" json:"code"`
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type InsuranceClaim struct {
	ID                 uint        `gorm:"primaryKey" json:"id"`
	ClaimNumber        string      `gorm:"uniqueIndex" json:"claim_number"`
	UHEID              string      `gorm:"index" json:"uheid"`
	ProviderID         uint        `json:"provider_id"` // external provider (hospital)
	InsurerID          uint        `json:"insurer_id"`  // insurance company
	Amount             int64       `json:"amount"`      // in cents
	Status             ClaimStatus `gorm:"default:'pending'" json:"status"`
	DiagnosisCode      string      `json:"diagnosis_code"`
	TreatmentCode      string      `json:"treatment_code"`
	EscrowTransactionID string     `json:"escrow_transaction_id"`
	Remarks            string      `json:"remarks"`
	// DocumentID links the uploaded supporting document in the private vault.
	DocumentID         *uint       `gorm:"index" json:"document_id,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}
