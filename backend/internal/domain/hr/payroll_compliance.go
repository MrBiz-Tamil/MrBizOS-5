package hr

import (
	"fmt"
	"time"

	"drayya-core/internal/database"
)

// ── Models ────────────────────────────────────────────────────────────────────

// PayrollCompliance stores computed statutory deductions for a staff member's monthly payroll.
type PayrollCompliance struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	Month          string    `json:"month"` // "2026-03"
	GrossSalary    int64     `json:"gross_salary"`    // paise
	BasicSalary    int64     `json:"basic_salary"`    // paise (used for PF base)
	PFEmployee     int64     `json:"pf_employee"`     // 12% of basic, paise
	PFEmployer     int64     `json:"pf_employer"`     // 12% of basic, paise
	ESIEmployee    int64     `json:"esi_employee"`    // 0.75% of gross (if eligible)
	ESIEmployer    int64     `json:"esi_employer"`    // 3.25% of gross (if eligible)
	ProfTax        int64     `json:"prof_tax"`        // state-specific slab, paise
	IncomeTaxTDS   int64     `json:"income_tax_tds"`  // monthly TDS, paise
	NetSalary      int64     `json:"net_salary"`      // paise
	ESIEligible    bool      `json:"esi_eligible"`    // gross <= 21000/month
	PFEligible     bool      `json:"pf_eligible"`
}

// Payslip is the employee-facing summary for a given month.
type Payslip struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	Month          string    `json:"month"`
	EmployeeName   string    `json:"employee_name"`
	Designation    string    `json:"designation"`
	PAN            string    `json:"pan,omitempty"`
	UAN            string    `json:"uan,omitempty"` // Universal Account Number (PF)
	ESINumber      string    `json:"esi_number,omitempty"`
	GrossSalary    int64     `json:"gross_salary"`
	BasicSalary    int64     `json:"basic_salary"`
	HRA            int64     `json:"hra"`
	SpecialAllowance int64   `json:"special_allowance"`
	PFEmployee     int64     `json:"pf_employee"`
	ESIEmployee    int64     `json:"esi_employee"`
	ProfTax        int64     `json:"prof_tax"`
	IncomeTaxTDS   int64     `json:"income_tax_tds"`
	TotalDeductions int64    `json:"total_deductions"`
	NetSalary      int64     `json:"net_salary"`
	PaidDays       int       `json:"paid_days"`
	LopDays        float64   `json:"lop_days"` // loss-of-pay
}

// ── Professional Tax Slabs (Karnataka — adjust per state) ─────────────────────

type profTaxSlab struct {
	MaxMonthlyGross int64 // paise
	Tax             int64 // paise per month
}

// ptSlabs for Karnataka (effective 2024-25)
var ptSlabs = []profTaxSlab{
	{MaxMonthlyGross: 1500000, Tax: 0},       // <= ₹15,000
	{MaxMonthlyGross: 2000000, Tax: 15000},   // ₹15,001–₹20,000 → ₹150
	{MaxMonthlyGross: 1<<62, Tax: 20000},     // > ₹20,000 → ₹200
}

func calcProfTax(grossPaise int64) int64 {
	for _, slab := range ptSlabs {
		if grossPaise <= slab.MaxMonthlyGross {
			return slab.Tax
		}
	}
	return 20000
}

// ── Computation ───────────────────────────────────────────────────────────────

// ComputePayrollCompliance calculates all statutory deductions for a given employee month.
// grossPaise = total gross salary for the month in paise.
// basicPercent = fraction of gross treated as "basic" (default 0.50).
func ComputePayrollCompliance(userID uint, month string, grossPaise int64, basicPercent float64, monthlyTDSPaise int64) *PayrollCompliance {
	if basicPercent <= 0 || basicPercent > 1 {
		basicPercent = 0.50
	}

	basic := int64(float64(grossPaise) * basicPercent)

	// PF cap: PF on max ₹15,000 basic (₹1,500,000 paise)
	pfBase := basic
	const pfCap int64 = 1500000 // ₹15,000 in paise
	if pfBase > pfCap {
		pfBase = pfCap
	}

	pfEmployee := int64(float64(pfBase) * 0.12)
	pfEmployer := int64(float64(pfBase) * 0.12)

	// ESI: eligible if gross ≤ ₹21,000 (₹2,100,000 paise)
	const esiThreshold int64 = 2100000
	esiEmployee, esiEmployer := int64(0), int64(0)
	esiEligible := grossPaise <= esiThreshold
	if esiEligible {
		esiEmployee = int64(float64(grossPaise) * 0.0075)
		esiEmployer = int64(float64(grossPaise) * 0.0325)
	}

	pt := calcProfTax(grossPaise)
	totalDeductions := pfEmployee + esiEmployee + pt + monthlyTDSPaise
	net := grossPaise - totalDeductions

	return &PayrollCompliance{
		UserID:       userID,
		Month:        month,
		GrossSalary:  grossPaise,
		BasicSalary:  basic,
		PFEmployee:   pfEmployee,
		PFEmployer:   pfEmployer,
		ESIEmployee:  esiEmployee,
		ESIEmployer:  esiEmployer,
		ProfTax:      pt,
		IncomeTaxTDS: monthlyTDSPaise,
		NetSalary:    net,
		ESIEligible:  esiEligible,
		PFEligible:   true,
	}
}

// SavePayrollCompliance persists the computed record (upsert by user+month).
func SavePayrollCompliance(pc *PayrollCompliance) error {
	var existing PayrollCompliance
	err := database.DB.Where("user_id = ? AND month = ?", pc.UserID, pc.Month).First(&existing).Error
	if err != nil {
		return database.DB.Create(pc).Error
	}
	pc.ID = existing.ID
	return database.DB.Save(pc).Error
}

// GetPayrollCompliance fetches the compliance record for a user-month.
func GetPayrollComplianceByUserMonth(userID uint, month string) (*PayrollCompliance, error) {
	var pc PayrollCompliance
	err := database.DB.Where("user_id = ? AND month = ?", userID, month).First(&pc).Error
	return &pc, err
}

// GeneratePayslip creates a Payslip from a PayrollCompliance record and staff profile info.
func GeneratePayslip(pc *PayrollCompliance, name, designation, pan, uan, esiNumber string, paidDays int, lopDays float64) (*Payslip, error) {
	// Standard allowance split: 50% basic, 20% HRA, 30% special
	hra := int64(float64(pc.BasicSalary) * 0.40)
	special := pc.GrossSalary - pc.BasicSalary - hra
	totalDed := pc.PFEmployee + pc.ESIEmployee + pc.ProfTax + pc.IncomeTaxTDS

	ps := &Payslip{
		UserID:           pc.UserID,
		Month:            pc.Month,
		EmployeeName:     name,
		Designation:      designation,
		PAN:              pan,
		UAN:              uan,
		ESINumber:        esiNumber,
		GrossSalary:      pc.GrossSalary,
		BasicSalary:      pc.BasicSalary,
		HRA:              hra,
		SpecialAllowance: special,
		PFEmployee:       pc.PFEmployee,
		ESIEmployee:      pc.ESIEmployee,
		ProfTax:          pc.ProfTax,
		IncomeTaxTDS:     pc.IncomeTaxTDS,
		TotalDeductions:  totalDed,
		NetSalary:        pc.NetSalary,
		PaidDays:         paidDays,
		LopDays:          lopDays,
	}
	if err := database.DB.Create(ps).Error; err != nil {
		return nil, err
	}
	return ps, nil
}

// GetPayslip retrieves a payslip for a user and month.
func GetPayslip(userID uint, month string) (*Payslip, error) {
	var ps Payslip
	err := database.DB.Where("user_id = ? AND month = ?", userID, month).First(&ps).Error
	return &ps, err
}

// GetAllPayslips returns all payslips for a user.
func GetAllPayslips(userID uint) ([]Payslip, error) {
	var slips []Payslip
	err := database.DB.Where("user_id = ?", userID).Order("month DESC").Find(&slips).Error
	return slips, err
}

// GenerateEPFStatement produces EPF contribution data for a month across all staff.
func GenerateEPFStatement(month string) ([]map[string]interface{}, error) {
	var records []PayrollCompliance
	if err := database.DB.Where("month = ? AND pf_eligible = true", month).Find(&records).Error; err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, r := range records {
		result = append(result, map[string]interface{}{
			"user_id":     r.UserID,
			"month":       r.Month,
			"basic_paise": r.BasicSalary,
			"ee_pf":       r.PFEmployee,
			"er_pf":       r.PFEmployer,
			"total_pf":    r.PFEmployee + r.PFEmployer,
		})
	}
	return result, nil
}

// GenerateESIStatement produces ESI contribution data for a month across all eligible staff.
func GenerateESIStatement(month string) ([]map[string]interface{}, error) {
	var records []PayrollCompliance
	if err := database.DB.Where("month = ? AND esi_eligible = true", month).Find(&records).Error; err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, r := range records {
		result = append(result, map[string]interface{}{
			"user_id":      r.UserID,
			"month":        r.Month,
			"gross_paise":  r.GrossSalary,
			"ee_esi":       r.ESIEmployee,
			"er_esi":       r.ESIEmployer,
			"total_esi":    r.ESIEmployee + r.ESIEmployer,
		})
	}
	return result, nil
}

// PayrollSummary returns aggregate payroll stats for a month.
func GetPayrollSummary(month string) (map[string]interface{}, error) {
	var records []PayrollCompliance
	if err := database.DB.Where("month = ?", month).Find(&records).Error; err != nil {
		return nil, err
	}
	var totalGross, totalPFEE, totalPFER, totalESIEE, totalESIER, totalPT, totalTDS, totalNet int64
	for _, r := range records {
		totalGross += r.GrossSalary
		totalPFEE += r.PFEmployee
		totalPFER += r.PFEmployer
		totalESIEE += r.ESIEmployee
		totalESIER += r.ESIEmployer
		totalPT += r.ProfTax
		totalTDS += r.IncomeTaxTDS
		totalNet += r.NetSalary
	}
	return map[string]interface{}{
		"month":              month,
		"headcount":          len(records),
		"total_gross_paise":  totalGross,
		"total_pf_ee":        totalPFEE,
		"total_pf_er":        totalPFER,
		"total_esi_ee":       totalESIEE,
		"total_esi_er":       totalESIER,
		"total_prof_tax":     totalPT,
		"total_tds":          totalTDS,
		"total_net_paise":    totalNet,
		"total_employer_cost": totalGross + totalPFER + totalESIER,
		"generated_at":       fmt.Sprintf("%s", time.Now().Format(time.RFC3339)),
	}, nil
}
