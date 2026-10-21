package hr

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"

	"github.com/gofiber/fiber/v2"
)

// Form26QLine is one deductee row in a quarterly 26Q TDS return — the freelance
// counterpart to the EPF/ESI statements produced for salaried staff.
type Form26QLine struct {
	DoctorUserID uint   `json:"doctor_user_id"`
	DoctorName   string `json:"doctor_name"`
	PAN          string `json:"pan"`
	PANVerified  bool   `json:"pan_verified"`
	ConsultCount int    `json:"consult_count"`
	GrossPaise   int64  `json:"gross_paise"`        // amount paid/credited
	TDSPaise     int64  `json:"tds_paise"`          // tax deducted
	Section      string `json:"section"`            // 194J
}

// form26QExport produces a filing-ready 26Q return for a financial-year quarter:
// one line per panel doctor with their gross credited and TDS deducted, plus a
// PAN-missing flag (deductees without a valid PAN attract a higher TDS rate and
// must be flagged before filing).
//
// GET /hr/payroll/tds/26q?financial_year=2026-27&quarter=Q1
//
// @Summary Form 26Q Export
// @Description Quarterly TDS return lines for freelance (panel) doctors.
// @Tags HR
// @Produce json
// @Security ApiKeyAuth
// @Param financial_year query string true "Financial year, e.g. 2026-27"
// @Param quarter query string true "Q1 | Q2 | Q3 | Q4"
// @Success 200 {object} response.StandardResponse "26Q export"
// @Router /api/v1/hr/payroll/tds/26q [get]
// computeForm26Q builds the 26Q deductee lines + totals for a FY quarter. Shared
// by the JSON endpoint and the CSV export.
func computeForm26Q(fy, quarter string) (lines []Form26QLine, totalGross, totalTDS int64, missingPAN int, from, to string, err error) {
	from, to, err = quarterDateRange(fy, quarter)
	if err != nil {
		return nil, 0, 0, 0, "", "", err
	}

	db := database.DB
	type row struct {
		DoctorID        uint
		ConsultationFee int64
		Cnt             int
	}
	var rows []row
	db.Table("appointments").
		Select("doctor_id, consultation_fee, COUNT(*) AS cnt").
		Where("status = ? AND doctor_id IS NOT NULL AND created_at >= ? AND created_at < ?", "completed", from, to).
		Group("doctor_id, consultation_fee").
		Scan(&rows)

	lineMap := map[uint]*Form26QLine{}
	for _, r := range rows {
		if r.DoctorID == 0 || r.Cnt == 0 {
			continue
		}
		split := finance.ComputeConsultationSplit(r.ConsultationFee)
		l, ok := lineMap[r.DoctorID]
		if !ok {
			l = &Form26QLine{DoctorUserID: r.DoctorID, Section: "194J"}
			lineMap[r.DoctorID] = l
		}
		l.ConsultCount += r.Cnt
		l.GrossPaise += r.ConsultationFee * int64(r.Cnt)
		l.TDSPaise += split.TDS * int64(r.Cnt)
	}

	if len(lineMap) > 0 {
		ids := make([]uint, 0, len(lineMap))
		for id := range lineMap {
			ids = append(ids, id)
		}
		var users []identity.User
		db.Where("id IN ?", ids).Find(&users)
		nameMap := map[uint]string{}
		for _, u := range users {
			nameMap[u.ID] = u.FullName
		}
		type panRow struct {
			UserID      uint
			PAN         string
			PANVerified bool
		}
		var pans []panRow
		db.Table("doctorprofile").Select("user_id, pan, pan_verified").Where("user_id IN ?", ids).Scan(&pans)
		panMap := map[uint]panRow{}
		for _, p := range pans {
			panMap[p.UserID] = p
		}
		for id, l := range lineMap {
			l.DoctorName = nameMap[id]
			l.PAN = panMap[id].PAN
			l.PANVerified = panMap[id].PANVerified
		}
	}

	for _, l := range lineMap {
		lines = append(lines, *l)
		totalGross += l.GrossPaise
		totalTDS += l.TDSPaise
		if l.PAN == "" {
			missingPAN++
		}
	}
	return lines, totalGross, totalTDS, missingPAN, from, to, nil
}

func form26QExport(c *fiber.Ctx) error {
	fy := c.Query("financial_year")
	quarter := c.Query("quarter")
	if fy == "" || quarter == "" {
		return c.Status(400).JSON(fiber.Map{"error": "financial_year and quarter are required"})
	}
	lines, totalGross, totalTDS, missingPAN, from, to, err := computeForm26Q(fy, quarter)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"financial_year":    fy,
		"quarter":           quarter,
		"period_from":       from,
		"period_to":         to,
		"deductee_count":    len(lines),
		"missing_pan_count": missingPAN,
		"total_gross_paise": totalGross,
		"total_tds_paise":   totalTDS,
		"lines":             lines,
	})
}
