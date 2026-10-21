// Package gst provides GST (Goods and Services Tax) computation, e-invoice
// generation, and GSTR report building for Dr.Ayya.
package gst

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// ── Tax Rates ─────────────────────────────────────────────────────────────────

const (
	RateExempt   float64 = 0.00  // Healthcare services (exempt)
	RateReduced  float64 = 0.05  // Some medicines
	RateStandard float64 = 0.12  // Diagnostics / lab
	RateFull     float64 = 0.18  // Software/SaaS, subscription plans
)

// ServiceType maps a service category to its GST rate.
type ServiceType string

const (
	ServiceConsultation ServiceType = "consultation" // Exempt
	ServiceMedicine     ServiceType = "medicine"     // 5 or 12%
	ServiceLab          ServiceType = "lab"          // 5%
	ServiceSubscription ServiceType = "subscription" // 18%
	ServiceSoftware     ServiceType = "software"     // 18%
)

// GSTRate returns the applicable GST rate for a given service type.
func GSTRate(svc ServiceType) float64 {
	switch svc {
	case ServiceConsultation:
		return RateExempt
	case ServiceMedicine:
		return RateReduced
	case ServiceLab:
		return RateReduced
	case ServiceSubscription, ServiceSoftware:
		return RateFull
	default:
		return RateStandard
	}
}

// ── Tax Amounts ───────────────────────────────────────────────────────────────

// TaxBreakdown holds the computed GST split.
type TaxBreakdown struct {
	BaseAmountPaise int64   `json:"base_amount_paise"`
	Rate            float64 `json:"rate"`
	CGSTPaise       int64   `json:"cgst_paise"` // half of GST (intra-state)
	SGSTPaise       int64   `json:"sgst_paise"` // half of GST (intra-state)
	IGSTPaise       int64   `json:"igst_paise"` // full (inter-state)
	TotalGSTPaise   int64   `json:"total_gst_paise"`
	TotalPaise      int64   `json:"total_paise"` // base + GST
	IsInterState    bool    `json:"is_inter_state"`
}

// ComputeTax computes GST on a base amount.
// interState = true for cross-state (IGST only), false for intra-state (CGST+SGST).
func ComputeTax(baseAmountPaise int64, rate float64, interState bool) TaxBreakdown {
	totalGST := int64(float64(baseAmountPaise) * rate)
	tb := TaxBreakdown{
		BaseAmountPaise: baseAmountPaise,
		Rate:            rate,
		TotalGSTPaise:   totalGST,
		TotalPaise:      baseAmountPaise + totalGST,
		IsInterState:    interState,
	}
	if interState {
		tb.IGSTPaise = totalGST
	} else {
		tb.CGSTPaise = totalGST / 2
		tb.SGSTPaise = totalGST - tb.CGSTPaise // handle odd paise
	}
	return tb
}

// ── E-Invoice (IRN) ───────────────────────────────────────────────────────────

// EInvoiceItem represents a single line item on the invoice.
type EInvoiceItem struct {
	Description  string      `json:"description"`
	HSNCode      string      `json:"hsn_code"` // HSN/SAC code
	ServiceType  ServiceType `json:"service_type"`
	Quantity     float64     `json:"quantity"`
	UnitPaise    int64       `json:"unit_paise"`
	AmountPaise  int64       `json:"amount_paise"`
	Tax          TaxBreakdown `json:"tax"`
}

// EInvoice is the full e-invoice document.
type EInvoice struct {
	IRN          string         `json:"irn"`            // Invoice Reference Number (stub)
	InvoiceNo    string         `json:"invoice_no"`
	InvoiceDate  time.Time      `json:"invoice_date"`
	SellerGSTIN  string         `json:"seller_gstin"`
	BuyerGSTIN   string         `json:"buyer_gstin,omitempty"`
	BuyerName    string         `json:"buyer_name"`
	BuyerAddress string         `json:"buyer_address"`
	Items        []EInvoiceItem `json:"items"`
	SubTotal     int64          `json:"sub_total_paise"`
	TotalCGST    int64          `json:"total_cgst_paise"`
	TotalSGST    int64          `json:"total_sgst_paise"`
	TotalIGST    int64          `json:"total_igst_paise"`
	TotalTax     int64          `json:"total_tax_paise"`
	GrandTotal   int64          `json:"grand_total_paise"`
	QRCode       string         `json:"qr_code,omitempty"` // base64 stub
	GeneratedAt  time.Time      `json:"generated_at"`
}

// GenerateIRN generates a stub Invoice Reference Number.
// In production this should call NIC IRP API.
func GenerateIRN() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "IRN" + hex.EncodeToString(b)
}

// GenerateEInvoice builds the complete e-invoice document.
func GenerateEInvoice(invoiceNo, sellerGSTIN, buyerName, buyerAddress, buyerGSTIN string, items []EInvoiceItem, interState bool) (*EInvoice, error) {
	if sellerGSTIN == "" {
		return nil, fmt.Errorf("seller GSTIN is required")
	}

	// Compute taxes per item
	var subTotal, totalCGST, totalSGST, totalIGST int64
	for i, item := range items {
		rate := GSTRate(item.ServiceType)
		tax := ComputeTax(item.AmountPaise, rate, interState)
		items[i].Tax = tax
		subTotal += item.AmountPaise
		totalCGST += tax.CGSTPaise
		totalSGST += tax.SGSTPaise
		totalIGST += tax.IGSTPaise
	}

	totalTax := totalCGST + totalSGST + totalIGST
	inv := &EInvoice{
		IRN:          GenerateIRN(),
		InvoiceNo:    invoiceNo,
		InvoiceDate:  time.Now(),
		SellerGSTIN:  sellerGSTIN,
		BuyerGSTIN:   buyerGSTIN,
		BuyerName:    buyerName,
		BuyerAddress: buyerAddress,
		Items:        items,
		SubTotal:     subTotal,
		TotalCGST:    totalCGST,
		TotalSGST:    totalSGST,
		TotalIGST:    totalIGST,
		TotalTax:     totalTax,
		GrandTotal:   subTotal + totalTax,
		GeneratedAt:  time.Now(),
	}
	return inv, nil
}

// ── GSTR-1 (Outward Supplies) ─────────────────────────────────────────────────

// GSTR1Entry is one taxable outward supply row.
type GSTR1Entry struct {
	InvoiceNo   string      `json:"invoice_no"`
	InvoiceDate time.Time   `json:"invoice_date"`
	BuyerGSTIN  string      `json:"buyer_gstin,omitempty"`
	BuyerName   string      `json:"buyer_name"`
	ServiceType ServiceType `json:"service_type"`
	Taxable     int64       `json:"taxable_paise"`
	CGST        int64       `json:"cgst_paise"`
	SGST        int64       `json:"sgst_paise"`
	IGST        int64       `json:"igst_paise"`
	TotalTax    int64       `json:"total_tax_paise"`
	GrandTotal  int64       `json:"grand_total_paise"`
}

// GSTR1Report is the B2B + B2C summary for filing.
type GSTR1Report struct {
	Period     string       `json:"period"` // "2026-03"
	GSTIN      string       `json:"gstin"`
	B2B        []GSTR1Entry `json:"b2b"`         // registered buyers
	B2C        []GSTR1Entry `json:"b2c"`         // unregistered buyers
	Exempt     []GSTR1Entry `json:"exempt"`      // exempt supplies
	TotalTaxable int64      `json:"total_taxable_paise"`
	TotalTax   int64        `json:"total_tax_paise"`
	GeneratedAt time.Time   `json:"generated_at"`
}

// BuildGSTR1 aggregates EInvoice entries into a GSTR-1 report.
func BuildGSTR1(period, gstin string, invoices []EInvoice) *GSTR1Report {
	report := &GSTR1Report{
		Period:      period,
		GSTIN:       gstin,
		GeneratedAt: time.Now(),
	}
	for _, inv := range invoices {
		for _, item := range inv.Items {
			entry := GSTR1Entry{
				InvoiceNo:   inv.InvoiceNo,
				InvoiceDate: inv.InvoiceDate,
				BuyerGSTIN:  inv.BuyerGSTIN,
				BuyerName:   inv.BuyerName,
				ServiceType: item.ServiceType,
				Taxable:     item.AmountPaise,
				CGST:        item.Tax.CGSTPaise,
				SGST:        item.Tax.SGSTPaise,
				IGST:        item.Tax.IGSTPaise,
				TotalTax:    item.Tax.TotalGSTPaise,
				GrandTotal:  item.Tax.TotalPaise,
			}
			if item.Tax.Rate == 0 {
				report.Exempt = append(report.Exempt, entry)
			} else if inv.BuyerGSTIN != "" {
				report.B2B = append(report.B2B, entry)
			} else {
				report.B2C = append(report.B2C, entry)
			}
			report.TotalTaxable += entry.Taxable
			report.TotalTax += entry.TotalTax
		}
	}
	return report
}

// ── GSTR-3B (Summary Return) ──────────────────────────────────────────────────

// GSTR3BReport is the monthly summary return.
type GSTR3BReport struct {
	Period            string    `json:"period"`
	GSTIN             string    `json:"gstin"`
	OutwardTaxable    int64     `json:"outward_taxable_paise"`
	OutwardExempt     int64     `json:"outward_exempt_paise"`
	OutwardNilRated   int64     `json:"outward_nil_rated_paise"`
	TotalCGST         int64     `json:"total_cgst_paise"`
	TotalSGST         int64     `json:"total_sgst_paise"`
	TotalIGST         int64     `json:"total_igst_paise"`
	TotalTaxLiability int64     `json:"total_tax_liability_paise"`
	InputTaxCredit    int64     `json:"input_tax_credit_paise"`
	TaxPayable        int64     `json:"tax_payable_paise"`
	GeneratedAt       time.Time `json:"generated_at"`
}

// BuildGSTR3B derives the GSTR-3B from a GSTR-1 report and input tax credit.
func BuildGSTR3B(gstr1 *GSTR1Report, inputTaxCreditPaise int64) *GSTR3BReport {
	var exemptTotal int64
	for _, e := range gstr1.Exempt {
		exemptTotal += e.Taxable
	}
	taxLiability := gstr1.TotalTax
	payable := taxLiability - inputTaxCreditPaise
	if payable < 0 {
		payable = 0
	}
	return &GSTR3BReport{
		Period:            gstr1.Period,
		GSTIN:             gstr1.GSTIN,
		OutwardTaxable:    gstr1.TotalTaxable - exemptTotal,
		OutwardExempt:     exemptTotal,
		TotalTaxLiability: taxLiability,
		InputTaxCredit:    inputTaxCreditPaise,
		TaxPayable:        payable,
		GeneratedAt:       time.Now(),
	}
}
