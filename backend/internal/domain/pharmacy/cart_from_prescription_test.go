package pharmacy_test

import (
	"context"
	"testing"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/money"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupCartDB(t *testing.T) {
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
	// Migrate only what this test needs.
	if err := db.AutoMigrate(&pharmacy.Medicine{}, &pharmacy.PharmacyInventory{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// PrescriptionItem lives in the emr domain; we recreate the table by hand to
	// avoid an import cycle in this test file.
	if err := db.Exec(`
		CREATE TABLE prescription_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			prescription_id INTEGER,
			medicine_id INTEGER,
			total_quantity REAL,
			substituted_medicine_id INTEGER,
			deleted_at DATETIME
		)`).Error; err != nil {
		t.Fatalf("create prescription_items: %v", err)
	}
	database.DB = db
}

func TestGenerateCartFromPrescription_StructuredItems(t *testing.T) {
	setupCartDB(t)
	db := database.DB

	// Two AYUSH medicines: one with stock, one out of stock.
	medA := pharmacy.Medicine{
		Name: "Nilavembu Kashayam", NameTa: "நிலவேம்பு கசாயம்",
		Form: "kashayam", PackSize: "100 ml",
		Prices: money.RegionalPricing{"INR": 12000}, GSTPercent: 5, IsActive: true,
	}
	medB := pharmacy.Medicine{
		Name: "Triphala Churna", NameHi: "त्रिफला चूर्ण",
		Form: "churna", PackSize: "100 g",
		Prices: money.RegionalPricing{"INR": 8000}, GSTPercent: 5, IsActive: true,
	}
	db.Create(&medA)
	db.Create(&medB)
	// medA: 5 units in stock.
	if err := db.Create(&pharmacy.PharmacyInventory{MedicineID: medA.ID, StockQuantity: 5, IsActive: true}).Error; err != nil {
		t.Fatalf("create inventory: %v", err)
	}
	// Sanity: confirm the row landed before testing the cart logic itself.
	var sanity int64
	db.Table("pharmacyinventory").Where("medicine_id = ? AND is_active = ?", medA.ID, true).
		Select("COALESCE(SUM(stock_quantity), 0)").Scan(&sanity)
	if sanity != 5 {
		t.Fatalf("inventory seed: expected stock 5, got %d", sanity)
	}
	// medB: 0 units (no inventory row).

	// Two structured items on a prescription: doctor wants 2 of A and 1 of B.
	db.Exec(`INSERT INTO prescription_items (prescription_id, medicine_id, total_quantity) VALUES (?, ?, ?)`,
		100, medA.ID, 2.0)
	db.Exec(`INSERT INTO prescription_items (prescription_id, medicine_id, total_quantity) VALUES (?, ?, ?)`,
		100, medB.ID, 1.0)

	svc := pharmacy.NewService(pharmacy.NewDefaultRepository(db), finance.NewService(finance.NewRepository(db)))
	cart, err := svc.GenerateCartFromPrescription(context.Background(), 100)
	if err != nil {
		t.Fatalf("GenerateCart: %v", err)
	}
	if len(cart) != 2 {
		t.Fatalf("expected 2 cart lines, got %d", len(cart))
	}

	line0 := cart[0].(pharmacy.CartLine)
	if line0.MedicineID != medA.ID {
		t.Errorf("line 0 medicine = %d, want %d", line0.MedicineID, medA.ID)
	}
	if line0.LineTotalPaise != 24000 { // 2 × 12000
		t.Errorf("line 0 total = %d paise, want 24000", line0.LineTotalPaise)
	}
	if line0.NameLocal != "நிலவேம்பு கசாயம்" {
		t.Errorf("Tamil localised name not surfaced: %q", line0.NameLocal)
	}
	if !line0.InStock {
		t.Error("line 0 should be in stock (5 >= 2)")
	}

	line1 := cart[1].(pharmacy.CartLine)
	if line1.MedicineID != medB.ID {
		t.Errorf("line 1 medicine = %d, want %d", line1.MedicineID, medB.ID)
	}
	if line1.NameLocal != "त्रिफला चूर्ण" {
		t.Errorf("Hindi localised name fallback not surfaced: %q", line1.NameLocal)
	}
	if line1.InStock {
		t.Error("line 1 should be out of stock (no inventory)")
	}
}

func TestGenerateCartFromPrescription_NoStructuredItems_FallsBack(t *testing.T) {
	setupCartDB(t)
	svc := pharmacy.NewService(pharmacy.NewDefaultRepository(database.DB), finance.NewService(finance.NewRepository(database.DB)))
	cart, err := svc.GenerateCartFromPrescription(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cart) != 0 {
		t.Errorf("legacy / unstructured prescription should yield an empty cart so the pharmacist enters items manually; got %d lines", len(cart))
	}
}
