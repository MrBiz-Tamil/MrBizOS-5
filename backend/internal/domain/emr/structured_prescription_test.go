package emr

import (
	"context"
	"testing"

	"drayya-core/internal/database"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// rawMedicine is a thin stand-in for pharmacy.Medicine — we can't import the
// pharmacy package here without creating an emr ← pharmacy ← emr cycle. The
// `medicine` table is created by hand with the columns writePrescriptionItems
// validates against.
type rawMedicine struct {
	ID       uint `gorm:"primaryKey"`
	Name     string
	IsActive bool
}

func (rawMedicine) TableName() string { return "medicine" }

func setupStructuredRxDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&PrescriptionItem{}, &rawMedicine{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.DB = db
	return db
}

func TestWritePrescriptionItems_PersistsAllRowsWithComputedTotal(t *testing.T) {
	db := setupStructuredRxDB(t)
	db.Create(&rawMedicine{Name: "Nilavembu Kashayam", IsActive: true})
	db.Create(&rawMedicine{Name: "Triphala Churna", IsActive: true})

	svc := &service{repo: NewRepository(db)}
	err := svc.writePrescriptionItems(context.Background(), 1, []PrescriptionItemInput{
		// Item 1: 1 × 2 × 7 = 14
		{MedicineID: 1, DosageText: "1 tsp twice a day after meals",
			QuantityPerDose: 1, FrequencyPerDay: 2, DurationDays: 7, RefillsAllowed: 0},
		// Item 2: 1 × 1 × 30 = 30; refills carry over
		{MedicineID: 2, DosageText: "1 tsp at bedtime",
			QuantityPerDose: 1, FrequencyPerDay: 1, DurationDays: 30, RefillsAllowed: 2},
	})
	if err != nil {
		t.Fatalf("writePrescriptionItems: %v", err)
	}

	var items []PrescriptionItem
	db.Where("prescription_id = ?", 1).Order("id").Find(&items)
	if len(items) != 2 {
		t.Fatalf("expected 2 PrescriptionItem rows, got %d", len(items))
	}
	if items[0].TotalQuantity != 14 {
		t.Errorf("item 0 total = %v, want 14 (qty 1 × freq 2 × dur 7)", items[0].TotalQuantity)
	}
	if items[1].TotalQuantity != 30 {
		t.Errorf("item 1 total = %v, want 30", items[1].TotalQuantity)
	}
	if items[1].RefillsAllowed != 2 {
		t.Errorf("item 1 refills = %d, want 2", items[1].RefillsAllowed)
	}
}

func TestWritePrescriptionItems_RejectsUnknownMedicineID(t *testing.T) {
	db := setupStructuredRxDB(t)
	db.Create(&rawMedicine{Name: "Tulsi Drops", IsActive: true})

	svc := &service{repo: NewRepository(db)}
	err := svc.writePrescriptionItems(context.Background(), 1,
		[]PrescriptionItemInput{
			{MedicineID: 1, DurationDays: 5, QuantityPerDose: 1, FrequencyPerDay: 1}, // valid
			{MedicineID: 999, DurationDays: 5, QuantityPerDose: 1, FrequencyPerDay: 1}, // unknown
		})
	if err == nil {
		t.Fatal("expected validation error for unknown medicine_id; got nil")
	}
}

func TestWritePrescriptionItems_RejectsInactiveMedicine(t *testing.T) {
	db := setupStructuredRxDB(t)
	db.Create(&rawMedicine{Name: "Active", IsActive: true})
	db.Create(&rawMedicine{Name: "Inactive", IsActive: false})

	svc := &service{repo: NewRepository(db)}
	err := svc.writePrescriptionItems(context.Background(), 1,
		[]PrescriptionItemInput{
			{MedicineID: 2, DurationDays: 5, QuantityPerDose: 1, FrequencyPerDay: 1},
		})
	if err == nil {
		t.Fatal("expected validation error for inactive medicine; got nil")
	}
}
