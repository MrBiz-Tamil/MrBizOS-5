package main

import (
	"fmt"
	"log"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func seedFinance(db *gorm.DB) {
	// Ledger accounts
	accounts := []finance.LedgerAccount{
		{ID: uuid.New(), Name: "Platform Revenue", Type: finance.TypeRevenue, Currency: "INR", Balance: 1500000},
		{ID: uuid.New(), Name: "Doctor Payouts", Type: finance.TypeExpense, Currency: "INR", Balance: 0},
		{ID: uuid.New(), Name: "Pharmacy Revenue", Type: finance.TypeRevenue, Currency: "INR", Balance: 500000},
		{ID: uuid.New(), Name: "Operating Expenses", Type: finance.TypeExpense, Currency: "INR", Balance: 0},
		{ID: uuid.New(), Name: "Cash Account", Type: finance.TypeAsset, Currency: "INR", Balance: 5000000},
		{ID: uuid.New(), Name: "Patient Deposits", Type: finance.TypeLiability, Currency: "INR", Balance: 200000},
	}
	for i := range accounts {
		var existing finance.LedgerAccount
		if db.Where("name = ?", accounts[i].Name).First(&existing).Error == nil {
			continue
		}
		db.Create(&accounts[i])
	}

	// Seed patient wallet
	var patient identity.User
	if db.Where("email = ?", "patient1@Dr.Ayya.com").First(&patient).Error == nil {
		walletName := fmt.Sprintf("PatientWallet:%d", patient.ID)
		var walletAcct finance.LedgerAccount
		if db.Where("name = ?", walletName).First(&walletAcct).Error != nil {
			walletAcct = finance.LedgerAccount{
				ID:       uuid.New(),
				Name:     walletName,
				Type:     finance.TypeLiability,
				Currency: "INR",
				Balance:  100000, // 1000 INR
			}
			db.Create(&walletAcct)
		} else {
			db.Model(&walletAcct).Update("balance", 100000)
		}
	}

	// Reload accounts to get IDs
	var revenue, doctorPay, cashAcct finance.LedgerAccount
	db.Where("name = ?", "Platform Revenue").First(&revenue)
	db.Where("name = ?", "Doctor Payouts").First(&doctorPay)
	db.Where("name = ?", "Cash Account").First(&cashAcct)

	if revenue.ID == uuid.Nil || doctorPay.ID == uuid.Nil {
		log.Println("          Accounts not found, skipping transactions")
		return
	}

	// Sample transactions
	now := time.Now()
	txns := []struct {
		refID   string
		txnType string
		desc    string
		entries []finance.EntryData
	}{
		{
			refID: "APT-001", txnType: "CONSULTATION", desc: "Consultation fee for appointment #1",
			entries: []finance.EntryData{
				{AccountID: cashAcct.ID, Direction: finance.DirectionDebit, Amount: 50000},
				{AccountID: revenue.ID, Direction: finance.DirectionCredit, Amount: 7500},
				{AccountID: doctorPay.ID, Direction: finance.DirectionDebit, Amount: 42500},
			},
		},
		{
			refID: "APT-002", txnType: "CONSULTATION", desc: "Consultation fee for appointment #2",
			entries: []finance.EntryData{
				{AccountID: cashAcct.ID, Direction: finance.DirectionDebit, Amount: 80000},
				{AccountID: revenue.ID, Direction: finance.DirectionCredit, Amount: 12000},
				{AccountID: doctorPay.ID, Direction: finance.DirectionDebit, Amount: 68000},
			},
		},
		{
			refID: "TOPUP-001", txnType: "TOPUP", desc: "Patient wallet top-up",
			entries: []finance.EntryData{
				{AccountID: cashAcct.ID, Direction: finance.DirectionDebit, Amount: 200000},
			},
		},
	}

	for _, t := range txns {
		var existing finance.LedgerTransaction
		if db.Where("reference_id = ?", t.refID).First(&existing).Error == nil {
			continue
		}
		txn := finance.LedgerTransaction{
			ID:          uuid.New(),
			ReferenceID: t.refID,
			Type:        t.txnType,
			Description: t.desc,
			Status:      "POSTED",
			CreatedAt:   now,
		}
		db.Create(&txn)

		for _, e := range t.entries {
			db.Create(&finance.LedgerEntry{
				TransactionID: txn.ID,
				AccountID:     e.AccountID,
				Direction:     e.Direction,
				Amount:        e.Amount,
				BalanceAfter:  0, // Simplified
			})
		}
	}

	log.Printf("       Created %d ledger accounts and %d transactions", len(accounts), len(txns))
}

