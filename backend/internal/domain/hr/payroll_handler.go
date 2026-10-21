package hr

import (
	"drayya-core/internal/database"
	"drayya-core/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// RegisterPayrollRoutes wires payroll compliance endpoints.
func RegisterPayrollRoutes(router fiber.Router) {
	payroll := router.Group("/hr/payroll", middleware.Protected(), middleware.RequireRoles("admin", "super_admin", "hr_manager"))

	payroll.Post("/compute", computePayroll)
	payroll.Post("/:id/disburse", disbursePayroll)
	payroll.Get("/summary", payrollSummary)
	payroll.Get("/epf-statement", epfStatement)
	payroll.Get("/esi-statement", esiStatement)
	// Freelancer (panel doctor) compliance.
	payroll.Get("/tds-certificate", tdsCertificate) // Form 16A per doctor
	payroll.Get("/tds/26q", form26QExport)          // quarterly 26Q return
	payroll.Get("/tds/remittances", listTDSRemittances)
	payroll.Post("/tds/remittances", recordTDSRemittance)

	// Printable documents (browser print-to-PDF).
	payroll.Get("/payslip-doc", payslipDocument)
	payroll.Get("/tds-certificate-doc", tdsCertificateDocument)

	// CSV exports.
	payroll.Get("/export/epf", exportEPFCSV)
	payroll.Get("/export/esi", exportESICSV)
	payroll.Get("/export/26q", export26QCSV)
	payroll.Get("/export/attendance", exportAttendanceCSV)

	// Employee-facing payslips (self-service)
	self := router.Group("/hr/payslips", middleware.Protected())
	self.Get("/", getMyPayslips)
	self.Get("/:month", getMyPayslip)
}

type ComputePayrollRequest struct {
	UserID         uint    `json:"user_id"`
	Month          string  `json:"month"` // "2026-03"
	GrossPaise     int64   `json:"gross_paise"`
	BasicPercent   float64 `json:"basic_percent"` // e.g. 0.5
	MonthlyTDSPaise int64  `json:"monthly_tds_paise"`
	// Payslip metadata
	EmployeeName string  `json:"employee_name"`
	Designation  string  `json:"designation"`
	PAN          string  `json:"pan"`
	UAN          string  `json:"uan"`
	ESINumber    string  `json:"esi_number"`
	PaidDays     int     `json:"paid_days"`
	LopDays      float64 `json:"lop_days"`
}

func computePayroll(c *fiber.Ctx) error {
	var req ComputePayrollRequest
	if err := c.BodyParser(&req); err != nil || req.GrossPaise <= 0 || req.Month == "" {
		return c.Status(400).JSON(fiber.Map{"error": "gross_paise and month are required"})
	}
	if req.PaidDays == 0 {
		req.PaidDays = 26 // default working days
	}

	// Freelance panel doctors must never receive a payslip with PF/ESI/PT — they
	// are revenue-share vendors. Their compliance artifact is the TDS certificate
	// (Form 16A) at /hr/payroll/tds-certificate.
	var profile StaffProfile
	if err := database.DB.Where("user_id = ?", req.UserID).First(&profile).Error; err == nil && profile.IsFreelancer() {
		return c.Status(400).JSON(fiber.Map{
			"error": "user is a freelancer (revenue-share); payslips do not apply — use /hr/payroll/tds-certificate",
		})
	}

	pc := ComputePayrollCompliance(req.UserID, req.Month, req.GrossPaise, req.BasicPercent, req.MonthlyTDSPaise)
	if err := SavePayrollCompliance(pc); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Generate payslip
	ps, err := GeneratePayslip(pc, req.EmployeeName, req.Designation, req.PAN, req.UAN, req.ESINumber, req.PaidDays, req.LopDays)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "compliance saved but payslip failed: " + err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{
		"compliance": pc,
		"payslip":    ps,
	})
}

func payrollSummary(c *fiber.Ctx) error {
	month := c.Query("month")
	if month == "" {
		return c.Status(400).JSON(fiber.Map{"error": "month query param required (YYYY-MM)"})
	}
	summary, err := GetPayrollSummary(month)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(summary)
}

func epfStatement(c *fiber.Ctx) error {
	month := c.Query("month")
	if month == "" {
		return c.Status(400).JSON(fiber.Map{"error": "month required"})
	}
	data, err := GenerateEPFStatement(month)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(data)
}

func esiStatement(c *fiber.Ctx) error {
	month := c.Query("month")
	if month == "" {
		return c.Status(400).JSON(fiber.Map{"error": "month required"})
	}
	data, err := GenerateESIStatement(month)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(data)
}

func getMyPayslips(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	slips, err := GetAllPayslips(userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(slips)
}

func getMyPayslip(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	month := c.Params("month")
	ps, err := GetPayslip(userID, month)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "payslip not found"})
	}
	return c.JSON(ps)
}
