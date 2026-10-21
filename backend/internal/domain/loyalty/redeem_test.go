package loyalty

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
)

func acctBalance(t *testing.T, db *gorm.DB, name string) int64 {
	var a finance.LedgerAccount
	require.NoError(t, db.Where("name = ?", name).First(&a).Error)
	return a.Balance
}

// The money leg of a loyalty redemption: debit Marketing:Rewards, credit the
// user's wallet, idempotent on the points-ledger entry id.
func TestPostRedemptionToWallet(t *testing.T) {
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	database.DB = db
	require.NoError(t, finance.AutoMigrate(db))

	// 1000 points * ₹0.25 = ₹250 = 25000 paise
	postRedemptionToWallet(7, 1, 25000)

	assert.Equal(t, int64(25000), acctBalance(t, db, finance.PatientWallet(7)), "wallet credited")
	assert.Equal(t, int64(25000), acctBalance(t, db, finance.ExpMarketingRewards), "marketing expense debited")

	// Replay with the same ledger id must not double-credit.
	postRedemptionToWallet(7, 1, 25000)
	assert.Equal(t, int64(25000), acctBalance(t, db, finance.PatientWallet(7)), "idempotent")
}
