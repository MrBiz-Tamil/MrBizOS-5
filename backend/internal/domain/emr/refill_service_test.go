package emr

import (
	"context"
	"testing"
	"time"

	"drayya-core/internal/database"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupRefillDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&Prescription{}, &PrescriptionItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.DB = db
	return db
}

func TestComputeAdherence_Ratio(t *testing.T) {
	now := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	twoDaysAgo := now.Add(-48 * time.Hour)

	cases := []struct {
		name      string
		item      PrescriptionItem
		wantPct   float64
		wantOverdue bool
	}{
		{
			"all refills used — 100% adherence, not overdue",
			PrescriptionItem{RefillsAllowed: 3, RefillsUsed: 3},
			100, false,
		},
		{
			"halfway — 50%",
			PrescriptionItem{RefillsAllowed: 4, RefillsUsed: 2},
			50, false,
		},
		{
			"never refilled — 0%",
			PrescriptionItem{RefillsAllowed: 3, RefillsUsed: 0},
			0, false,
		},
		{
			"next refill was due 2 days ago and we still have refills — overdue",
			PrescriptionItem{RefillsAllowed: 3, RefillsUsed: 1, NextRefillDueAt: &twoDaysAgo},
			float64(1) / 3 * 100, true,
		},
		{
			"due in the past but no refills left — not overdue (Rx exhausted)",
			PrescriptionItem{RefillsAllowed: 3, RefillsUsed: 3, NextRefillDueAt: &twoDaysAgo},
			100, false,
		},
		{
			"refills_allowed = 0 — divide-by-zero safety",
			PrescriptionItem{RefillsAllowed: 0, RefillsUsed: 0},
			0, false,
		},
	}
	for _, c := range cases {
		s := computeAdherence(c.item, now)
		if s.AdherencePercent < c.wantPct-0.01 || s.AdherencePercent > c.wantPct+0.01 {
			t.Errorf("%s: pct = %.2f, want %.2f", c.name, s.AdherencePercent, c.wantPct)
		}
		if s.IsOverdue != c.wantOverdue {
			t.Errorf("%s: overdue = %v, want %v", c.name, s.IsOverdue, c.wantOverdue)
		}
	}
}

func TestSweepRefillsDue_ReturnsWindowAndStampsCooldown(t *testing.T) {
	db := setupRefillDB(t)

	// Create one prescription owned by patient #7.
	db.Exec(`CREATE TABLE IF NOT EXISTS prescriptions (id INTEGER PRIMARY KEY, user_id INTEGER)`)
	db.Exec(`INSERT INTO prescriptions (id, user_id) VALUES (1, 7)`)

	now := time.Now()
	dueIn2Days := now.Add(48 * time.Hour)
	dueIn10Days := now.Add(10 * 24 * time.Hour) // outside the 3-day window

	// In-window, has refills left, never reminded → should sweep.
	db.Create(&PrescriptionItem{
		PrescriptionID: 1, MedicineID: 100, RefillsAllowed: 3, RefillsUsed: 1,
		NextRefillDueAt: &dueIn2Days,
	})
	// Out-of-window → should not sweep.
	db.Create(&PrescriptionItem{
		PrescriptionID: 1, MedicineID: 101, RefillsAllowed: 3, RefillsUsed: 0,
		NextRefillDueAt: &dueIn10Days,
	})
	// In-window but no refills left → should not sweep.
	db.Create(&PrescriptionItem{
		PrescriptionID: 1, MedicineID: 102, RefillsAllowed: 1, RefillsUsed: 1,
		NextRefillDueAt: &dueIn2Days,
	})
	// In-window but reminded an hour ago → cooldown blocks.
	recent := now.Add(-1 * time.Hour)
	db.Create(&PrescriptionItem{
		PrescriptionID: 1, MedicineID: 103, RefillsAllowed: 2, RefillsUsed: 0,
		NextRefillDueAt: &dueIn2Days, LastReminderSentAt: &recent,
	})

	items, err := SweepRefillsDue(context.Background(), db)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if len(items) != 1 || items[0].MedicineID != 100 {
		t.Fatalf("sweep should surface exactly the in-window, has-refills-left, not-recently-reminded item; got %+v", items)
	}
	if items[0].PatientID != 7 {
		t.Errorf("patient id should be joined from prescriptions.user_id = 7; got %d", items[0].PatientID)
	}

	// Second sweep within cooldown must not re-surface the same item.
	items2, err := SweepRefillsDue(context.Background(), db)
	if err != nil {
		t.Fatalf("re-sweep: %v", err)
	}
	if len(items2) != 0 {
		t.Errorf("cooldown should silence the second sweep; got %d items", len(items2))
	}
}
