package admin

// Admin Revenue Dashboard
// -----------------------
// GET /admin/revenue/dashboard  — complete revenue snapshot
// GET /admin/revenue/trends     — daily revenue for last N days (for charts)
// GET /admin/revenue/doctors    — top doctors by revenue generated + pending payouts
//
// All figures are sourced from:
//   • finance.LedgerTransaction  — typed transactions (TOPUP_RAZORPAY, CONSULTATION_PAYMENT…)
//   • appointments table          — consultation counts + payout_status
//   • payout_records              — individual payout history

import (
	"fmt"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	payoutdomain "drayya-core/internal/domain/payout"
	"drayya-core/internal/middleware"
	"drayya-core/internal/services/email"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// ─── helpers ────────────────────────────────────────────────────────────────

func ledgerSum(txnType string, from, to time.Time) float64 {
	db := database.DB
	var total int64
	q := db.Table("ledger_transaction_v2 t").
		Joins("JOIN ledger_entry_v2 e ON e.transaction_id = t.id").
		Joins("JOIN ledger_account_v2 a ON a.id = e.account_id").
		Where("t.type = ? AND e.direction = ? AND a.type = ?",
			txnType, finance.DirectionCredit, finance.TypeLiability)
	if !from.IsZero() {
		q = q.Where("t.created_at >= ?", from)
	}
	if !to.IsZero() {
		q = q.Where("t.created_at < ?", to)
	}
	q.Select("COALESCE(SUM(e.amount),0)").Scan(&total)
	return float64(total) / 100.0
}

func appointmentRevSum(from, to time.Time) float64 {
	db := database.DB
	var total int64
	q := db.Model(&appointment.Appointment{}).
		Where("status IN ?", []string{"completed", "confirmed", "in_consultation"}).
		Select("COALESCE(SUM(consultation_fee),0)")
	if !from.IsZero() {
		q = q.Where("created_at >= ?", from)
	}
	if !to.IsZero() {
		q = q.Where("created_at < ?", to)
	}
	q.Scan(&total)
	return float64(total) / 100.0
}

// ─── GetRevenueDashboard ─────────────────────────────────────────────────────

// GetRevenueDashboard returns a full revenue snapshot for the admin.
// GET /admin/revenue/dashboard
func GetRevenueDashboard(c *fiber.Ctx) error {
	db := database.DB
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -int(now.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	ytdStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())

	// ── Revenue summary ───────────────────────────────────────────────────────
	type periodSummary struct {
		Today     float64 `json:"today"`
		ThisWeek  float64 `json:"this_week"`
		ThisMonth float64 `json:"this_month"`
		YTD       float64 `json:"ytd"`
		AllTime   float64 `json:"all_time"`
	}

	consult := periodSummary{
		Today:     appointmentRevSum(todayStart, now),
		ThisWeek:  appointmentRevSum(weekStart, now),
		ThisMonth: appointmentRevSum(monthStart, now),
		YTD:       appointmentRevSum(ytdStart, now),
		AllTime:   appointmentRevSum(time.Time{}, time.Time{}),
	}

	// Wallet top-ups (credit into PatientWallet ledger accounts)
	topupToday := ledgerSum("TOPUP_RAZORPAY", todayStart, now)
	topupMonth := ledgerSum("TOPUP_RAZORPAY", monthStart, now)
	topupYTD := ledgerSum("TOPUP_RAZORPAY", ytdStart, now)
	topupAll := ledgerSum("TOPUP_RAZORPAY", time.Time{}, time.Time{})

	// ── Doctor payout totals ───────────────────────────────────────────────────
	var totalPaidPaise, pendingPaise int64
	db.Model(&payoutdomain.Record{}).
		Where("status = 'processed'").
		Select("COALESCE(SUM(amount_paise),0)").Scan(&totalPaidPaise)

	db.Raw(`
		SELECT COALESCE(SUM(consultation_fee * 80 / 100), 0)
		FROM appointments
		WHERE status = 'completed' AND payout_status = 'pending'
	`).Scan(&pendingPaise)

	var thisMonthPaidPaise int64
	db.Model(&payoutdomain.Record{}).
		Where("status = 'processed' AND created_at >= ?", monthStart).
		Select("COALESCE(SUM(amount_paise),0)").Scan(&thisMonthPaidPaise)

	// Pharmacy order revenue (from pharmacy_orders where payment_status = 'paid')
	type pharmRev struct {
		AllTime   int64
		ThisMonth int64
		Today     int64
	}
	var pharmRevData pharmRev
	db.Raw(`
		SELECT
			COALESCE(SUM(total_amount), 0) AS all_time,
			COALESCE(SUM(CASE WHEN created_at >= ? THEN total_amount ELSE 0 END), 0) AS this_month,
			COALESCE(SUM(CASE WHEN created_at >= ? THEN total_amount ELSE 0 END), 0) AS today
		FROM pharmacy_orders WHERE payment_status = 'paid'
	`, monthStart, todayStart).Scan(&pharmRevData)

	// ── Platform fees (20% of consultation revenue) ───────────────────────────
	platformFeeAllTime := consult.AllTime * 0.20
	platformFeeMonth := consult.ThisMonth * 0.20

	// ── Counts ────────────────────────────────────────────────────────────────
	var totalConsultations, todayConsultations int64
	db.Model(&appointment.Appointment{}).
		Where("status IN ?", []string{"completed", "confirmed"}).
		Count(&totalConsultations)
	db.Model(&appointment.Appointment{}).
		Where("status IN ? AND created_at >= ?", []string{"completed", "confirmed"}, todayStart).
		Count(&todayConsultations)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"summary": fiber.Map{
			"consultations": consult,
			"wallet_topups": fiber.Map{
				"today":      topupToday,
				"this_month": topupMonth,
				"ytd":        topupYTD,
				"all_time":   topupAll,
			},
			"platform_fees": fiber.Map{
				"rate_pct":   20,
				"all_time":   platformFeeAllTime,
				"this_month": platformFeeMonth,
			},
			"pharmacy": fiber.Map{
				"today":      float64(pharmRevData.Today) / 100.0,
				"this_month": float64(pharmRevData.ThisMonth) / 100.0,
				"all_time":   float64(pharmRevData.AllTime) / 100.0,
			},
		},
		"payouts": fiber.Map{
			"total_paid_inr":    float64(totalPaidPaise) / 100.0,
			"pending_inr":       float64(pendingPaise) / 100.0,
			"this_month_inr":    float64(thisMonthPaidPaise) / 100.0,
		},
		"counts": fiber.Map{
			"total_consultations": totalConsultations,
			"today_consultations": todayConsultations,
		},
		"as_of": now,
	}, "Revenue dashboard fetched")
}

// ─── GetRevenueTrends ─────────────────────────────────────────────────────────

// GetRevenueTrends returns daily consultation revenue for the last N days.
// GET /admin/revenue/trends?days=30
func GetRevenueTrends(c *fiber.Ctx) error {
	days := c.QueryInt("days", 30)
	if days < 1 || days > 365 {
		days = 30
	}

	db := database.DB
	now := time.Now()

	type dayRow struct {
		Date    string  `json:"date"`
		Revenue float64 `json:"revenue_inr"`
		Count   int64   `json:"consultation_count"`
	}

	result := make([]dayRow, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := time.Date(now.Year(), now.Month(), now.Day()-i, 0, 0, 0, 0, now.Location())
		dEnd := d.Add(24 * time.Hour)

		var total int64
		var count int64
		db.Model(&appointment.Appointment{}).
			Where("status IN ? AND created_at >= ? AND created_at < ?",
				[]string{"completed", "confirmed"}, d, dEnd).
			Select("COALESCE(SUM(consultation_fee),0)").Scan(&total)
		db.Model(&appointment.Appointment{}).
			Where("status IN ? AND created_at >= ? AND created_at < ?",
				[]string{"completed", "confirmed"}, d, dEnd).
			Count(&count)

		result = append(result, dayRow{
			Date:    d.Format("2006-01-02"),
			Revenue: float64(total) / 100.0,
			Count:   count,
		})
	}

	return response.JSON(c, fiber.StatusOK, result, "Revenue trends fetched")
}

// ─── GetTopDoctors ────────────────────────────────────────────────────────────

// GetTopDoctors returns top doctors by gross revenue generated.
// GET /admin/revenue/doctors?limit=10
func GetTopDoctors(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 10)
	if limit < 1 || limit > 100 {
		limit = 10
	}

	db := database.DB

	type doctorRow struct {
		DoctorUserID      uint    `json:"doctor_user_id"`
		FullName          string  `json:"full_name"`
		GrossRevINR       float64 `json:"gross_revenue_inr"`
		DoctorShareINR    float64 `json:"doctor_share_inr"`
		PlatformFeeINR    float64 `json:"platform_fee_inr"`
		Consultations     int64   `json:"consultations"`
		PendingPayoutINR  float64 `json:"pending_payout_inr"`
		TotalPaidINR      float64 `json:"total_paid_inr"`
		AverageRating     float64 `json:"average_rating"`
	}

	type apptAgg struct {
		DoctorID  uint
		GrossRev  int64
		Count     int64
		PendingP  int64
	}
	var agg []apptAgg
	db.Raw(`
		SELECT doctor_id,
		       SUM(consultation_fee)                                   AS gross_rev,
		       COUNT(*)                                                 AS count,
		       SUM(CASE WHEN payout_status='pending' THEN consultation_fee * 80/100 ELSE 0 END) AS pending_p
		FROM appointments
		WHERE status IN ('completed','confirmed') AND doctor_id IS NOT NULL
		GROUP BY doctor_id
		ORDER BY gross_rev DESC
		LIMIT ?
	`, limit).Scan(&agg)

	// Paid totals from payout_records
	type paidRow struct {
		DoctorUserID uint
		TotalPaid    int64
	}
	var paid []paidRow
	db.Raw(`
		SELECT doctor_user_id, SUM(amount_paise) AS total_paid
		FROM payout_records WHERE status='processed'
		GROUP BY doctor_user_id
	`).Scan(&paid)
	paidMap := map[uint]int64{}
	for _, p := range paid {
		paidMap[p.DoctorUserID] = p.TotalPaid
	}

	// Doctor profiles for rating
	docIDs := make([]uint, 0, len(agg))
	for _, a := range agg {
		docIDs = append(docIDs, a.DoctorID)
	}
	var profiles []identity.DoctorProfile
	if len(docIDs) > 0 {
		db.Where("user_id IN ?", docIDs).Find(&profiles)
	}
	ratingMap := map[uint]float64{}
	for _, p := range profiles {
		ratingMap[p.UserID] = p.AverageRating
	}
	// Doctor names
	var users []identity.User
	if len(docIDs) > 0 {
		db.Where("id IN ?", docIDs).Find(&users)
	}
	nameMap := map[uint]string{}
	for _, u := range users {
		nameMap[u.ID] = u.FullName
	}

	rows := make([]doctorRow, 0, len(agg))
	for _, a := range agg {
		docShare := float64(a.GrossRev) * 0.80 / 100.0
		platFee := float64(a.GrossRev) * 0.20 / 100.0
		rows = append(rows, doctorRow{
			DoctorUserID:   a.DoctorID,
			FullName:       nameMap[a.DoctorID],
			GrossRevINR:    float64(a.GrossRev) / 100.0,
			DoctorShareINR: docShare,
			PlatformFeeINR: platFee,
			Consultations:  a.Count,
			PendingPayoutINR: float64(a.PendingP) / 100.0,
			TotalPaidINR:   float64(paidMap[a.DoctorID]) / 100.0,
			AverageRating:  ratingMap[a.DoctorID],
		})
	}

	return response.JSON(c, fiber.StatusOK, rows, "Top doctors by revenue fetched")
}

// ─── RegisterRevenueRoutes ────────────────────────────────────────────────────

// RegisterRevenueRoutes adds all revenue dashboard endpoints under /admin.
// Call from dashboard.go RegisterAdminRoutes.
// SendTestEmail sends a test email to verify the Resend integration.
// POST /admin/notifications/test-email
// Body: { "to": "someone@example.com", "template": "welcome|otp|reminder|prescription|payout|payment|topup" }
func SendTestEmail(c *fiber.Ctx) error {
	var body struct {
		To       string `json:"to"`
		Template string `json:"template"`
	}
	if err := c.BodyParser(&body); err != nil || body.To == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "to field is required")
	}

	template := body.Template
	if template == "" {
		template = "welcome"
	}

	var subject, html string
	switch template {
	case "otp":
		subject = "Your Dr.Ayya Login Code"
		html = email.StaffMFAEmail("Test User", "123456")
	case "reminder":
		subject = "Appointment Reminder — Dr.Ayya"
		html = email.AppointmentReminderEmail("Test Patient", "Dr. Test", "10:30 AM", 1)
	case "prescription":
		subject = "Prescription Issued — Dr.Ayya"
		html = email.PrescriptionIssuedEmail(email.PrescriptionEmailData{
			PatientName: "Test Patient",
			DoctorName:  "Dr. Test",
			Date:        time.Now().Format("02 Jan 2006"),
			Diagnosis:   "Test prescription",
			PrescID:     1,
		})
	case "payout":
		subject = "Payout Transferred — Dr.Ayya"
		html = email.DoctorPayoutEmail(email.PayoutEmailData{
			DoctorName:       "Dr. Test",
			Amount:           1250.00,
			PayoutID:         "pout_test123",
			AppointmentCount: 5,
			Date:             time.Now().Format("02 Jan 2006, 3:04 PM"),
		})
	case "payment":
		subject = "Payment Confirmed — Dr.Ayya"
		html = email.PaymentReceipt(email.PaymentEmailData{
			Name:        "Test Patient",
			Amount:      500.00,
			TxnID:       "pay_test123",
			Description: "Consultation with Dr. Test",
			Date:        time.Now().Format("02 Jan 2006, 3:04 PM"),
		})
	case "topup":
		subject = "Wallet Top-up Successful — Dr.Ayya"
		html = email.WalletTopUpEmail("Test User", 500.00, 1500.00, "pay_test123")
	default: // "welcome"
		subject = "Welcome to Dr.Ayya!"
		html = email.WelcomeEmail("Test User", "patient")
	}

	if err := email.SendSimple(body.To, subject, html); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SEND_FAILED", err.Error())
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"to":       body.To,
		"template": template,
		"subject":  subject,
	}, "Test email sent")
}

// SendTestWhatsApp sends a test WhatsApp template message to verify the Meta Cloud API integration.
// POST /admin/notifications/test-whatsapp
// Body: { "to": "+919876543210", "template": "otp|welcome|appointment|reminder|payment|topup|payout|prescription|cancelled|instant_timeout" }
func SendTestWhatsApp(c *fiber.Ctx) error {
	var body struct {
		To       string `json:"to"`
		Template string `json:"template"`
	}
	if err := c.BodyParser(&body); err != nil || body.To == "" {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "to field is required")
	}

	template := body.Template
	if template == "" {
		template = "welcome"
	}

	now := time.Now()
	dateStr := now.Format("02 Jan 2006")
	timeStr := now.Format("3:04 PM")

	var templateName string
	var params []string

	switch template {
	case "otp":
		templateName = "drayya_otp"
		params = []string{"123456"}
	case "appointment":
		templateName = "appointment_confirmed"
		params = []string{"Test Patient", "Dr. Test", dateStr, timeStr}
	case "reminder":
		templateName = "appointment_reminder_1hr"
		params = []string{"Test Patient", "Dr. Test", timeStr}
	case "cancelled":
		templateName = "appointment_cancelled"
		params = []string{"Test Patient", "Dr. Test", dateStr}
	case "doctor_appointment":
		templateName = "doctor_new_appointment"
		params = []string{"Dr. Test", "Test Patient", dateStr, timeStr}
	case "payment":
		templateName = "payment_receipt"
		params = []string{"Test Patient", "500", "Consultation with Dr. Test"}
	case "topup":
		templateName = "wallet_topup"
		params = []string{"Test User", "500", "1500"}
	case "payout":
		templateName = "doctor_payout_processed"
		params = []string{"Dr. Test", "1250", "pout_test123"}
	case "prescription":
		templateName = "prescription_issued"
		params = []string{"Test Patient", "Dr. Test", dateStr}
	case "instant_timeout":
		templateName = "instant_timeout_refund"
		params = []string{"Test Patient", "500"}
	default: // "welcome"
		templateName = "welcome_drayya"
		params = []string{"Test User", "patient"}
	}

	result, err := notification.SendTemplate(body.To, templateName, params)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "SEND_FAILED", err.Error())
	}

	messageID := ""
	if result != nil {
		messageID = result.MessageID
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"to":            body.To,
		"template":      template,
		"template_name": templateName,
		"params":        params,
		"message_id":    messageID,
		"note":          fmt.Sprintf("Template '%s' sent with %d param(s)", templateName, len(params)),
	}, "Test WhatsApp message sent")
}

func RegisterRevenueRoutes(adminGroup fiber.Router) {
	rev := adminGroup.Group("/revenue",
		middleware.Protected(),
		middleware.AdminOnly(),
	)
	rev.Get("/dashboard", GetRevenueDashboard)
	rev.Get("/trends", GetRevenueTrends)
	rev.Get("/doctors", GetTopDoctors)

	// Notification test endpoints under /admin/notifications/
	notif := adminGroup.Group("/notifications",
		middleware.Protected(),
		middleware.AdminOnly(),
	)
	notif.Post("/test-email", SendTestEmail)
	notif.Post("/test-whatsapp", SendTestWhatsApp)
}
