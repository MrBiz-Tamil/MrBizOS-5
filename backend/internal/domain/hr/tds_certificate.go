package hr

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"

	"github.com/gofiber/fiber/v2"
)

// DoctorTDSCertificate is Form 16A data for one freelance doctor over a period.
// TDS is withheld at source (10% of doctor gross) on every consultation and must
// be remitted to the government; this certificate is the doctor's proof of credit.
//
// All amounts are paise.
type DoctorTDSCertificate struct {
	DoctorUserID   uint   `json:"doctor_user_id"`
	DoctorName     string `json:"doctor_name"`
	PAN            string `json:"pan,omitempty"`
	PANVerified    bool   `json:"pan_verified"`
	From           string `json:"from"` // YYYY-MM-DD inclusive
	To             string `json:"to"`   // YYYY-MM-DD exclusive
	ConsultCount   int    `json:"consult_count"`
	GrossPaise     int64  `json:"gross_paise"`      // total consultation fees
	DoctorGross    int64  `json:"doctor_gross_paise"` // after commission + GST
	TDSPaise       int64  `json:"tds_paise"`        // tax deducted at source
	NetPaidPaise   int64  `json:"net_paid_paise"`   // doctor net (gross - commission - gst - tds)
	TDSRatePercent int    `json:"tds_rate_percent"` // statutory rate applied
}

// tdsCertificate generates a Form 16A / TDS certificate for a freelance doctor
// over a date range. Reuses finance.ComputeConsultationSplit so the TDS shown is
// exactly what the ledger withheld.
//
// GET /hr/payroll/tds-certificate?doctor_id=&from=YYYY-MM-DD&to=YYYY-MM-DD
//
// @Summary Doctor TDS Certificate (Form 16A)
// @Description Aggregate TDS withheld for a freelance doctor over a period.
// @Tags HR
// @Produce json
// @Security ApiKeyAuth
// @Param doctor_id query int true "Doctor user ID"
// @Param from query string true "Start date (YYYY-MM-DD, inclusive)"
// @Param to query string true "End date (YYYY-MM-DD, exclusive)"
// @Success 200 {object} response.StandardResponse{data=DoctorTDSCertificate} "Certificate generated"
// @Router /api/v1/hr/payroll/tds-certificate [get]
func tdsCertificate(c *fiber.Ctx) error {
	doctorID := c.QueryInt("doctor_id")
	from := c.Query("from")
	to := c.Query("to")
	if doctorID <= 0 || from == "" || to == "" {
		return c.Status(400).JSON(fiber.Map{"error": "doctor_id, from and to are required"})
	}
	cert := computeTDSCertificate(uint(doctorID), from, to)
	return c.JSON(cert)
}

// computeTDSCertificate builds a Form 16A certificate for a doctor over a period.
// Shared by the JSON endpoint and the printable HTML document.
func computeTDSCertificate(doctorID uint, from, to string) DoctorTDSCertificate {
	db := database.DB
	type feeBucket struct {
		ConsultationFee int64
		Cnt             int
	}
	var buckets []feeBucket
	db.Table("appointments").
		Select("consultation_fee, COUNT(*) AS cnt").
		Where("doctor_id = ? AND status = ? AND created_at >= ? AND created_at < ?",
			doctorID, "completed", from, to).
		Group("consultation_fee").
		Scan(&buckets)

	cert := DoctorTDSCertificate{
		DoctorUserID:   doctorID,
		From:           from,
		To:             to,
		TDSRatePercent: finance.TDSPercent,
	}
	for _, b := range buckets {
		if b.Cnt == 0 {
			continue
		}
		split := finance.ComputeConsultationSplit(b.ConsultationFee)
		cert.ConsultCount += b.Cnt
		cert.GrossPaise += b.ConsultationFee * int64(b.Cnt)
		cert.DoctorGross += split.DoctorGross * int64(b.Cnt)
		cert.TDSPaise += split.TDS * int64(b.Cnt)
		cert.NetPaidPaise += split.DoctorNet * int64(b.Cnt)
	}

	var user identity.User
	if err := db.First(&user, doctorID).Error; err == nil {
		cert.DoctorName = user.FullName
	}
	var prof struct {
		PAN         string
		PANVerified bool
	}
	db.Table("doctorprofile").Select("pan, pan_verified").Where("user_id = ?", doctorID).Scan(&prof)
	cert.PAN = prof.PAN
	cert.PANVerified = prof.PANVerified
	return cert
}
