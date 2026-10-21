package hr

import "time"

// TDSRemittance records a quarterly TDS challan deposit to the government.
// The Fiber HTTP handlers that recorded/listed remittances were removed with
// the REST server; the model remains for migrations and reporting.
// AmountPaise is paise.
type TDSRemittance struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	FinancialYear string    `gorm:"index" json:"financial_year"` // "2026-27"
	Quarter       string    `gorm:"index" json:"quarter"`        // Q1 | Q2 | Q3 | Q4
	ChallanNumber string    `json:"challan_number"`
	AmountPaise   int64     `json:"amount_paise"`
	Section       string    `gorm:"default:'194J'" json:"section"` // 194J = professional fees
	RemittedOn    time.Time `json:"remitted_on"`
	Notes         string    `gorm:"type:text" json:"notes"`
}

func (TDSRemittance) TableName() string { return "tds_remittances" }
