package hr

import (
	"bytes"
	"drayya-core/internal/database"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// CSV export endpoints for the statutory statements + attendance. Each returns a
// downloadable text/csv attachment with amounts rendered in rupees (paise/100).

func sendCSV(c *fiber.Ctx, filename string, header []string, rows [][]string) error {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(header)
	for _, r := range rows {
		_ = w.Write(r)
	}
	w.Flush()
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return c.Send(buf.Bytes())
}

func paiseStr(v int64) string { return fmt.Sprintf("%.2f", float64(v)/100) }

// GET /hr/payroll/export/epf?month=YYYY-MM
func exportEPFCSV(c *fiber.Ctx) error {
	month := c.Query("month")
	if month == "" {
		return c.Status(400).JSON(fiber.Map{"error": "month required"})
	}
	data, err := GenerateEPFStatement(month)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	rows := make([][]string, 0, len(data))
	for _, d := range data {
		rows = append(rows, []string{
			fmt.Sprintf("%v", d["user_id"]), fmt.Sprintf("%v", d["month"]),
			paiseStr(toInt64(d["basic_paise"])), paiseStr(toInt64(d["ee_pf"])),
			paiseStr(toInt64(d["er_pf"])), paiseStr(toInt64(d["total_pf"])),
		})
	}
	return sendCSV(c, "epf_"+month+".csv",
		[]string{"User ID", "Month", "Basic", "EE PF", "ER PF", "Total PF"}, rows)
}

// GET /hr/payroll/export/esi?month=YYYY-MM
func exportESICSV(c *fiber.Ctx) error {
	month := c.Query("month")
	if month == "" {
		return c.Status(400).JSON(fiber.Map{"error": "month required"})
	}
	data, err := GenerateESIStatement(month)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	rows := make([][]string, 0, len(data))
	for _, d := range data {
		rows = append(rows, []string{
			fmt.Sprintf("%v", d["user_id"]), fmt.Sprintf("%v", d["month"]),
			paiseStr(toInt64(d["gross_paise"])), paiseStr(toInt64(d["ee_esi"])),
			paiseStr(toInt64(d["er_esi"])), paiseStr(toInt64(d["total_esi"])),
		})
	}
	return sendCSV(c, "esi_"+month+".csv",
		[]string{"User ID", "Month", "Gross", "EE ESI", "ER ESI", "Total ESI"}, rows)
}

// GET /hr/payroll/export/26q?financial_year=&quarter=
func export26QCSV(c *fiber.Ctx) error {
	fy := c.Query("financial_year")
	quarter := c.Query("quarter")
	if fy == "" || quarter == "" {
		return c.Status(400).JSON(fiber.Map{"error": "financial_year and quarter required"})
	}
	lines, _, _, _, _, _, err := computeForm26Q(fy, quarter)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	rows := make([][]string, 0, len(lines))
	for _, l := range lines {
		pan := l.PAN
		if pan == "" {
			pan = "MISSING"
		}
		rows = append(rows, []string{
			strconv.FormatUint(uint64(l.DoctorUserID), 10), l.DoctorName, pan, l.Section,
			strconv.Itoa(l.ConsultCount), paiseStr(l.GrossPaise), paiseStr(l.TDSPaise),
		})
	}
	return sendCSV(c, fmt.Sprintf("26q_%s_%s.csv", fy, quarter),
		[]string{"User ID", "Deductee", "PAN", "Section", "Consults", "Gross", "TDS"}, rows)
}

// GET /hr/payroll/export/attendance?user_id=&month=&year=
func exportAttendanceCSV(c *fiber.Ctx) error {
	userID := c.QueryInt("user_id")
	month := c.QueryInt("month")
	year := c.QueryInt("year")
	if userID <= 0 || month < 1 || month > 12 || year < 2000 {
		return c.Status(400).JSON(fiber.Map{"error": "user_id, month (1-12) and year required"})
	}
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	var records []StaffAttendance
	database.DB.Where("user_id = ? AND date >= ? AND date < ?", userID, start.Format("2006-01-02"), end.Format("2006-01-02")).
		Order("date ASC").Find(&records)
	rows := make([][]string, 0, len(records))
	for _, r := range records {
		rows = append(rows, []string{r.Date.Format("2006-01-02"), r.Status})
	}
	return sendCSV(c, fmt.Sprintf("attendance_%d_%02d_%d.csv", userID, month, year),
		[]string{"Date", "Status"}, rows)
}

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	}
	return 0
}
