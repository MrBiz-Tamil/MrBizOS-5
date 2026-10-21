package finance_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drayya-core/internal/domain/finance"
)

// A clean ledger built only through the service should pass every check.
func TestReconcile_CleanLedgerPasses(t *testing.T) {
	db := newFinanceTestDB(t)
	svc := finance.NewService(finance.NewRepository(db))

	// Post a couple of balanced sales.
	_, err := svc.PostRevenue(context.Background(), "REC-1", "SUBSCRIPTION", "s1",
		finance.RevenueInput{
			FundingAccount: finance.AcctRazorpay, FundingType: finance.TypeAsset, Total: 125000,
			Credits: []finance.RevenueLeg{
				{Account: finance.RevSubscriptions, Type: finance.TypeRevenue, Amount: 105933},
				{Account: finance.LiabGST, Type: finance.TypeLiability, Amount: 19067},
			},
		})
	require.NoError(t, err)
	_, err = svc.PostRevenue(context.Background(), "REC-2", "STORE", "s2",
		finance.RevenueInput{
			FundingAccount: finance.AcctCash, FundingType: finance.TypeAsset, Total: 30000,
			Credits: []finance.RevenueLeg{{Account: finance.RevStore, Type: finance.TypeRevenue, Amount: 30000}},
		})
	require.NoError(t, err)

	report := svc.Reconcile(db)
	assert.True(t, report.AllPassed, "clean ledger should pass: %+v", report.Results)
}

// Tampering with a stored account balance out-of-band must be caught by the
// per-account balance proof.
func TestReconcile_DetectsBalanceTampering(t *testing.T) {
	db := newFinanceTestDB(t)
	svc := finance.NewService(finance.NewRepository(db))

	_, err := svc.PostRevenue(context.Background(), "REC-T", "SUBSCRIPTION", "s",
		finance.RevenueInput{
			FundingAccount: finance.AcctRazorpay, FundingType: finance.TypeAsset, Total: 100000,
			Credits: []finance.RevenueLeg{{Account: finance.RevSubscriptions, Type: finance.TypeRevenue, Amount: 100000}},
		})
	require.NoError(t, err)

	// Out-of-band fraud: inflate the revenue balance directly, bypassing entries.
	require.NoError(t, db.Table("ledger_account_v2").
		Where("name = ?", finance.RevSubscriptions).
		Update("balance", 999999).Error)

	report := svc.Reconcile(db)
	assert.False(t, report.AllPassed, "tampered balance must fail reconciliation")

	var balanceCheck *finance.CheckResult
	for i := range report.Results {
		if report.Results[i].Name == "account_balance_proof" {
			balanceCheck = &report.Results[i]
		}
	}
	require.NotNil(t, balanceCheck)
	assert.False(t, balanceCheck.Passed, "balance proof should flag the drift")
}
