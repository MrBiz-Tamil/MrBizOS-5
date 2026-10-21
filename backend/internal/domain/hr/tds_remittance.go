package hr

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// TDSRemittance records a deposit of withheld TDS to the government. We *compute*
// TDS withheld per consultation, but until it is remitted (and a challan issued)
// the books carry an open liability. One record per challan.
//
// AmountPaise is paise.
type TDSRemittance struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	FinancialYear string    `gorm:"index" json:"financial_year"` // "2026-27"
	Quarter       string    `gorm:"index" json:"quarter"`        // Q1 | Q2 | Q3 | Q4
	ChallanNumber string    `json:"challan_number"`
	AmountPaise   int64     `json:"amount_paise"`
	Section       string    `gorm:"default:'194J'" json:"section"` // 194J = professional fees
	RemittedOn    time.Time `json:"remitted_on"`
	Notes         string    `gorm:"type:text" json:"notes"`
}

func (TDSRemittance) TableName() string { return "tds_remittances" }

// quarterDateRange returns [from, to) for an Indian financial-year quarter.
// FY "2026-27" Q1 = 2026-04-01 .. 2026-07-01, etc. Q4 spans into the next year.
func quarterDateRange(fy, quarter string) (from, to string, err error) {
	var startYear int
	if _, e := fmt.Sscanf(fy, "%d", &startYear); e != nil || startYear < 2000 {
		return "", "", fmt.Errorf("invalid financial_year (expected e.g. 2026-27)")
	}
	switch quarter {
	case "Q1":
		return fmt.Sprintf("%d-04-01", startYear), fmt.Sprintf("%d-07-01", startYear), nil
	case "Q2":
		return fmt.Sprintf("%d-07-01", startYear), fmt.Sprintf("%d-10-01", startYear), nil
	case "Q3":
		return fmt.Sprintf("%d-10-01", startYear), fmt.Sprintf("%d-01-01", startYear+1), nil
	case "Q4":
		return fmt.Sprintf("%d-01-01", startYear+1), fmt.Sprintf("%d-04-01", startYear+1), nil
	default:
		return "", "", fmt.Errorf("invalid quarter (expected Q1..Q4)")
	}
}

// computeTDSWithheld totals the exact TDS withheld across all completed panel-doctor
// consultations in [from, to), using the canonical split.
func computeTDSWithheld(from, to string) (total int64, consults int) {
	type bucket struct {
		ConsultationFee int64
		Cnt             int
	}
	var buckets []bucket
	database.DB.Table("appointments").
		Select("consultation_fee, COUNT(*) AS cnt").
		Where("status = ? AND doctor_id IS NOT NULL AND created_at >= ? AND created_at < ?", "completed", from, to).
		Group("consultation_fee").
		Scan(&buckets)
	for _, b := range buckets {
		total += finance.ComputeConsultationSplit(b.ConsultationFee).TDS * int64(b.Cnt)
		consults += b.Cnt
	}
	return total, consults
}

// recordTDSRemittance logs a challan against a financial-year quarter.
// POST /hr/payroll/tds/remittances
func recordTDSRemittance(c *fiber.Ctx) error {
	var req struct {
		FinancialYear string `json:"financial_year"`
		Quarter       string `json:"quarter"`
		ChallanNumber string `json:"challan_number"`
		AmountPaise   int64  `json:"amount_paise"`
		Section       string `json:"section"`
		RemittedOn    string `json:"remitted_on"` // YYYY-MM-DD
		Notes         string `json:"notes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}
	if req.FinancialYear == "" || req.Quarter == "" || req.ChallanNumber == "" || req.AmountPaise <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "financial_year, quarter, challan_number and amount_paise are required"})
	}
	if _, _, err := quarterDateRange(req.FinancialYear, req.Quarter); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	remittedOn := time.Now()
	if req.RemittedOn != "" {
		if t, err := time.Parse("2006-01-02", req.RemittedOn); err == nil {
			remittedOn = t
		}
	}
	if req.Section == "" {
		req.Section = "194J"
	}
	rec := TDSRemittance{
		FinancialYear: req.FinancialYear,
		Quarter:       req.Quarter,
		ChallanNumber: req.ChallanNumber,
		AmountPaise:   req.AmountPaise,
		Section:       req.Section,
		RemittedOn:    remittedOn,
		Notes:         req.Notes,
	}
	if err := database.DB.Create(&rec).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(201).JSON(rec)
}

// listTDSRemittances returns recorded challans plus a withheld-vs-remitted
// reconciliation for the requested financial-year quarter.
// GET /hr/payroll/tds/remittances?financial_year=2026-27&quarter=Q1
func listTDSRemittances(c *fiber.Ctx) error {
	fy := c.Query("financial_year")
	quarter := c.Query("quarter")
	if fy == "" || quarter == "" {
		return c.Status(400).JSON(fiber.Map{"error": "financial_year and quarter are required"})
	}
	from, to, err := quarterDateRange(fy, quarter)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	var records []TDSRemittance
	database.DB.Where("financial_year = ? AND quarter = ?", fy, quarter).Order("remitted_on DESC").Find(&records)

	var remitted int64
	for _, r := range records {
		remitted += r.AmountPaise
	}
	withheld, consults := computeTDSWithheld(from, to)

	return c.JSON(fiber.Map{
		"financial_year":   fy,
		"quarter":          quarter,
		"period_from":      from,
		"period_to":        to,
		"consult_count":    consults,
		"withheld_paise":   withheld,
		"remitted_paise":   remitted,
		"outstanding_paise": withheld - remitted,
		"records":          records,
	})
}
