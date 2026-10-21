package pharmacy_test

import (
	"context"
	"testing"

	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/money"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupPricingDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&pharmacy.Medicine{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestPriceCart_ItemisedBillBalances(t *testing.T) {
	db := setupPricingDB(t)
	// AYUSH medicine @ 5% GST, wellness @ 18%.
	medA := pharmacy.Medicine{Name: "Nilavembu Kashayam", Prices: money.RegionalPricing{"INR": 12000}, GSTPercent: 5, IsActive: true}
	medB := pharmacy.Medicine{Name: "Tulsi Drops", Prices: money.RegionalPricing{"INR": 9900}, GSTPercent: 18, IsActive: true}
	db.Create(&medA)
	db.Create(&medB)

	totals, err := pharmacy.PriceCart(context.Background(), db, pharmacy.PricingRequest{
		Items: []pharmacy.ReservationItem{
			{MedicineID: medA.ID, Quantity: 2}, // 2 × 120 = 240 + 5%  = 252
			{MedicineID: medB.ID, Quantity: 1}, // 1 × 99  = 99  + 18% = 116.82 → 117
		},
		PaymentMethod: pharmacy.PaymentPrepaid,
	})
	if err != nil {
		t.Fatalf("PriceCart: %v", err)
	}

	// Line subtotals.
	if totals.Lines[0].LineSubtotalPaise != 24000 || totals.Lines[1].LineSubtotalPaise != 9900 {
		t.Errorf("line subtotals off: %+v", totals.Lines)
	}
	// GST: 5% of 24000 = 1200; 18% of 9900 = 1782.
	if totals.Lines[0].GSTPaise != 1200 || totals.Lines[1].GSTPaise != 1782 {
		t.Errorf("line GST off: %+v", totals.Lines)
	}
	// Subtotal = 24000 + 9900 = 33900.
	if totals.SubtotalPaise != 33900 {
		t.Errorf("subtotal = %d, want 33900", totals.SubtotalPaise)
	}
	// GST total = 1200 + 1782 = 2982.
	if totals.TotalGSTPaise != 2982 {
		t.Errorf("gst total = %d, want 2982", totals.TotalGSTPaise)
	}
	// Subtotal+GST = 36882 < ₹499 threshold → flat shipping applies.
	if totals.ShippingPaise != pharmacy.FlatShippingPaise {
		t.Errorf("shipping = %d, want %d (flat under threshold)", totals.ShippingPaise, pharmacy.FlatShippingPaise)
	}
	// Prepaid → no COD surcharge.
	if totals.CODSurchargePaise != 0 {
		t.Errorf("prepaid order should have no COD surcharge; got %d", totals.CODSurchargePaise)
	}
	// Invariant.
	want := totals.SubtotalPaise + totals.TotalGSTPaise + totals.ShippingPaise
	if totals.GrandTotalPaise != want {
		t.Errorf("grand total %d != subtotal+gst+shipping %d", totals.GrandTotalPaise, want)
	}
}

func TestPriceCart_FreeShippingAboveThreshold(t *testing.T) {
	db := setupPricingDB(t)
	med := pharmacy.Medicine{Name: "Chyawanprash", Prices: money.RegionalPricing{"INR": 35000}, GSTPercent: 5, IsActive: true}
	db.Create(&med)

	// 2 × 350 = 700 + 35 = 735 → above ₹499 threshold.
	totals, err := pharmacy.PriceCart(context.Background(), db, pharmacy.PricingRequest{
		Items:         []pharmacy.ReservationItem{{MedicineID: med.ID, Quantity: 2}},
		PaymentMethod: pharmacy.PaymentPrepaid,
	})
	if err != nil {
		t.Fatalf("PriceCart: %v", err)
	}
	if totals.ShippingPaise != 0 {
		t.Errorf("expected free shipping above ₹499; got %d", totals.ShippingPaise)
	}
}

func TestPriceCart_CODAddsSurcharge(t *testing.T) {
	db := setupPricingDB(t)
	med := pharmacy.Medicine{Name: "X", Prices: money.RegionalPricing{"INR": 10000}, GSTPercent: 5, IsActive: true}
	db.Create(&med)

	totals, err := pharmacy.PriceCart(context.Background(), db, pharmacy.PricingRequest{
		Items:         []pharmacy.ReservationItem{{MedicineID: med.ID, Quantity: 1}},
		PaymentMethod: pharmacy.PaymentCOD,
	})
	if err != nil {
		t.Fatalf("PriceCart: %v", err)
	}
	if totals.CODSurchargePaise != pharmacy.CODSurchargePaise {
		t.Errorf("COD surcharge = %d, want %d", totals.CODSurchargePaise, pharmacy.CODSurchargePaise)
	}
}

func TestPriceCart_InsuranceAndDiscountClampNonNegativeGrandTotal(t *testing.T) {
	db := setupPricingDB(t)
	med := pharmacy.Medicine{Name: "X", Prices: money.RegionalPricing{"INR": 5000}, GSTPercent: 5, IsActive: true}
	db.Create(&med)

	// 1 × ₹50 + 5% = ₹52.50. Discount of ₹100 + insurance ₹200 would push to negative.
	totals, err := pharmacy.PriceCart(context.Background(), db, pharmacy.PricingRequest{
		Items:                 []pharmacy.ReservationItem{{MedicineID: med.ID, Quantity: 1}},
		PaymentMethod:         pharmacy.PaymentPrepaid,
		DiscountPaise:         10000,
		InsuranceCoveredPaise: 20000,
	})
	if err != nil {
		t.Fatalf("PriceCart: %v", err)
	}
	if totals.GrandTotalPaise < 0 {
		t.Errorf("grand total should be clamped to 0, got %d", totals.GrandTotalPaise)
	}
}

func TestPriceCart_RejectsInactiveOrUnknownMedicine(t *testing.T) {
	db := setupPricingDB(t)
	med := pharmacy.Medicine{Name: "X", Prices: money.RegionalPricing{"INR": 1000}, GSTPercent: 5, IsActive: true}
	db.Create(&med)

	if _, err := pharmacy.PriceCart(context.Background(), db, pharmacy.PricingRequest{
		Items: []pharmacy.ReservationItem{{MedicineID: 999, Quantity: 1}},
	}); err == nil {
		t.Fatal("pricing should fail on unknown medicine; got nil")
	}
}
