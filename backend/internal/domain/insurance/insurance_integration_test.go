package insurance

import (
	"context"
	"testing"
	"drayya-core/internal/domain/finance"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	
	// Migrate all needed models
	err = db.AutoMigrate(&InsuranceProvider{}, &InsuranceClaim{}, &finance.LedgerAccount{}, &finance.LedgerTransaction{}, &finance.LedgerEntry{})
	assert.NoError(t, err)
	return db
}

func TestInsuranceClaimSettlement(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	financeRepo := finance.NewRepository(db)
	financeSvc := finance.NewService(financeRepo)
	svc := NewService(repo, financeSvc)

	// 1. Submit Claim
	claim := InsuranceClaim{
		UHEID:      "DA-ECON-TEST",
		InsurerID:  1,
		ProviderID: 1,
		Amount:     500000, // 5000 INR
	}
	submitted, err := svc.SubmitClaim(claim)
	assert.NoError(t, err)
	assert.Equal(t, ClaimPending, submitted.Status)

	// 2. Approve Claim
	err = svc.ApproveClaim(submitted.ID, "Approved for surgery")
	assert.NoError(t, err)

	// 3. Settle Claim (Ledger Integration)
	err = svc.SettleClaim(submitted.ID)
	assert.NoError(t, err)

	// 4. Verify Ledger Impact
	// Check Insurer Receivable Account
	insurerAcct, err := financeSvc.GetOrCreateAccount(context.Background(), "Insurer:1", finance.TypeAsset)
	assert.NoError(t, err)
	assert.Equal(t, int64(500000), insurerAcct.Balance)

	// Check Provider Liability Account
	providerAcct, err := financeSvc.GetOrCreateAccount(context.Background(), "ExternalProvider:1", finance.TypeLiability)
	assert.NoError(t, err)
	assert.Equal(t, int64(500000), providerAcct.Balance)

	// 5. Verify Claim Status
	finalClaim, _ := svc.GetClaim(submitted.ID)
	assert.Equal(t, ClaimSettled, finalClaim.Status)
	assert.NotEmpty(t, finalClaim.EscrowTransactionID)
}
