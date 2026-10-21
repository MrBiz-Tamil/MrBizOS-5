package finance

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	
	err = db.AutoMigrate(&LedgerAccount{}, &LedgerTransaction{}, &LedgerEntry{}, &TaxRule{})
	assert.NoError(t, err)
	return db
}

func TestTaxEngine(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)

	// 1. Create Tax Rule (GST 18% for Consultations in TN)
	rule := TaxRule{
		Region:   "IN-TN",
		Category: "CONSULTATION",
		TaxType:  "GST",
		Rate:     18.0,
	}
	err := svc.CreateTaxRule(context.Background(), rule)
	assert.NoError(t, err)

	// 2. Calculate Tax
	tax, err := svc.CalculateTax(context.Background(), 100000, "IN-TN", "CONSULTATION") // 1000 INR
	assert.NoError(t, err)
	assert.Equal(t, int64(18000), tax) // 180 INR = 18000 paise

	// 3. Test non-existent rule
	tax2, err := svc.CalculateTax(context.Background(), 100000, "US-CA", "SERVICE")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), tax2)
}

func TestDynamicPricing(t *testing.T) {
	repo := &financeRepository{} // Mock-ish
	svc := NewService(repo)

	// Low demand
	m1 := svc.GetDynamicPriceMultiplier(context.Background(), "GENERAL", 5)
	assert.Equal(t, 1.05, m1)

	// High demand
	m2 := svc.GetDynamicPriceMultiplier(context.Background(), "GENERAL", 50)
	assert.Equal(t, 1.5, m2)

	// Max demand (capped)
	m3 := svc.GetDynamicPriceMultiplier(context.Background(), "GENERAL", 200)
	assert.Equal(t, 2.0, m3)
}
