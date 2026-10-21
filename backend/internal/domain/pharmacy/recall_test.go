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

func setupRecallDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&pharmacy.PharmacyOrder{}, &pharmacy.PharmacyOrderItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Hand-create the user table the join expects (avoid pulling identity model).
	db.Exec(`CREATE TABLE "user" (id INTEGER PRIMARY KEY, full_name TEXT, phone_number TEXT, email TEXT)`)
	return db
}

func TestLookupOrderItemsByBatch_AggregatesAffectedPatients(t *testing.T) {
	db := setupRecallDB(t)
	// 3 patients.
	db.Exec(`INSERT INTO "user" (id, full_name, phone_number, email) VALUES
		(7,  'Alice',   '9991110001', 'alice@example.com'),
		(8,  'Bob',     '9991110002', 'bob@example.com'),
		(9,  'Charlie', '9991110003', 'charlie@example.com')`)

	// Each patient places an order containing the same medicine; only Alice and
	// Bob receive the recalled batch. Charlie got a different batch.
	for _, o := range []pharmacy.PharmacyOrder{
		{ID: 1, UserID: 7, OrderNumber: "RX-0001"},
		{ID: 2, UserID: 8, OrderNumber: "RX-0002"},
		{ID: 3, UserID: 9, OrderNumber: "RX-0003"},
	} {
		o := o
		db.Create(&o)
	}
	exp := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	db.Create(&pharmacy.PharmacyOrderItem{OrderID: 1, MedicineID: 42, BatchNumber: "B-RECALL", Quantity: 2, ExpiryDate: &exp})
	db.Create(&pharmacy.PharmacyOrderItem{OrderID: 2, MedicineID: 42, BatchNumber: "B-RECALL", Quantity: 1, ExpiryDate: &exp})
	db.Create(&pharmacy.PharmacyOrderItem{OrderID: 3, MedicineID: 42, BatchNumber: "B-SAFE", Quantity: 3, ExpiryDate: &exp})

	affected, err := pharmacy.LookupOrderItemsByBatch(context.Background(), db, "B-RECALL", 42)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(affected) != 2 {
		t.Fatalf("expected 2 affected order items, got %d", len(affected))
	}
	gotNames := map[string]bool{}
	for _, a := range affected {
		gotNames[a.PatientName] = true
		if a.Phone == "" || a.Email == "" {
			t.Errorf("affected patient %s missing contact info: %+v", a.PatientName, a)
		}
	}
	if !gotNames["Alice"] || !gotNames["Bob"] {
		t.Errorf("expected Alice + Bob; got %+v", gotNames)
	}
	if gotNames["Charlie"] {
		t.Error("Charlie received the safe batch and must NOT appear in the recall")
	}
}

func TestLookupOrderItemsByBatch_RequiresBatch(t *testing.T) {
	db := setupRecallDB(t)
	if _, err := pharmacy.LookupOrderItemsByBatch(context.Background(), db, "  ", 0); err == nil {
		t.Fatal("blank batch_no should fail validation")
	}
}

func TestAttachBatchToOrderItem_RecordsBatchAndExpiry(t *testing.T) {
	db := setupRecallDB(t)
	db.Exec(`INSERT INTO pharmacy_orders (id, user_id, order_number) VALUES (1, 7, 'RX-9')`)
	db.Create(&pharmacy.PharmacyOrderItem{ID: 100, OrderID: 1, MedicineID: 0, Quantity: 1})

	exp := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := pharmacy.AttachBatchToOrderItem(context.Background(), db, 100, 42, "B-2026A", &exp); err != nil {
		t.Fatalf("attach: %v", err)
	}
	var got pharmacy.PharmacyOrderItem
	db.First(&got, 100)
	if got.BatchNumber != "B-2026A" {
		t.Errorf("batch = %q, want B-2026A", got.BatchNumber)
	}
	if got.MedicineID != 42 {
		t.Errorf("medicine_id = %d, want 42", got.MedicineID)
	}
	if got.ExpiryDate == nil || !got.ExpiryDate.Equal(exp) {
		t.Errorf("expiry not recorded: %+v", got.ExpiryDate)
	}
}
