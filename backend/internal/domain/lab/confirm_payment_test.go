package lab

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

func setupLabTest(t *testing.T) (Service, *gorm.DB) {
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	database.DB = db
	repo := NewRepository(db)
	require.NoError(t, repo.AutoMigrate())
	require.NoError(t, finance.AutoMigrate(db))

	require.NoError(t, db.Create(&LabPartner{ID: 1, Name: "Thyrocare", Slug: "thyrocare"}).Error)
	require.NoError(t, db.Create(&LabCenter{ID: 1, PartnerID: 1, Name: "Center A"}).Error)
	require.NoError(t, db.Create(&LabTest{ID: 1, Name: "CBC", BasePrice: 100000, IsActive: true}).Error)
	// Server-authoritative price for (center 1, test 1).
	require.NoError(t, db.Create(&LabCenterPricing{LabCenterID: 1, LabTestID: 1, PricePaise: 100000, IsAvailable: true}).Error)

	finSvc := finance.NewService(finance.NewRepository(db))
	return NewService(repo, finSvc), db
}

func acctBalance(t *testing.T, db *gorm.DB, name string) int64 {
	var a finance.LedgerAccount
	require.NoError(t, db.Where("name = ?", name).First(&a).Error)
	return a.Balance
}

func TestConfirmLabPayment_MarketplaceSplit(t *testing.T) {
	svc, db := setupLabTest(t)
	ctx := context.Background()

	b, err := svc.CreateBooking(ctx, 42, BookingRequest{
		LabCenterID: 1, LabTestID: 1, BookingType: "lab_visit",
		PreferredDate: "2026-06-10", TestPrice: 100000,
	})
	require.NoError(t, err)

	paid, err := svc.ConfirmLabPayment(ctx, b.ID, "razorpay", "pay_xyz")
	require.NoError(t, err)
	assert.Equal(t, "paid", paid.PaymentStatus)

	// total 100000 → gst 4761, net 95239, commission 19047, provider 76192
	assert.Equal(t, int64(100000), acctBalance(t, db, finance.AcctRazorpay))
	assert.Equal(t, int64(19047), acctBalance(t, db, finance.RevLab))
	assert.Equal(t, int64(4761), acctBalance(t, db, finance.LiabGST))
	assert.Equal(t, int64(76192), acctBalance(t, db, finance.ProviderWallet("Lab", 1)))
}

// A malicious client sends a tiny TestPrice; the server must ignore it and use
// the authoritative center pricing (₹1000), not ₹1.
func TestCreateBooking_IgnoresClientPrice(t *testing.T) {
	svc, _ := setupLabTest(t)
	ctx := context.Background()

	b, err := svc.CreateBooking(ctx, 42, BookingRequest{
		LabCenterID: 1, LabTestID: 1, BookingType: "lab_visit",
		PreferredDate: "2026-06-10",
		TestPrice:     1, // attacker-supplied — must be ignored
	})
	require.NoError(t, err)
	assert.Equal(t, int64(100000), b.TotalPaise, "server pricing wins over client input")
}

// A Stripe-funded payment books the INR price into Asset:Stripe (the customer
// paid in their currency, but Stripe settles to our INR bank).
func TestConfirmLabPayment_StripeFundsInINR(t *testing.T) {
	svc, db := setupLabTest(t)
	ctx := context.Background()
	b, err := svc.CreateBooking(ctx, 42, BookingRequest{
		LabCenterID: 1, LabTestID: 1, BookingType: "lab_visit", PreferredDate: "2026-06-10",
	})
	require.NoError(t, err)

	_, err = svc.ConfirmLabPayment(ctx, b.ID, "stripe", "pi_123")
	require.NoError(t, err)

	assert.Equal(t, int64(100000), acctBalance(t, db, finance.AcctStripe), "INR price booked into Asset:Stripe")
	assert.Equal(t, int64(19047), acctBalance(t, db, finance.RevLab))
}

func TestConfirmLabPayment_Idempotent(t *testing.T) {
	svc, db := setupLabTest(t)
	ctx := context.Background()
	b, err := svc.CreateBooking(ctx, 42, BookingRequest{
		LabCenterID: 1, LabTestID: 1, BookingType: "lab_visit",
		PreferredDate: "2026-06-10", TestPrice: 100000,
	})
	require.NoError(t, err)

	_, err = svc.ConfirmLabPayment(ctx, b.ID, "razorpay", "pay_1")
	require.NoError(t, err)
	_, err = svc.ConfirmLabPayment(ctx, b.ID, "razorpay", "pay_1")
	require.NoError(t, err)

	assert.Equal(t, int64(100000), acctBalance(t, db, finance.AcctRazorpay), "must not double-post")
}
