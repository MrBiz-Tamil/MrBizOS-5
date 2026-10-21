package integration

import (
	"testing"

	"drayya-core/internal/domain/finance"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupFinanceTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to memory db: %v", err)
	}

	if err := finance.AutoMigrate(db); err != nil {
		t.Fatalf("finance automigrate failed: %v", err)
	}

	return db
}

func TestGroupProfitabilityAggregator(t *testing.T) {
	db := setupFinanceTestDB(t)
	svc := finance.NewService(finance.NewRepository(db))

	// 1. Record Ledger Entries for BU 1 (Travel)
	_, _ = svc.RecordEntry(&finance.FinancialLedgerEntry{BusinessUnitID: 1, EntryType: finance.EntryRevenue, Amount: 150000.00, Description: "Travel & Visa Booking Fees"})
	_, _ = svc.RecordEntry(&finance.FinancialLedgerEntry{BusinessUnitID: 1, EntryType: finance.EntryCOGS, Amount: 45000.00, Description: "Airline & Hotel Vendor Direct Costs"})
	_, _ = svc.RecordEntry(&finance.FinancialLedgerEntry{BusinessUnitID: 1, EntryType: finance.EntryAdvertising, Amount: 12000.00, Description: "Google Ads Travel Campaigns"})

	// 2. Record Ledger Entries for BU 2 (Healthcare)
	_, _ = svc.RecordEntry(&finance.FinancialLedgerEntry{BusinessUnitID: 2, EntryType: finance.EntryRevenue, Amount: 320000.00, Description: "AYUSH Consultations & Lab Revenue"})
	_, _ = svc.RecordEntry(&finance.FinancialLedgerEntry{BusinessUnitID: 2, EntryType: finance.EntryExpense, Amount: 95000.00, Description: "Doctor Compensation & Lab Operations"})

	// 3. Record Ledger Entries for BU 5 (Software Consulting)
	_, _ = svc.RecordEntry(&finance.FinancialLedgerEntry{BusinessUnitID: 5, EntryType: finance.EntryRevenue, Amount: 240000.00, Description: "Enterprise SOW Software Contracts"})
	_, _ = svc.RecordEntry(&finance.FinancialLedgerEntry{BusinessUnitID: 5, EntryType: finance.EntryExpense, Amount: 80000.00, Description: "Engineering Team Payroll"})

	// 4. Generate Consolidated Group Profitability Report
	report, err := svc.GetGroupProfitabilityReport()
	if err != nil {
		t.Fatalf("GetGroupProfitabilityReport failed: %v", err)
	}

	// Total Gross Revenue = $150k + $320k + $240k = $710,000
	if report.TotalGrossRevenue != 710000.00 {
		t.Fatalf("expected total revenue 710000.00, got %.2f", report.TotalGrossRevenue)
	}

	// Total Group Expenses = ($45k + $12k) + $95k + $80k = $232,000
	if report.TotalGroupExpenses != 232000.00 {
		t.Fatalf("expected total expenses 232000.00, got %.2f", report.TotalGroupExpenses)
	}

	// Net Profit = $710,000 - $232,000 = $478,000
	if report.NetGroupProfit != 478000.00 {
		t.Fatalf("expected net group profit 478000.00, got %.2f", report.NetGroupProfit)
	}

	// Net Group Profit Margin = (478000 / 710000) * 100 = 67.32%
	if report.NetGroupProfitMargin < 67.3 || report.NetGroupProfitMargin > 67.4 {
		t.Fatalf("expected profit margin ~67.32%%, got %.2f%%", report.NetGroupProfitMargin)
	}
}
