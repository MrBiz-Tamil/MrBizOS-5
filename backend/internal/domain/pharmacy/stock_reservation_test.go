package pharmacy_test

import (
	"context"
	"testing"
	"time"

	"drayya-core/internal/domain/pharmacy"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupReservationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}
	if err := db.AutoMigrate(&pharmacy.Pharmacy{}, &pharmacy.Medicine{}, &pharmacy.PharmacyInventory{}, &pharmacy.StockReservation{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// seedTwoPharmacies puts pharmacy "Near" at coordinates close to the patient and
// "Far" further away. Both stock the same medicine. Returns (nearID, farID, medID).
func seedTwoPharmacies(t *testing.T, db *gorm.DB, nearStock, farStock int) (uint, uint, uint) {
	t.Helper()
	near := pharmacy.Pharmacy{Name: "Near", Latitude: 13.082, Longitude: 80.270, Pincode: "600001"}
	far := pharmacy.Pharmacy{Name: "Far", Latitude: 19.076, Longitude: 72.877, Pincode: "400001"}
	med := pharmacy.Medicine{Name: "Nilavembu Kashayam", Category: "siddha", IsActive: true}
	db.Create(&near)
	db.Create(&far)
	db.Create(&med)
	if nearStock > 0 {
		db.Create(&pharmacy.PharmacyInventory{PharmacyID: near.ID, MedicineID: med.ID, StockQuantity: nearStock, IsActive: true})
	}
	if farStock > 0 {
		db.Create(&pharmacy.PharmacyInventory{PharmacyID: far.ID, MedicineID: med.ID, StockQuantity: farStock, IsActive: true})
	}
	return near.ID, far.ID, med.ID
}

func TestPickNearestPharmacyAndReserve_PicksNearest(t *testing.T) {
	db := setupReservationDB(t)
	nearID, _, medID := seedTwoPharmacies(t, db, 5, 5)

	// Patient sits at Chennai-ish coords — Near (Chennai) should win over Far (Mumbai).
	res, err := pharmacy.PickNearestPharmacyAndReserve(context.Background(), db,
		7, 13.083, 80.275,
		[]pharmacy.ReservationItem{{MedicineID: medID, Quantity: 2}},
		15*time.Minute)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if res.PharmacyID != nearID {
		t.Errorf("picker chose pharmacy %d, want %d (nearest)", res.PharmacyID, nearID)
	}
	if len(res.ReservationIDs) != 1 {
		t.Errorf("expected 1 reservation, got %d", len(res.ReservationIDs))
	}
	if res.DistanceKm > 1 {
		t.Errorf("near distance should be ~0, got %.2f km", res.DistanceKm)
	}
}

func TestPickNearestPharmacyAndReserve_FallsBackWhenNearestIsOutOfStock(t *testing.T) {
	db := setupReservationDB(t)
	_, farID, medID := seedTwoPharmacies(t, db, 1, 10) // Near only has 1, we want 5

	res, err := pharmacy.PickNearestPharmacyAndReserve(context.Background(), db,
		7, 13.083, 80.275,
		[]pharmacy.ReservationItem{{MedicineID: medID, Quantity: 5}},
		15*time.Minute)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if res.PharmacyID != farID {
		t.Errorf("picker should have fallen back to Far (which has stock), got %d", res.PharmacyID)
	}
}

func TestPickNearestPharmacyAndReserve_PreventsOversoldUnderConcurrentCarts(t *testing.T) {
	db := setupReservationDB(t)
	nearID, _, medID := seedTwoPharmacies(t, db, 3, 0) // single source with exactly 3 units

	// First cart reserves 2 units — succeeds.
	first, err := pharmacy.PickNearestPharmacyAndReserve(context.Background(), db,
		7, 13.083, 80.275,
		[]pharmacy.ReservationItem{{MedicineID: medID, Quantity: 2}},
		15*time.Minute)
	if err != nil {
		t.Fatalf("first reserve: %v", err)
	}
	if first.PharmacyID != nearID {
		t.Fatalf("first cart should have picked %d, got %d", nearID, first.PharmacyID)
	}

	// Second cart immediately tries 2 more — only 1 unit is unreserved, so it
	// must fail rather than silently overselling.
	_, err = pharmacy.PickNearestPharmacyAndReserve(context.Background(), db,
		8, 13.083, 80.275,
		[]pharmacy.ReservationItem{{MedicineID: medID, Quantity: 2}},
		15*time.Minute)
	if err == nil {
		t.Fatal("second cart should have failed (only 1 unit free after first cart's hold); got success")
	}
}

func TestPickNearestPharmacyAndReserve_ExpiredHoldsFreeUpStock(t *testing.T) {
	db := setupReservationDB(t)
	nearID, _, medID := seedTwoPharmacies(t, db, 3, 0)

	// Create an EXPIRED reservation directly so we can verify it's ignored.
	expired := pharmacy.StockReservation{
		CartID: "OLD-CART", PharmacyID: nearID, MedicineID: medID, Quantity: 2,
		ExpiresAt: time.Now().Add(-1 * time.Minute), Status: "active",
	}
	db.Create(&expired)

	// New cart wants 3 units. Without lazy expiry it would see only 1 free and
	// fail; with lazy expiry the old reservation no longer counts.
	res, err := pharmacy.PickNearestPharmacyAndReserve(context.Background(), db,
		7, 13.083, 80.275,
		[]pharmacy.ReservationItem{{MedicineID: medID, Quantity: 3}},
		15*time.Minute)
	if err != nil {
		t.Fatalf("reserve after expiry: %v (expected expired hold to be ignored)", err)
	}
	if res.PharmacyID != nearID {
		t.Errorf("expected pharmacy %d, got %d", nearID, res.PharmacyID)
	}
}

func TestConfirmReservations_DecrementsInventoryAtomically(t *testing.T) {
	db := setupReservationDB(t)
	nearID, _, medID := seedTwoPharmacies(t, db, 5, 0)

	res, err := pharmacy.PickNearestPharmacyAndReserve(context.Background(), db,
		7, 13.083, 80.275,
		[]pharmacy.ReservationItem{{MedicineID: medID, Quantity: 3}},
		15*time.Minute)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}

	if err := pharmacy.ConfirmReservations(context.Background(), db, res.CartID); err != nil {
		t.Fatalf("confirm: %v", err)
	}

	// Inventory should be 5 − 3 = 2 and the reservation row marked confirmed.
	var remaining int
	db.Table("pharmacyinventory").
		Where("pharmacy_id = ? AND medicine_id = ?", nearID, medID).
		Select("stock_quantity").Scan(&remaining)
	if remaining != 2 {
		t.Errorf("inventory after confirm = %d, want 2 (5 − 3)", remaining)
	}
	var status string
	db.Table("stock_reservations").Where("cart_id = ?", res.CartID).Select("status").Scan(&status)
	if status != "confirmed" {
		t.Errorf("reservation status = %q, want confirmed", status)
	}
}

func TestReleaseReservations_FreesStockForNextCart(t *testing.T) {
	db := setupReservationDB(t)
	_, _, medID := seedTwoPharmacies(t, db, 3, 0)

	first, err := pharmacy.PickNearestPharmacyAndReserve(context.Background(), db,
		7, 13.083, 80.275,
		[]pharmacy.ReservationItem{{MedicineID: medID, Quantity: 3}},
		15*time.Minute)
	if err != nil {
		t.Fatalf("first reserve: %v", err)
	}

	// Patient cancels — release the hold.
	if err := pharmacy.ReleaseReservations(context.Background(), db, first.CartID); err != nil {
		t.Fatalf("release: %v", err)
	}

	// A new cart now must succeed for the full 3 units.
	if _, err := pharmacy.PickNearestPharmacyAndReserve(context.Background(), db,
		8, 13.083, 80.275,
		[]pharmacy.ReservationItem{{MedicineID: medID, Quantity: 3}},
		15*time.Minute); err != nil {
		t.Fatalf("second cart should succeed after release; got %v", err)
	}
}
