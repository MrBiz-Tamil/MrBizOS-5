package gst

import (
	"github.com/gofiber/fiber/v2"
	"drayya-core/internal/middleware"
)

// RegisterGSTRoutes wires GST computation and report endpoints.
func RegisterGSTRoutes(router fiber.Router) {
	g := router.Group("/admin/gst", middleware.Protected(), middleware.RequireRoles("admin", "super_admin", "finance_manager"))

	g.Post("/compute-tax", computeTax)
	g.Post("/e-invoice", generateEInvoiceHandler)
	g.Post("/gstr1", buildGSTR1Handler)
	g.Post("/gstr3b", buildGSTR3BHandler)

	// Additional Dashboard Endpoints
	g.Get("/summary", getGSTSummary)
	g.Get("/returns", getGSTReturns)
	g.Get("/einvoices", getEInvoices)
	g.Post("/compute", computeGSTHandler)
	g.Post("/file", fileGSTReturnHandler)
	g.Post("/einvoice/:id/generate", generateEInvoiceByIdHandler)
}

type ComputeTaxRequest struct {
	BaseAmountPaise int64       `json:"base_amount_paise"`
	ServiceType     ServiceType `json:"service_type"`
	InterState      bool        `json:"inter_state"`
}

func computeTax(c *fiber.Ctx) error {
	var req ComputeTaxRequest
	if err := c.BodyParser(&req); err != nil || req.BaseAmountPaise <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "base_amount_paise is required"})
	}
	rate := GSTRate(req.ServiceType)
	breakdown := ComputeTax(req.BaseAmountPaise, rate, req.InterState)
	return c.JSON(breakdown)
}

type EInvoiceRequest struct {
	InvoiceNo    string         `json:"invoice_no"`
	SellerGSTIN  string         `json:"seller_gstin"`
	BuyerName    string         `json:"buyer_name"`
	BuyerAddress string         `json:"buyer_address"`
	BuyerGSTIN   string         `json:"buyer_gstin,omitempty"`
	InterState   bool           `json:"inter_state"`
	Items        []EInvoiceItem `json:"items"`
}

func generateEInvoiceHandler(c *fiber.Ctx) error {
	var req EInvoiceRequest
	if err := c.BodyParser(&req); err != nil || req.SellerGSTIN == "" {
		return c.Status(400).JSON(fiber.Map{"error": "seller_gstin and items are required"})
	}
	inv, err := GenerateEInvoice(req.InvoiceNo, req.SellerGSTIN, req.BuyerName, req.BuyerAddress, req.BuyerGSTIN, req.Items, req.InterState)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(inv)
}

type GSTR1Request struct {
	Period  string    `json:"period"`
	GSTIN   string    `json:"gstin"`
	Invoices []EInvoice `json:"invoices"`
}

func buildGSTR1Handler(c *fiber.Ctx) error {
	var req GSTR1Request
	if err := c.BodyParser(&req); err != nil || req.GSTIN == "" {
		return c.Status(400).JSON(fiber.Map{"error": "gstin and period are required"})
	}
	report := BuildGSTR1(req.Period, req.GSTIN, req.Invoices)
	return c.JSON(report)
}

type GSTR3BRequest struct {
	InputTaxCreditPaise int64      `json:"input_tax_credit_paise"`
	GSTR1               GSTR1Report `json:"gstr1"`
}

func buildGSTR3BHandler(c *fiber.Ctx) error {
	var req GSTR3BRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	report := BuildGSTR3B(&req.GSTR1, req.InputTaxCreditPaise)
	return c.JSON(report)
}

// Mock Handlers for Dashboard
func getGSTSummary(c *fiber.Ctx) error {
	period := c.Query("period", "2026-03")
	return c.JSON(fiber.Map{
		"period":           period,
		"taxable_turnover": 500000000,
		"total_cgst":       45000000,
		"total_sgst":       45000000,
		"total_igst":       10000000,
		"total_tax":        100000000,
		"itc_claimed":      20000000,
		"net_payable":      80000000,
		"b2b_invoices":     150,
		"b2c_invoices":     3200,
	})
}

func getGSTReturns(c *fiber.Ctx) error {
	return c.JSON([]fiber.Map{
		{
			"id":          1,
			"return_type": "GSTR-1",
			"period":      "2026-03",
			"status":      "draft",
			"due_date":    "2026-04-11T23:59:59Z",
			"tax_payable": 100000000,
		},
		{
			"id":          2,
			"return_type": "GSTR-3B",
			"period":      "2026-03",
			"status":      "draft",
			"due_date":    "2026-04-20T23:59:59Z",
			"tax_payable": 80000000,
		},
	})
}

func getEInvoices(c *fiber.Ctx) error {
	return c.JSON([]fiber.Map{
		{
			"id":             1,
			"invoice_number": "INV-2026-001",
			"customer_name":  "Acme Corp",
			"customer_gstin": "27AADCB2230M1Z2",
			"taxable_amount": 10000000,
			"total_tax":      1800000,
			"total_amount":   11800000,
			"status":         "pending",
			"created_at":     "2026-03-15T10:00:00Z",
		},
		{
			"id":             2,
			"invoice_number": "INV-2026-002",
			"customer_name":  "Global Health",
			"customer_gstin": "29AABCT1332L1Z1",
			"taxable_amount": 20000000,
			"total_tax":      3600000,
			"total_amount":   23600000,
			"irn":            "a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3",
			"status":         "generated",
			"created_at":     "2026-03-16T11:30:00Z",
		},
	})
}

func computeGSTHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true})
}

func fileGSTReturnHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"arn": "AA2703261234567"})
}

func generateEInvoiceByIdHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"irn": "a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t1u2v3w4x5y6z7a8b9c0d1e2f3"})
}

