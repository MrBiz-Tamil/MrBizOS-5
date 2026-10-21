package financials

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"drayya-core/internal/middleware"
)

// RegisterReportRoutes adds financial report endpoints under /admin/finance.
func RegisterReportRoutes(router fiber.Router) {
	r := router.Group("/admin/finance", middleware.Protected(), middleware.RequireRoles("admin", "super_admin", "finance_manager"))

	r.Get("/pl", plReport)
	r.Get("/balance-sheet", balanceSheetReport)
	r.Get("/cash-flow", cashFlowReport)
	r.Get("/kpis", revenueKPIs)
}

func plReport(c *fiber.Ctx) error {
	period := parsePeriod(c)
	report, err := GeneratePLReport(period)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(report)
}

func balanceSheetReport(c *fiber.Ctx) error {
	asAtStr := c.Query("as_at", time.Now().Format("2006-01-02"))
	asAt, err := time.Parse("2006-01-02", asAtStr)
	if err != nil {
		asAt = time.Now()
	}
	report, err := GenerateBalanceSheet(asAt)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(report)
}

func cashFlowReport(c *fiber.Ctx) error {
	period := parsePeriod(c)
	report, err := GenerateCashFlowReport(period)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(report)
}

func revenueKPIs(c *fiber.Ctx) error {
	period := parsePeriod(c)
	kpis, err := GetRevenueKPIs(period)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(kpis)
}

func parsePeriod(c *fiber.Ctx) PeriodFilter {
	preset := c.Query("preset") // "this_month" | "this_fy"
	if preset == "this_fy" {
		return ThisFY()
	}
	if preset == "this_month" || preset == "" {
		fromStr := c.Query("from", "")
		toStr := c.Query("to", "")
		if fromStr != "" && toStr != "" {
			from, err1 := time.Parse("2006-01-02", fromStr)
			to, err2 := time.Parse("2006-01-02", toStr)
			if err1 == nil && err2 == nil {
				return PeriodFilter{
					From:  from,
					To:    to.Add(23*time.Hour + 59*time.Minute + 59*time.Second),
					Label: fromStr + " to " + toStr,
				}
			}
		}
	}
	return ThisMonth()
}
