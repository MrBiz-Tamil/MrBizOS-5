package hr

import (
	"fmt"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
)

// Form26QLine is one deductee row in a quarterly 26Q TDS return — the freelance
// counterpart to the EPF/ESI statements produced for salaried staff.
type Form26QLine struct {
	DoctorUserID uint   `json:"doctor_user_id"`
	DoctorName   string `json:"doctor_name"`
	PAN          string `json:"pan"`
	PANVerified  bool   `json:"pan_verified"`
	ConsultCount int    `json:"consult_count"`
	GrossPaise   int64  `json:"gross_paise"` // amount paid/credited
	TDSPaise     int64  `json:"tds_paise"`   // tax deducted
	Section      string `json:"section"`     // 194J
}

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

// rupees formats paise into Indian currency string, e.g. ₹1234.56 or -₹50.00.
func rupees(paise int64) string {
	sign := ""
	if paise < 0 {
		sign = "-"
		paise = -paise
	}
	return fmt.Sprintf("%s₹%d.%02d", sign, paise/100, paise%100)
}

// computeForm26Q builds the 26Q deductee lines + totals for a FY quarter.
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
