package anomaly

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"drayya-core/internal/domain/finance"
)

func fraudTestDB(t *testing.T) *gorm.DB {
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&finance.PayoutRequest{}, &finance.Expense{}, &finance.LedgerAccount{},
	))
	return db
}

func TestDuplicatePayoutRule(t *testing.T) {
	db := fraudTestDB(t)
	now := time.Now()
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Create(&finance.PayoutRequest{
			DoctorID: 7, Amount: 50000, Status: "pending", CreatedAt: now, UpdatedAt: now,
		}).Error)
	}

	found, err := duplicatePayoutRule().Detect(db, now.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, SeverityCritical, found[0].Severity)
	assert.Equal(t, 2, found[0].Hits)
}

func TestOversizedExpenseRule(t *testing.T) {
	db := fraudTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&finance.Expense{
		Description: "Mystery consulting", Amount: 6_000_000, Category: "OTHER",
		Department: "GENERAL", Status: "pending", CreatedBy: 3, CreatedAt: now, UpdatedAt: now,
	}).Error)

	found, err := oversizedExpenseRule().Detect(db, now.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, uint(3), found[0].ActorID)
}

func TestExpenseFourEyesRule(t *testing.T) {
	db := fraudTestDB(t)
	now := time.Now()
	approver := uint(5)
	require.NoError(t, db.Create(&finance.Expense{
		Description: "Self approved", Amount: 7_000_000, Category: "OTHER", Department: "GENERAL",
		Status: "approved", CreatedBy: 5, ApprovedBy: &approver, CreatedAt: now, UpdatedAt: now,
	}).Error)

	found, err := expenseFourEyesRule().Detect(db, now.Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, SeverityCritical, found[0].Severity)
}

func TestNegativeAssetBalanceRule(t *testing.T) {
	db := fraudTestDB(t)
	require.NoError(t, db.Create(&finance.LedgerAccount{
		Name: finance.AcctBank, Type: finance.TypeAsset, Balance: -50000,
	}).Error)

	found, err := negativeAssetBalanceRule().Detect(db, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Contains(t, found[0].Title, "negative")
}

// A clean dataset must produce zero findings (no false positives).
func TestFinanceRules_NoFalsePositives(t *testing.T) {
	db := fraudTestDB(t)
	now := time.Now()
	approver := uint(9)
	require.NoError(t, db.Create(&finance.PayoutRequest{DoctorID: 1, Amount: 1000, Status: "paid", CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&finance.Expense{Description: "Tea", Amount: 5000, Category: "OTHER", Department: "GENERAL", Status: "approved", CreatedBy: 2, ApprovedBy: &approver, CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&finance.LedgerAccount{Name: finance.AcctBank, Type: finance.TypeAsset, Balance: 100000}).Error)

	for _, rule := range []Rule{duplicatePayoutRule(), oversizedExpenseRule(), expenseFourEyesRule(), negativeAssetBalanceRule(), stuckHighValuePayoutRule()} {
		found, err := rule.Detect(db, now.Add(-time.Hour))
		require.NoError(t, err)
		assert.Empty(t, found, "rule %s should not fire on clean data", rule.Name)
	}
}
