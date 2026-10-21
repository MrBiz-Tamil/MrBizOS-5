package corporate

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"drayya-core/internal/middleware"
)

func RegisterRoutes(router fiber.Router) {
	group := router.Group("/corporate", middleware.Protected())

	// Super-admin / corporate-admin only operations
	group.Post("/companies", createCompany)
	group.Get("/companies", listCompanies)
	group.Get("/companies/:id", getCompany)
	
	group.Post("/companies/:id/employees", addEmployee)
	group.Get("/companies/:id/employees", listEmployees)
	group.Post("/companies/:id/employees/bulk", bulkUploadEmployees)

	// Wallet & billing
	group.Post("/companies/:id/topup", topUpWallet)
	group.Get("/companies/:id/ledger", getCreditLedger)
	group.Get("/companies/:id/health-report", getHealthReport)
	group.Get("/companies/:id/invoices", listInvoices)
	group.Post("/companies/:id/invoices/generate", generateInvoice)

	// Employee self-service
	group.Get("/me", getMyEmployeeProfile)

	// My-company convenience endpoints (auto-detect company from logged-in user)
	group.Get("/my-status", getMyStatus)
	group.Get("/my-company/employees", getMyCompanyEmployees)
	group.Get("/my-company/dashboard", getMyCompanyDashboard)
}

func createCompany(c *fiber.Ctx) error {
	var req CompanyInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	company, err := RegisterCompany(req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(company)
}

func listCompanies(c *fiber.Ctx) error {
	companies, err := GetCompanies()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(companies)
}

func getCompany(c *fiber.Ctx) error {
	id, _ := strconv.Atoi(c.Params("id"))
	company, err := GetCompanyByID(uint(id))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Company not found"})
	}
	return c.JSON(company)
}

func addEmployee(c *fiber.Ctx) error {
	companyID, _ := strconv.Atoi(c.Params("id"))
	var req EmployeeInput
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	req.CompanyID = uint(companyID)

	emp, err := AddEmployee(req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(emp)
}

func listEmployees(c *fiber.Ctx) error {
	companyID, _ := strconv.Atoi(c.Params("id"))
	employees, err := GetEmployees(uint(companyID))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(employees)
}

func topUpWallet(c *fiber.Ctx) error {
	companyID, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		AmountPaise int64  `json:"amount_paise"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil || req.AmountPaise <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "amount_paise must be positive"})
	}
	if err := TopUpCompanyWallet(uint(companyID), req.AmountPaise, req.Description); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "wallet topped up", "amount_paise": req.AmountPaise})
}

func getCreditLedger(c *fiber.Ctx) error {
	companyID, _ := strconv.Atoi(c.Params("id"))
	entries, err := GetCreditLedger(uint(companyID), 50)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(entries)
}

func getHealthReport(c *fiber.Ctx) error {
	companyID, _ := strconv.Atoi(c.Params("id"))
	month := c.Query("month", time.Now().Format("2006-01"))
	report, err := GetHealthReport(uint(companyID), month)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(report)
}

func listInvoices(c *fiber.Ctx) error {
	companyID, _ := strconv.Atoi(c.Params("id"))
	invoices, err := GetCompanyInvoices(uint(companyID))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(invoices)
}

func generateInvoice(c *fiber.Ctx) error {
	companyID, _ := strconv.Atoi(c.Params("id"))
	var req struct {
		Month string `json:"month"` // "2026-03"
	}
	if err := c.BodyParser(&req); err != nil || req.Month == "" {
		req.Month = time.Now().Format("2006-01")
	}
	inv, err := GenerateMonthlyInvoice(uint(companyID), req.Month)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(inv)
}

func getMyEmployeeProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	emp, err := GetEmployeeByUserID(userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not linked to any company"})
	}
	return c.JSON(emp)
}

func getMyStatus(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	emp, err := GetEmployeeByUserID(userID)
	if err != nil {
		return c.JSON(fiber.Map{"is_corporate": false})
	}
	company, _ := GetCompanyByID(emp.CompanyID)
	return c.JSON(fiber.Map{
		"is_corporate":      true,
		"corporate_name":    company.Name,
		"employee_id":       emp.EmployeeID,
		"credits_available": company.WalletBalance,
	})
}

func getMyCompanyEmployees(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	emp, err := GetEmployeeByUserID(userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not linked to any company"})
	}
	employees, err := GetEmployees(emp.CompanyID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(employees)
}

func getMyCompanyDashboard(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uint)
	emp, err := GetEmployeeByUserID(userID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "not linked to any company"})
	}
	report, err := GetHealthReport(emp.CompanyID, time.Now().Format("2006-01"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(report)
}

func bulkUploadEmployees(c *fiber.Ctx) error {
	companyID, _ := strconv.Atoi(c.Params("id"))
	
	var req struct {
		Employees []EmployeeInput `json:"employees"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	created, err := BulkUploadEmployees(uint(companyID), req.Employees)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{
		"message": "Employees uploaded successfully",
		"count":   len(created),
	})
}
