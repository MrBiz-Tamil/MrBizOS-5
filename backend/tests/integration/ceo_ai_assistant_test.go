package integration

import (
	"testing"

	"drayya-core/internal/domain/ceo_ai"
	"drayya-core/internal/domain/finance"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCEOAITestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to memory db: %v", err)
	}

	if err := finance.AutoMigrate(db); err != nil {
		t.Fatalf("finance automigrate failed: %v", err)
	}

	return db
}

func TestCEOAIAssistantExecutiveQueries(t *testing.T) {
	db := setupCEOAITestDB(t)

	financeSvc := finance.NewService(finance.NewRepository(db))
	ceoSvc := ceo_ai.NewService(ceo_ai.NewRepository(db), financeSvc)

	// Seed Financial Ledger
	_, _ = financeSvc.RecordEntry(&finance.FinancialLedgerEntry{BusinessUnitID: 2, EntryType: finance.EntryRevenue, Amount: 500000.00, Description: "Group Healthcare Revenue"})
	_, _ = financeSvc.RecordEntry(&finance.FinancialLedgerEntry{BusinessUnitID: 2, EntryType: finance.EntryExpense, Amount: 150000.00, Description: "Clinical Operations Expense"})

	// 1. Executive Query: Group Financial Profitability
	resp, err := ceoSvc.QueryExecutiveAssistant(&ceo_ai.ExecutiveQueryRequest{
		Query:  "What is our total group revenue and profit margin?",
		UserID: 1,
	})
	if err != nil {
		t.Fatalf("QueryExecutiveAssistant failed: %v", err)
	}
	if resp.Answer == "" {
		t.Fatalf("expected non-empty executive answer")
	}

	// 2. Executive Query: Travel & Visa Status
	respTravel, err := ceoSvc.QueryExecutiveAssistant(&ceo_ai.ExecutiveQueryRequest{
		Query:  "Summarize pending visa applications and travel bookings.",
		UserID: 1,
	})
	if err != nil || respTravel.BusinessUnit != "Travel, Tours & Visa Ops" {
		t.Fatalf("expected Travel BU query response")
	}

	// 3. Executive Query: Real Estate Inventory
	respRE, err := ceoSvc.QueryExecutiveAssistant(&ceo_ai.ExecutiveQueryRequest{
		Query:  "Show me active real estate property listings and site visits.",
		UserID: 1,
	})
	if err != nil || respRE.BusinessUnit != "Real Estate & Property Inventory" {
		t.Fatalf("expected Real Estate BU query response")
	}
}
