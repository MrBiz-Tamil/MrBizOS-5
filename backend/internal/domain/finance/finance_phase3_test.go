package finance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpenseWorkflow(t *testing.T) {
	db := setupTestDB(t)
	// Add Expense to AutoMigrate in setup if not there, but setupTestDB in phase2_test might not have it.
	// Actually I updated model.go's AutoMigrate. 
	// But setupTestDB in phase2_test is local. I'll override it or use it.
	db.AutoMigrate(&Expense{}) 

	repo := NewRepository(db)
	svc := NewService(repo)

	// 1. Setup Accounts
	cashAcc, err := svc.CreateAccount(context.Background(), "Asset:Cash", TypeAsset, "INR")
	assert.NoError(t, err)

	// Seed Cash account with 100,000 paise (1,000 INR) to avoid negative balance guard trigger
	cashAcc.Balance = 100000
	cashAcc.LastChecksum = cashAcc.CalculateChecksum("SEED")
	db.Save(cashAcc)

	// 2. Log Expense
	exp := Expense{
		Description: "Office Supplies",
		Amount:      5000, // 50 INR
		Category:    "SUPPLIES",
		Department:  "ADMIN",
	}
	logged, err := svc.LogExpense(context.Background(), exp)
	assert.NoError(t, err)
	assert.NotZero(t, logged.ID)
	assert.Equal(t, "pending", logged.Status)

	// 3. Approve Expense
	// This should trigger: Debit Expense:General, Credit Asset:Cash
	err = svc.ApproveExpense(context.Background(), logged.ID, 1)
	assert.NoError(t, err)

	// Reload expense to check status and transaction ID
	var approved Expense
	db.First(&approved, logged.ID)
	assert.Equal(t, "approved", approved.Status)
	assert.NotZero(t, approved.LedgerTransactionID)

	// 4. Verify Ledger Balances
	var expAcc LedgerAccount
	db.Where("name = ?", "Expense:ADMIN:SUPPLIES").First(&expAcc)
	assert.Equal(t, int64(5000), expAcc.Balance)

	var cashAccReloaded LedgerAccount
	db.Where("name = ?", "Asset:Cash").First(&cashAccReloaded)
	assert.Equal(t, int64(95000), cashAccReloaded.Balance)

	// 5. Verify Transaction exists
	var tx LedgerTransaction
	err = db.First(&tx, "id = ?", approved.LedgerTransactionID).Error
	assert.NoError(t, err)
}

func TestTaxCalculationIntegration(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(NewRepository(db))

	// Setup Tax Rule
	rule := TaxRule{
		Region:   "IN-KA",
		Category: "PHARMACY",
		TaxType:  "GST",
		Rate:     12.0, // 12% GST on medicines
	}
	err := svc.CreateTaxRule(context.Background(), rule)
	assert.NoError(t, err)

	// Calculate Tax for 1000 INR
	tax, err := svc.CalculateTax(context.Background(), 100000, "IN-KA", "PHARMACY")
	assert.NoError(t, err)
	assert.Equal(t, int64(12000), tax) // 120 INR
}
