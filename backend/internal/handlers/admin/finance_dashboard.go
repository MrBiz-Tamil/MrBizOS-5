package admin

import (
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/billing"
	"drayya-core/internal/domain/corporate"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/franchise"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/services/financials"
	"drayya-core/internal/utils/config"
	"drayya-core/internal/utils/response"
	"time"

	"github.com/gofiber/fiber/v2"
)

type FinanceDashboardOverview struct {
	TotalRevenue        float64 `json:"total_revenue"`
	MonthlyRevenue      float64 `json:"monthly_revenue"`
	PendingPayouts      float64 `json:"pending_payouts"`
	OutstandingInvoices int64   `json:"outstanding_invoices"`
}

func GetFinanceDashboardOverview(c *fiber.Ctx) error {
	db := database.DB
	var overview FinanceDashboardOverview

	// 1. Total Consultation Revenue
	platformRev, _ := financials.GetAccountBalance(c.Context(), "Revenue:Consultations")
	overview.TotalRevenue = float64(platformRev) / 100

	// 2. Monthly Revenue (Current Month) — anchor the month boundary to the
	// configured business timezone so it's deterministic regardless of server TZ.
	loc := config.LoadTimezone()
	now := time.Now().In(loc)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	var monthlyRev int64
	db.Model(&finance.LedgerEntry{}).
		Joins("JOIN ledger_account_v2 ON ledger_entry_v2.account_id = ledger_account_v2.id").
		Where("ledger_account_v2.name = ? AND ledger_entry_v2.direction = ? AND ledger_entry_v2.created_at >= ?",
			"Revenue:Consultations", finance.DirectionCredit, monthStart).
		Select("COALESCE(SUM(ledger_entry_v2.amount), 0)").
		Scan(&monthlyRev)
	overview.MonthlyRevenue = float64(monthlyRev) / 100

	// 3. Outstanding Invoices (Pending Payments)
	db.Model(&billing.Payment{}).Where("status = ?", "pending").Count(&overview.OutstandingInvoices)

	// 4. Pending Payouts (Liability to Doctors)
	var totalPayouts int64
	db.Model(&finance.LedgerAccount{}).
		Where("name LIKE ? AND type = ?", "Doctor:%", finance.TypeLiability).
		Select("COALESCE(SUM(balance), 0)").
		Scan(&totalPayouts)
	overview.PendingPayouts = float64(totalPayouts) / 100

	return c.JSON(overview)
}

func GetFinancialLedger(c *fiber.Ctx) error {
	var payments []billing.Payment
	database.DB.Order("created_at DESC").Limit(100).Find(&payments)
	return c.JSON(payments)
}

// ExecutePayouts settles all pending doctor balances
func ExecutePayouts(c *fiber.Ctx) error {
	db := database.DB
	
	// 1. Find all accounts with positive liability balance
	var accounts []finance.LedgerAccount
	db.Where("name LIKE ? AND type = ? AND balance > 0", "Doctor:%", finance.TypeLiability).Find(&accounts)

	if len(accounts) == 0 {
		return response.JSON(c, fiber.StatusOK, nil, "No pending settlements found")
	}

	financeRepo := finance.NewRepository(db)
	financeSvc := finance.NewService(financeRepo)

	var totalSettled int64
	count := 0

	for _, acc := range accounts {
		var doctorID uint
		if _, err := fmt.Sscanf(acc.Name, "Doctor:%d", &doctorID); err == nil {
			// Create actual payout requests for approval
			_, err := financeSvc.CreatePayoutRequest(doctorID, acc.Balance)
			if err == nil {
				totalSettled += acc.Balance
				count++
			}
		}
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"count":         count,
		"total_amount":  float64(totalSettled) / 100,
		"message":       fmt.Sprintf("Created %d pending payout requests for admin approval.", count),
	}, "Payout requests generated")
}

// RunLedgerReconciliation runs all ledger-integrity checks on demand and returns
// the report. Same checks the nightly cron runs; lets an admin verify the books
// immediately (and powers the Control Tower's "run now" button).
func RunLedgerReconciliation(c *fiber.Ctx) error {
	finSvc := finance.NewService(finance.NewRepository(database.DB))
	report := finSvc.Reconcile(database.DB)

	status := "healthy"
	if !report.AllPassed {
		status = "exceptions_found"
	}
	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"status":     status,
		"all_passed": report.AllPassed,
		"ran_at":     report.RanAt,
		"checks":     report.Results,
	}, "Ledger reconciliation complete")
}

func ExportTDSReport(c *fiber.Ctx) error {
	// Real logic: Query TDS liability account and list deductions
	var totalTDS int64
	database.DB.Model(&finance.LedgerAccount{}).Where("name = ?", "Liability:Tax:TDS").Select("balance").Scan(&totalTDS)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"total_tds_payable": float64(totalTDS) / 100,
		"period":            "Q1 2026",
		"status":            "Draft report generated",
	}, "TDS Report generated")
}

func GetProjectPartners(c *fiber.Ctx) error {
	var companies []corporate.Company
	database.DB.Find(&companies)

	partners := make([]fiber.Map, 0, len(companies))
	for _, comp := range companies {
		partners = append(partners, fiber.Map{
			"id":             comp.ID,
			"name":           comp.Name,
			"type":           "Corporate Partner",
			"revenue_share":  "Variable",
			"pending_payout": comp.WalletBalance,
			"status":         "active",
		})
	}
	return c.JSON(partners)
}

func GetInvestors(c *fiber.Ctx) error {
	var investors []finance.Investor
	database.DB.Find(&investors)

	if len(investors) == 0 {
		// Return empty list instead of mock if none found
		return c.JSON([]interface{}{})
	}
	return c.JSON(investors)
}

func GetFranchiseDeals(c *fiber.Ctx) error {
	var franchises []franchise.Franchise
	database.DB.Preload("Owner").Find(&franchises)

	deals := make([]fiber.Map, 0, len(franchises))
	for _, f := range franchises {
		deals = append(deals, fiber.Map{
			"id":               f.ID,
			"location":         f.Region,
			"owner":            f.Owner.FullName,
			"royalty_rate":    fmt.Sprintf("%.1f%%", f.RoyaltyRate),
			"status":           map[bool]string{true: "active", false: "inactive"}[f.IsActive],
		})
	}
	return c.JSON(deals)
}

func GetPharmacySuppliers(c *fiber.Ctx) error {
	var suppliers []pharmacy.Supplier
	database.DB.Find(&suppliers)

	list := make([]fiber.Map, 0, len(suppliers))
	for _, s := range suppliers {
		list = append(list, fiber.Map{
			"id":      s.ID,
			"name":    s.Name,
			"email":   s.ContactEmail,
			"phone":   s.Phone,
			"status":  map[bool]string{true: "active", false: "inactive"}[s.IsActive],
		})
	}
	return c.JSON(list)
}
