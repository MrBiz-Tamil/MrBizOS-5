package pharmacy_test

import (
	"testing"

	"drayya-core/internal/domain/pharmacy"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSeedAYUSHCatalog_IsIdempotentAndCoversAllCategories(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&pharmacy.Medicine{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pharmacy.SeedAYUSHCatalog(db)

	type cat struct {
		Category string
		N        int64
	}
	var counts []cat
	db.Model(&pharmacy.Medicine{}).Select("category, COUNT(*) AS n").Group("category").Scan(&counts)

	want := map[string]bool{"siddha": false, "ayurveda": false, "wellness": false}
	totalFirst := int64(0)
	for _, c := range counts {
		if _, ok := want[c.Category]; ok {
			want[c.Category] = true
		}
		totalFirst += c.N
	}
	for k, seeded := range want {
		if !seeded {
			t.Errorf("category %q not seeded", k)
		}
	}
	if totalFirst == 0 {
		t.Fatal("catalog seed produced zero rows")
	}

	// Re-running must be a no-op — that's the safety guarantee on every restart.
	pharmacy.SeedAYUSHCatalog(db)
	var totalAfter int64
	db.Model(&pharmacy.Medicine{}).Count(&totalAfter)
	if totalAfter != totalFirst {
		t.Errorf("seed not idempotent: %d rows after first run, %d after second", totalFirst, totalAfter)
	}

	// At least one in-house product per category — otherwise the autocomplete's
	// "in-house first" ordering has nothing to surface.
	var inhouseCount int64
	db.Model(&pharmacy.Medicine{}).Where("is_inhouse_brand = ?", true).Count(&inhouseCount)
	if inhouseCount < 3 {
		t.Errorf("expected ≥3 Dr.Ayya in-house SKUs across the 3 categories; got %d", inhouseCount)
	}
}
