// Package tds provides TDS (Tax Deducted at Source) computation under the
// Indian Income Tax Act for Dr.Ayya's payroll and professional fee payments.
package tds

import (
	"fmt"
	"time"
)

// ── Section Codes ─────────────────────────────────────────────────────────────

const (
	Section194J = "194J" // Professional / technical services (10%)
	Section194C = "194C" // Contractor payments (1% individual, 2% company)
	Section192  = "192"  // Salary (slab-based)
)

// ── Salary TDS (Section 192) ──────────────────────────────────────────────────

// FYTaxSlab defines an income-tax slab for a financial year.
type FYTaxSlab struct {
	UpTo    int64   // annual income upper limit in paise; 0 = unlimited
	Rate    float64 // marginal rate
	Fixed   int64   // fixed tax on income below this slab, in paise
}

// newRegimeFY2526 reflects the New Tax Regime slabs for FY 2025-26.
var newRegimeFY2526 = []FYTaxSlab{
	{UpTo: 300000_00, Rate: 0.00, Fixed: 0},
	{UpTo: 700000_00, Rate: 0.05, Fixed: 0},
	{UpTo: 1000000_00, Rate: 0.10, Fixed: 20000_00},
	{UpTo: 1200000_00, Rate: 0.15, Fixed: 50000_00},
	{UpTo: 1500000_00, Rate: 0.20, Fixed: 80000_00},
	{UpTo: 0, Rate: 0.30, Fixed: 140000_00},
}

// ComputeAnnualTax calculates income tax (new regime) on annual taxable income (paise).
func ComputeAnnualTax(annualIncomePaise int64) int64 {
	var prevLimit int64
	for _, slab := range newRegimeFY2526 {
		limit := slab.UpTo
		if limit == 0 {
			// Top slab — no cap
			excess := annualIncomePaise - prevLimit
			return slab.Fixed + int64(float64(excess)*slab.Rate)
		}
		if annualIncomePaise <= limit {
			excess := annualIncomePaise - prevLimit
			return slab.Fixed + int64(float64(excess)*slab.Rate)
		}
		prevLimit = limit
	}
	return 0
}

// MonthlyTDS returns the monthly TDS amount to deduct from salary.
func MonthlyTDS(annualIncomePaise int64) int64 {
	annual := ComputeAnnualTax(annualIncomePaise)
	return annual / 12
}

// ── Professional Fee TDS (Section 194J) ───────────────────────────────────────

// TDSOnProfessionalFee computes TDS at 10% on professional/technical fees.
// thresholdPaise: TDS is deducted only when aggregate payment in the year exceeds ₹30,000.
func TDSOnProfessionalFee(amountPaise int64) int64 {
	const threshold int64 = 3000000 // ₹30,000 in paise
	if amountPaise < threshold {
		return 0
	}
	return int64(float64(amountPaise) * 0.10)
}

// ── Form 16A (TDS Certificate for non-salary) ─────────────────────────────────

// Form16AEntry is one deduction entry in Form 16A.
type Form16AEntry struct {
	DateOfPayment  time.Time `json:"date_of_payment"`
	AmountPaise    int64     `json:"amount_paise"`
	TDSPaise       int64     `json:"tds_paise"`
	Section        string    `json:"section"`
}

// Form16A is the TDS certificate issued to a vendor/consultant for non-salary TDS.
type Form16A struct {
	DeductorName   string         `json:"deductor_name"`
	DeductorTAN    string         `json:"deductor_tan"`
	DeductorGSTIN  string         `json:"deductor_gstin,omitempty"`
	DeducteeName   string         `json:"deductee_name"`
	DeducteePAN    string         `json:"deductee_pan"`
	FinancialYear  string         `json:"financial_year"` // "2025-26"
	Quarter        string         `json:"quarter"`         // "Q1" … "Q4"
	Entries        []Form16AEntry `json:"entries"`
	TotalPaid      int64          `json:"total_paid_paise"`
	TotalTDS       int64          `json:"total_tds_paise"`
	GeneratedAt    time.Time      `json:"generated_at"`
}

// GenerateForm16A builds a Form 16A for a deductee.
func GenerateForm16A(deductorName, tan, gstin, deducteeName, pan, fy, quarter string, entries []Form16AEntry) *Form16A {
	var totalPaid, totalTDS int64
	for _, e := range entries {
		totalPaid += e.AmountPaise
		totalTDS += e.TDSPaise
	}
	return &Form16A{
		DeductorName:  deductorName,
		DeductorTAN:   tan,
		DeductorGSTIN: gstin,
		DeducteeName:  deducteeName,
		DeducteePAN:   pan,
		FinancialYear: fy,
		Quarter:       quarter,
		Entries:       entries,
		TotalPaid:     totalPaid,
		TotalTDS:      totalTDS,
		GeneratedAt:   time.Now(),
	}
}

// ── Form 26Q (Quarterly TDS Return for non-salary) ────────────────────────────

// Form26QRow is one deductee row in the quarterly return.
type Form26QRow struct {
	DeducteePAN    string    `json:"deductee_pan"`
	DeducteeName   string    `json:"deductee_name"`
	Section        string    `json:"section"`
	PaymentDate    time.Time `json:"payment_date"`
	AmountPaise    int64     `json:"amount_paise"`
	TDSPaise       int64     `json:"tds_paise"`
	SurchargeAndCess int64   `json:"surcharge_cess_paise"`
	TotalTDS       int64     `json:"total_tds_paise"`
}

// Form26Q is the complete quarterly TDS return structure.
type Form26Q struct {
	DeductorTAN   string       `json:"deductor_tan"`
	DeductorName  string       `json:"deductor_name"`
	FinancialYear string       `json:"financial_year"`
	Quarter       string       `json:"quarter"`
	Rows          []Form26QRow `json:"rows"`
	TotalAmount   int64        `json:"total_amount_paise"`
	TotalTDS      int64        `json:"total_tds_paise"`
	GeneratedAt   time.Time    `json:"generated_at"`
}

// GenerateForm26Q builds the quarterly non-salary TDS return.
func GenerateForm26Q(tan, deductorName, fy, quarter string, rows []Form26QRow) *Form26Q {
	var totalAmt, totalTDS int64
	for _, r := range rows {
		totalAmt += r.AmountPaise
		totalTDS += r.TDSPaise
	}
	return &Form26Q{
		DeductorTAN:   tan,
		DeductorName:  deductorName,
		FinancialYear: fy,
		Quarter:       quarter,
		Rows:          rows,
		TotalAmount:   totalAmt,
		TotalTDS:      totalTDS,
		GeneratedAt:   time.Now(),
	}
}

// FYFromDate returns the financial year string (e.g. "2025-26") for a given date.
func FYFromDate(t time.Time) string {
	year := t.Year()
	if t.Month() < time.April {
		return fmt.Sprintf("%d-%02d", year-1, year%100)
	}
	return fmt.Sprintf("%d-%02d", year, (year+1)%100)
}

// QuarterFromDate returns the TDS quarter ("Q1"–"Q4") for a given date.
func QuarterFromDate(t time.Time) string {
	switch {
	case t.Month() >= time.April && t.Month() <= time.June:
		return "Q1"
	case t.Month() >= time.July && t.Month() <= time.September:
		return "Q2"
	case t.Month() >= time.October && t.Month() <= time.December:
		return "Q3"
	default:
		return "Q4"
	}
}
