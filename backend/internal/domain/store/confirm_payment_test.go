package store

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

func setupStoreTest(t *testing.T) (Service, Repository, *gorm.DB) {
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	database.DB = db
	repo := NewRepository(db)
	require.NoError(t, repo.AutoMigrate())
	require.NoError(t, finance.AutoMigrate(db))

	require.NoError(t, db.Create(&WellnessProduct{ID: 1, Name: "Vitamin C", SellingPrice: 50000, IsActive: true}).Error)
	require.NoError(t, db.Create(&WellnessInventory{ProductID: 1, PharmacyID: 1, StockQuantity: 10}).Error)

	finSvc := finance.NewService(finance.NewRepository(db))
	return NewService(repo, finSvc), repo, db
}

func acctBalance(t *testing.T, db *gorm.DB, name string) int64 {
	var a finance.LedgerAccount
	require.NoError(t, db.Where("name = ?", name).First(&a).Error)
	return a.Balance
}

func TestConfirmStorePayment_RevenueAndInventory(t *testing.T) {
	svc, repo, db := setupStoreTest(t)
	ctx := context.Background()

	order, err := svc.CreateOrder(ctx, 7, []OrderItemRequest{{ProductID: 1, Quantity: 2}}, "addr", 1, "9876543210", "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, int64(100000), order.TotalAmount)
	assert.Equal(t, "9876543210", string(order.ContactPhone))
	assert.Equal(t, "test@example.com", string(order.ContactEmail))

	// Stock untouched at order creation (no premature reservation).
	inv, _ := repo.GetInventory(ctx, 1, 1)
	assert.Equal(t, 10, inv.StockQuantity)

	paid, err := svc.ConfirmStorePayment(ctx, order.ID, "razorpay", "pay_s")
	require.NoError(t, err)
	assert.Equal(t, "paid", paid.PaymentStatus)

	// total 100000 → gst 100000*18/118 = 15254, revenue 84746
	assert.Equal(t, int64(100000), acctBalance(t, db, finance.AcctRazorpay))
	assert.Equal(t, int64(15254), acctBalance(t, db, finance.LiabGST))
	assert.Equal(t, int64(84746), acctBalance(t, db, finance.RevStore))

	// Stock decremented only on payment.
	inv, _ = repo.GetInventory(ctx, 1, 1)
	assert.Equal(t, 8, inv.StockQuantity)
}

func TestConfirmStorePayment_Idempotent(t *testing.T) {
	svc, repo, db := setupStoreTest(t)
	ctx := context.Background()
	order, err := svc.CreateOrder(ctx, 7, []OrderItemRequest{{ProductID: 1, Quantity: 2}}, "addr", 1, "9876543210", "test@example.com")
	require.NoError(t, err)

	_, err = svc.ConfirmStorePayment(ctx, order.ID, "razorpay", "p")
	require.NoError(t, err)
	_, err = svc.ConfirmStorePayment(ctx, order.ID, "razorpay", "p")
	require.NoError(t, err)

	assert.Equal(t, int64(100000), acctBalance(t, db, finance.AcctRazorpay), "no double-post")
	inv, _ := repo.GetInventory(ctx, 1, 1)
	assert.Equal(t, 8, inv.StockQuantity, "stock decremented once")
}

func TestGetOrderByID(t *testing.T) {
	svc, _, _ := setupStoreTest(t)
	ctx := context.Background()
	
	order, err := svc.CreateOrder(ctx, 7, []OrderItemRequest{{ProductID: 1, Quantity: 1}}, "addr", 1, "9876543210", "test@example.com")
	require.NoError(t, err)

	fetched, err := svc.GetOrderByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, order.ID, fetched.ID)
	assert.Equal(t, uint(7), fetched.UserID)
}
