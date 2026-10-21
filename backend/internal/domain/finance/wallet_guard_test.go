package finance_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"drayya-core/internal/domain/finance"
)

// The ledger must reject a wallet debit that would drive the balance negative —
// enforced inside the locked transaction, so concurrent double-spends can't slip
// through the service-level pre-check (TOCTOU).
func TestRecordTransaction_RejectsWalletOverspend(t *testing.T) {
	db := newFinanceTestDB(t)
	svc := finance.NewService(finance.NewRepository(db))
	ctx := context.Background()

	wallet, err := svc.GetOrCreateAccount(ctx, finance.PatientWallet(1), finance.TypeLiability)
	require.NoError(t, err)
	gateway, err := svc.GetOrCreateAccount(ctx, "Gateway:Test", finance.TypeAsset)
	require.NoError(t, err)
	revenue, err := svc.GetOrCreateAccount(ctx, finance.RevConsultations, finance.TypeRevenue)
	require.NoError(t, err)

	// Top up the wallet to ₹500.
	_, err = svc.RecordTransaction(ctx, "TOPUP-1", "TOPUP", "top up", []finance.EntryData{
		{AccountID: gateway.ID, Direction: finance.DirectionDebit, Amount: 50000},
		{AccountID: wallet.ID, Direction: finance.DirectionCredit, Amount: 50000},
	})
	require.NoError(t, err)

	// Attempt to spend ₹600 — must be rejected and rolled back.
	_, err = svc.RecordTransaction(ctx, "SPEND-1", "SPEND", "overspend", []finance.EntryData{
		{AccountID: wallet.ID, Direction: finance.DirectionDebit, Amount: 60000},
		{AccountID: revenue.ID, Direction: finance.DirectionCredit, Amount: 60000},
	})
	assert.Error(t, err, "wallet overspend must be rejected")

	assert.Equal(t, int64(50000), acctBalance(t, db, finance.PatientWallet(1)), "balance unchanged after rejected overspend")
}
