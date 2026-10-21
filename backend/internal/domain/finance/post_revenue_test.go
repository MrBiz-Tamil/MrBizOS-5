package finance_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
)

func newFinanceTestDB(t *testing.T) *gorm.DB {
	// Unique shared-cache name per test → isolated in-memory DB, no cross-test bleed.
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	database.DB = db
	require.NoError(t, db.AutoMigrate(
		&finance.LedgerAccount{}, &finance.LedgerTransaction{}, &finance.LedgerEntry{},
	))
	return db
}

func acctBalance(t *testing.T, db *gorm.DB, name string) int64 {
	var a finance.LedgerAccount
	require.NoError(t, db.Where("name = ?", name).First(&a).Error)
	return a.Balance
}

// A subscription-style sale: ₹1250 gross, GST extracted inclusive at 18%.
func TestPostRevenue_BalancedSplit(t *testing.T) {
	db := newFinanceTestDB(t)
	svc := finance.NewService(finance.NewRepository(db))

	total := int64(125000)
	gst := total * 18 / 118 // 19067
	net := total - gst      // 105933

	_, err := svc.PostRevenue(context.Background(), "SUB-PAY-1", "SUBSCRIPTION", "test charge",
		finance.RevenueInput{
			FundingAccount: finance.AcctRazorpay, FundingType: finance.TypeAsset, Total: total,
			Credits: []finance.RevenueLeg{
				{Account: finance.RevSubscriptions, Type: finance.TypeRevenue, Amount: net},
				{Account: finance.LiabGST, Type: finance.TypeLiability, Amount: gst},
			},
		})
	require.NoError(t, err)

	assert.Equal(t, total, acctBalance(t, db, finance.AcctRazorpay), "funding asset = gross")
	assert.Equal(t, net, acctBalance(t, db, finance.RevSubscriptions))
	assert.Equal(t, gst, acctBalance(t, db, finance.LiabGST))
}

func TestPostRevenue_Idempotent(t *testing.T) {
	db := newFinanceTestDB(t)
	svc := finance.NewService(finance.NewRepository(db))

	in := finance.RevenueInput{
		FundingAccount: finance.AcctRazorpay, FundingType: finance.TypeAsset, Total: 50000,
		Credits: []finance.RevenueLeg{{Account: finance.RevSubscriptions, Type: finance.TypeRevenue, Amount: 50000}},
	}
	t1, err := svc.PostRevenue(context.Background(), "SUB-PAY-DUP", "SUBSCRIPTION", "first", in)
	require.NoError(t, err)
	t2, err := svc.PostRevenue(context.Background(), "SUB-PAY-DUP", "SUBSCRIPTION", "replay", in)
	require.NoError(t, err)

	assert.Equal(t, t1.ID, t2.ID, "duplicate reference returns the same transaction")
	assert.Equal(t, int64(50000), acctBalance(t, db, finance.RevSubscriptions), "balance not double-counted")
}

func TestPostRevenue_RejectsUnbalanced(t *testing.T) {
	db := newFinanceTestDB(t)
	svc := finance.NewService(finance.NewRepository(db))

	_, err := svc.PostRevenue(context.Background(), "SUB-BAD", "SUBSCRIPTION", "bad",
		finance.RevenueInput{
			FundingAccount: finance.AcctRazorpay, FundingType: finance.TypeAsset, Total: 100000,
			Credits: []finance.RevenueLeg{{Account: finance.RevSubscriptions, Type: finance.TypeRevenue, Amount: 90000}},
		})
	assert.Error(t, err, "legs that don't sum to total must be rejected")
}
