package tds

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"drayya-core/internal/middleware"
)

// RegisterTDSRoutes wires TDS computation and certificate endpoints.
func RegisterTDSRoutes(router fiber.Router) {
	t := router.Group("/admin/tds", middleware.Protected(), middleware.RequireRoles("admin", "super_admin", "finance_manager"))

	t.Post("/compute-salary", computeSalaryTDS)
	t.Post("/compute-194j", compute194J)
	t.Post("/form16a", generateForm16AHandler)
	t.Post("/form26q", generateForm26QHandler)
}

func computeSalaryTDS(c *fiber.Ctx) error {
	var req struct {
		AnnualIncomePaise int64 `json:"annual_income_paise"`
	}
	if err := c.BodyParser(&req); err != nil || req.AnnualIncomePaise <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "annual_income_paise is required"})
	}
	annual := ComputeAnnualTax(req.AnnualIncomePaise)
	monthly := MonthlyTDS(req.AnnualIncomePaise)
	return c.JSON(fiber.Map{
		"annual_income_paise": req.AnnualIncomePaise,
		"annual_tax_paise":    annual,
		"monthly_tds_paise":   monthly,
		"section":             Section192,
		"fy":                  FYFromDate(time.Now()),
	})
}

func compute194J(c *fiber.Ctx) error {
	var req struct {
		AmountPaise int64 `json:"amount_paise"`
	}
	if err := c.BodyParser(&req); err != nil || req.AmountPaise <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "amount_paise is required"})
	}
	tdsAmt := TDSOnProfessionalFee(req.AmountPaise)
	return c.JSON(fiber.Map{
		"amount_paise":  req.AmountPaise,
		"tds_paise":     tdsAmt,
		"rate":          "10%",
		"section":       Section194J,
	})
}

type Form16ARequest struct {
	DeductorName  string         `json:"deductor_name"`
	TAN           string         `json:"tan"`
	GSTIN         string         `json:"gstin,omitempty"`
	DeducteeName  string         `json:"deductee_name"`
	DeducteePAN   string         `json:"deductee_pan"`
	FinancialYear string         `json:"financial_year"`
	Quarter       string         `json:"quarter"`
	Entries       []Form16AEntry `json:"entries"`
}

func generateForm16AHandler(c *fiber.Ctx) error {
	var req Form16ARequest
	if err := c.BodyParser(&req); err != nil || req.TAN == "" {
		return c.Status(400).JSON(fiber.Map{"error": "tan and entries are required"})
	}
	cert := GenerateForm16A(req.DeductorName, req.TAN, req.GSTIN, req.DeducteeName, req.DeducteePAN, req.FinancialYear, req.Quarter, req.Entries)
	return c.JSON(cert)
}

type Form26QRequest struct {
	TAN          string       `json:"tan"`
	DeductorName string       `json:"deductor_name"`
	FY           string       `json:"financial_year"`
	Quarter      string       `json:"quarter"`
	Rows         []Form26QRow `json:"rows"`
}

func generateForm26QHandler(c *fiber.Ctx) error {
	var req Form26QRequest
	if err := c.BodyParser(&req); err != nil || req.TAN == "" {
		return c.Status(400).JSON(fiber.Map{"error": "tan and rows are required"})
	}
	ret := GenerateForm26Q(req.TAN, req.DeductorName, req.FY, req.Quarter, req.Rows)
	return c.JSON(ret)
}

