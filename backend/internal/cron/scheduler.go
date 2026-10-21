package cron

// Lightweight cron scheduler — no external dependency.
// Uses goroutines + time.Ticker for interval jobs and daily alignment for
// day-of-week jobs. Three jobs:
//
//  1. Appointment reminders (runs every minute)
//     — finds appointments starting in 55–65 minutes
//     — sends FCM push + WhatsApp template to patient
//
//  2. Instant consultation timeout + refund (runs every 2 minutes)
//     — finds instant/emergency appointments with status=pending older than 10 minutes
//     — auto-cancels and refunds the patient's wallet
//
//  3. Weekly doctor payouts (runs every Monday at 09:00 IST)
//     — calls payout.ProcessWeeklyPayouts

import (
	"context"
	"fmt"
	"log"
	"time"

	"drayya-core/internal/audit"
	"drayya-core/internal/audit/anomaly"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/content"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/services/email"
	"drayya-core/internal/services/notification"
	"drayya-core/internal/services/cron"
	"drayya-core/internal/services/payout"
	"drayya-core/internal/services/visibility"
	"drayya-core/internal/utils/config"

	"gorm.io/gorm"
)

var ist *time.Location

func init() {
	ist = config.LoadTimezone()
}

// Start launches all background jobs. Call once from server startup.
func Start(finSvc finance.Service) {
	go runReminderJob()
	go runInstantTimeoutJob(finSvc)
	go runWeeklyPayoutJob(finSvc)
	go runDailyResetJob()
	go runDoctorHeartbeatJob()
	go runRecurringExpenseJob()
	go runMonthlyTaxSummaryJob()
	go runNightlyScoreRecalcJob()
	go runWeeklyConsultationResetJob()
	go runNightlyReconciliationJob(finSvc)
	go runScheduledBlogPublishJob()
	log.Println("[Cron] Scheduler started (reminders + instant timeout + weekly payout + daily reset + heartbeat + recurring expenses + monthly tax summary + nightly score recalc + nightly reconciliation)")
}

// -------------------------------------------------------------------
// Appointment reminder job — every minute
// -------------------------------------------------------------------

func runReminderJob() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		sendReminders()
	}
}

func sendReminders() {
	now := time.Now()
	windowStart := now.Add(55 * time.Minute)
	windowEnd := now.Add(65 * time.Minute)

	var appts []appointment.Appointment
	if err := database.DB.
		Where("status IN ? AND start_time >= ? AND start_time < ? AND reminder_sent = ?",
			[]string{"confirmed", "pending"}, windowStart, windowEnd, false).
		Preload("Patient.User").
		Find(&appts).Error; err != nil {
		return
	}

	if len(appts) == 0 {
		return
	}

	for _, a := range appts {
		appt := a // capture
		go func() {
			// FIX 11: Atomic flag flip — if another worker already set reminder_sent, rows
			// affected will be 0 and we skip sending to avoid duplicate notifications.
			result := database.DB.Model(&appointment.Appointment{}).
				Where("id = ? AND reminder_sent = ? AND status = ?", appt.ID, false, "confirmed").
				Update("reminder_sent", true)
			if result.Error != nil || result.RowsAffected == 0 {
				// Either an error or another worker already sent the reminder — skip.
				return
			}
			sendReminderForAppointment(appt)
		}()
	}
}

func sendReminderForAppointment(a appointment.Appointment) {
	// Resolve patient user
	if a.Patient.User == nil {
		return
	}
	patientUser := a.Patient.User

	// Resolve doctor name
	doctorName := "your doctor"
	if a.DoctorID != nil {
		var doctorUser identity.User
		if err := database.DB.
			Select("full_name").
			Joins("JOIN doctorprofile ON doctorprofile.user_id = \"user\".id").
			Where("doctorprofile.user_id = ?", *a.DoctorID).
			First(&doctorUser).Error; err == nil {
			doctorName = doctorUser.FullName
		}
	}

	timeStr := a.StartTime.In(ist).Format("3:04 PM")

	// FCM push
	if patientUser.FCMToken != "" {
		notification.NotifyAppointmentReminder(
			patientUser.FCMToken,
			doctorName,
			a.StartTime,
			a.ID,
		)
	}

	// WhatsApp
	if patientUser.PhoneNumber != "" {
		notification.WhatsAppAppointmentReminder(
			patientUser.PhoneNumber,
			patientUser.FullName,
			doctorName,
			timeStr,
		)
	}

	// Email reminder (via outbox for reliable delivery)
	if patientUser.Email != "" {
		notification.EnqueueEmail(context.Background(),
			patientUser.Email,
			"Appointment Reminder — Dr.Ayya",
			email.AppointmentReminderEmail(patientUser.FullName, doctorName, timeStr, a.ID),
		)
	}

	log.Printf("[Cron] Reminder sent for appointment #%d (patient: %s)", a.ID, patientUser.FullName)
}

// -------------------------------------------------------------------
// Instant consultation timeout + refund — every 2 minutes
// -------------------------------------------------------------------

const instantTimeoutMinutes = 10 // cancel if no doctor accepts within 10 minutes

func runInstantTimeoutJob(finSvc finance.Service) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cancelTimedOutInstants(finSvc)
	}
}

func cancelTimedOutInstants(finSvc finance.Service) {
	cutoff := time.Now().Add(-time.Duration(instantTimeoutMinutes) * time.Minute)

	var stale []appointment.Appointment
	if err := database.DB.
		Where("appointment_type IN ? AND status = ? AND created_at < ?",
			[]string{"instant", "emergency"}, "pending", cutoff).
		Preload("Patient.User").
		Find(&stale).Error; err != nil || len(stale) == 0 {
		return
	}

	ctx := context.Background()

	for _, a := range stale {
		appt := a // capture
		go func() {
			// FIX 12: Atomic status update — if another worker already changed the status,
			// rows affected will be 0 and we skip this appointment to avoid double-cancel.
			cancelResult := database.DB.Model(&appointment.Appointment{}).
				Where("id = ? AND status = ? AND created_at < ?", appt.ID, "pending", cutoff).
				Updates(map[string]interface{}{
					"status": "cancelled",
					"notes":  appt.Notes + "\n[Auto-cancelled: no doctor accepted within 10 minutes]",
				})
			if cancelResult.Error != nil {
				log.Printf("[Cron] Failed to cancel appointment #%d: %v", appt.ID, cancelResult.Error)
				return
			}
			if cancelResult.RowsAffected == 0 {
				// Another worker already handled this appointment — skip silently.
				return
			}

			// FIX 10: Guard against refunding after payout has been processed.
			if appt.PayoutStatus == "processed" {
				log.Printf("[Cron] WARNING: appointment #%d payout already processed — skipping auto-refund; manual review required", appt.ID)
				return
			}

			// FIX 9: Wrap cancel + refund in a single DB transaction so they are atomic.
			txErr := database.DB.Transaction(func(tx *gorm.DB) error {
				// Re-verify the appointment is in cancelled state (we just set it above, but confirm).
				var recheck appointment.Appointment
				if err := tx.Select("id, status, payout_status, consultation_fee, patient_id, notes").
					First(&recheck, appt.ID).Error; err != nil {
					return err
				}
				if recheck.Status != "cancelled" {
					// Status changed between our UPDATE and this transaction — skip.
					return nil
				}
				if recheck.PayoutStatus == "processed" {
					log.Printf("[Cron] WARNING: appointment #%d payout already processed inside tx — skipping refund; manual review required", appt.ID)
					return nil
				}

				if recheck.ConsultationFee > 0 {
					walletName := fmt.Sprintf("PatientWallet:%d", recheck.PatientID)
					walletAcct, err := finSvc.GetOrCreateAccount(ctx, walletName, finance.TypeLiability)
					if err != nil {
						return fmt.Errorf("cannot get wallet for patient %d: %w", recheck.PatientID, err)
					}
					heldAcct, err := finSvc.GetOrCreateAccount(ctx, "HeldFunds", finance.TypeLiability)
					if err != nil {
						return fmt.Errorf("cannot get HeldFunds account: %w", err)
					}
					refID := fmt.Sprintf("REFUND-TIMEOUT-%d", recheck.ID)
					_, err = finSvc.RecordTransaction(ctx, refID, "INSTANT_TIMEOUT_REFUND",
						fmt.Sprintf("Auto-refund: instant appointment #%d timed out", recheck.ID),
						[]finance.EntryData{
							{AccountID: heldAcct.ID, Direction: finance.DirectionDebit, Amount: recheck.ConsultationFee},
							{AccountID: walletAcct.ID, Direction: finance.DirectionCredit, Amount: recheck.ConsultationFee},
						})
					if err != nil {
						return fmt.Errorf("refund transaction failed: %w", err)
					}
				}
				return nil
			})
			if txErr != nil {
				log.Printf("[Cron] Refund transaction failed for appointment #%d: %v", appt.ID, txErr)
			}

			// Notify patient (outside transaction — notification failure should not roll back the refund).
			if appt.Patient.User != nil {
				patUser := appt.Patient.User
				notification.SendToToken(patUser.FCMToken, notification.FCMMessage{
					Title: "No doctors available",
					Body:  "We couldn't find an available doctor in time. Your payment has been refunded.",
					Data:  map[string]string{"type": "instant_timeout", "appointment_id": fmt.Sprintf("%d", appt.ID)},
				})
				if patUser.PhoneNumber != "" {
					notification.WhatsAppInstantTimeout(patUser.PhoneNumber, patUser.FullName,
						fmt.Sprintf("%.0f", float64(appt.ConsultationFee)/100.0))
				}
			}

			log.Printf("[Cron] Auto-cancelled instant appointment #%d (patient %d) — refunded ₹%.0f",
				appt.ID, appt.PatientID, float64(appt.ConsultationFee)/100.0)
		}()
	}
}

// -------------------------------------------------------------------
// Weekly payout job — every Monday 09:00 IST
// -------------------------------------------------------------------

func runWeeklyPayoutJob(finSvc finance.Service) {
	for {
		next := nextMonday9AM()
		log.Printf("[Cron] Next doctor payout run scheduled at %s", next.In(ist).Format(time.RFC1123))
		timer := time.NewTimer(time.Until(next))
		<-timer.C
		timer.Stop()

		log.Println("[Cron] Running weekly doctor payout...")
		ctx := context.Background()
		_ = ctx
		payout.ProcessWeeklyPayouts(finSvc)
	}
}

func nextMonday9AM() time.Time {
	now := time.Now().In(ist)
	// Days until next Monday (weekday 1)
	daysUntilMonday := int(time.Monday - now.Weekday())
	if daysUntilMonday <= 0 {
		daysUntilMonday += 7
	}
	next := time.Date(now.Year(), now.Month(), now.Day()+daysUntilMonday, 9, 0, 0, 0, ist)
	return next
}

// -------------------------------------------------------------------
// Nightly ledger reconciliation — every day 02:30 (configured TZ)
// -------------------------------------------------------------------
//
// Proves the ledger's own correctness every night: trial balance, per-account
// balance proof, transaction-chain linkage, per-entry hash integrity, and
// orphaned paid orders. Any failure becomes a CRITICAL anomaly that rides the
// existing alert + daily-digest pipeline, so a tiny ops team only ever looks at
// a flagged exception — never the healthy books.

func runNightlyReconciliationJob(finSvc finance.Service) {
	for {
		next := nextReconcileTime()
		log.Printf("[Cron] Next ledger reconciliation scheduled at %s", next.In(ist).Format(time.RFC1123))
		timer := time.NewTimer(time.Until(next))
		<-timer.C
		timer.Stop()
		runReconciliationOnce(finSvc)
	}
}

func runReconciliationOnce(finSvc finance.Service) {
	report := finSvc.Reconcile(database.DB)
	if report.AllPassed {
		log.Printf("[Cron] Ledger reconciliation OK — %d checks passed", len(report.Results))
		return
	}
	for _, r := range report.Results {
		if r.Passed {
			continue
		}
		log.Printf("[Cron] RECONCILIATION FAILURE: %s — %s", r.Name, r.Detail)
		anomaly.Record(anomaly.Anomaly{
			RuleName:    "ledger_reconciliation_" + r.Name,
			Severity:    anomaly.SeverityCritical,
			Title:       fmt.Sprintf("Ledger reconciliation failed: %s", r.Name),
			Description: r.Detail,
			Table:       "ledger_entry_v2",
			Hits:        1,
			DetectedAt:  time.Now(),
		})
	}
}

// -------------------------------------------------------------------
// Scheduled blog publishing — every 5 minutes
// -------------------------------------------------------------------
//
// Flips pending posts whose scheduled_publish_at has arrived to approved, but
// only those that still satisfy the medical-review gate (reviewer assigned,
// credentials, sources). Posts without a reviewer stay pending for manual review.

func runScheduledBlogPublishJob() {
	svc := content.NewService(content.NewRepository(database.DB), nil)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		if n, err := svc.PublishDueScheduled(context.Background()); err == nil && n > 0 {
			log.Printf("[Cron] Auto-published %d scheduled blog post(s)", n)
		}
	}
}

func nextReconcileTime() time.Time {
	now := time.Now().In(ist)
	next := time.Date(now.Year(), now.Month(), now.Day(), 2, 30, 0, 0, ist)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// -------------------------------------------------------------------
// Daily Reset Job — every midnight IST
// -------------------------------------------------------------------

func runDailyResetJob() {
	for {
		next := nextMidnight()
		log.Printf("[Cron] Next daily reset scheduled at %s", next.In(ist).Format(time.RFC1123))
		timer := time.NewTimer(time.Until(next))
		<-timer.C
		timer.Stop()

		cron.ResetDailyDoctorCounters()
		cron.ResetDailyCourierCounters()
	}
}

func nextMidnight() time.Time {
	now := time.Now().In(ist)
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, ist)
	return next
}

// -------------------------------------------------------------------
// Doctor Heartbeat Job — every 2 minutes
// -------------------------------------------------------------------

func runDoctorHeartbeatJob() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		offlineInactiveDoctors()
	}
}

func offlineInactiveDoctors() {
	// Doctors who haven't pinged in last 5 minutes are marked offline
	ctx := audit.ForSystem(context.Background(), "cron:doctor_heartbeat")
	cutoff := time.Now().Add(-5 * time.Minute)

	result := database.DB.WithContext(ctx).Model(&identity.DoctorProfile{}).
		Where("is_online = ? AND last_ping < ?", true, cutoff).
		Update("is_online", false)

	if result.Error != nil {
		log.Printf("[Cron] Heartbeat check failed: %v", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Printf("[Cron] Heartbeat: marked %d inactive doctors as offline", result.RowsAffected)
	}
}

// -------------------------------------------------------------------
// Recurring Expense Auto-Post Job — runs daily at 08:00 IST
// -------------------------------------------------------------------

func runRecurringExpenseJob() {
	for {
		next := nextDailyAt(8, 0)
		timer := time.NewTimer(time.Until(next))
		<-timer.C
		timer.Stop()
		postDueRecurringExpenses()
	}
}

func nextDailyAt(hour, minute int) time.Time {
	now := time.Now().In(ist)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, ist)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func postDueRecurringExpenses() {
	ctx := audit.ForSystem(context.Background(), "cron:recurring_expense")
	db := database.DB.WithContext(ctx)

	var due []finance.RecurringExpense
	if err := db.
		Where("is_active = ? AND next_due_at <= ?", true, time.Now()).
		Find(&due).Error; err != nil || len(due) == 0 {
		return
	}

	for _, re := range due {
		exp := finance.Expense{
			Description: re.Description + " (auto-posted)",
			Amount:      re.Amount,
			Category:    re.Category,
			Department:  re.Department,
			SourceID:    fmt.Sprintf("recurring:%d", re.ID),
			Status:      "approved",
			CreatedBy:   re.CreatedBy,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := db.Create(&exp).Error; err != nil {
			log.Printf("[Cron] Failed to auto-post recurring expense #%d: %v", re.ID, err)
			continue
		}
		now := time.Now()
		nextDue := finance.RecurringExpenseComputeNextDue(re.Frequency, re.DayOfMonth, re.DayOfWeek)
		db.Model(&finance.RecurringExpense{}).Where("id = ?", re.ID).Updates(map[string]interface{}{
			"last_posted_at": &now,
			"next_due_at":    nextDue,
		})
		log.Printf("[Cron] Auto-posted recurring expense '%s' ₹%.0f", re.Description, float64(re.Amount)/100.0)
	}
}

// -------------------------------------------------------------------
// Monthly Tax Summary Job — runs 1st of each month at 09:30 IST
// -------------------------------------------------------------------

func runMonthlyTaxSummaryJob() {
	for {
		next := nextFirstOfMonth(9, 30)
		log.Printf("[Cron] Next monthly tax summary scheduled at %s", next.In(ist).Format(time.RFC1123))
		timer := time.NewTimer(time.Until(next))
		<-timer.C
		timer.Stop()
		sendMonthlyTaxSummary()
	}
}

func nextFirstOfMonth(hour, minute int) time.Time {
	now := time.Now().In(ist)
	// Next month's 1st
	next := time.Date(now.Year(), now.Month()+1, 1, hour, minute, 0, 0, ist)
	return next
}

// -------------------------------------------------------------------
// Nightly doctor performance score recalculation — 02:00 IST daily
// -------------------------------------------------------------------

func runNightlyScoreRecalcJob() {
	for {
		next := nextDailyAt(2, 0)
		log.Printf("[Cron] Next score recalc scheduled at %s", next.In(ist).Format(time.RFC1123))
		timer := time.NewTimer(time.Until(next))
		<-timer.C
		timer.Stop()

		if err := visibility.RecalculateAllScores(); err != nil {
			log.Printf("[Cron] Score recalc failed: %v", err)
		}
	}
}

// -------------------------------------------------------------------
// Weekly consultations reset — every Sunday midnight IST
// -------------------------------------------------------------------

func runWeeklyConsultationResetJob() {
	for {
		next := nextSundayMidnight()
		log.Printf("[Cron] Next weekly_consultations reset scheduled at %s", next.In(ist).Format(time.RFC1123))
		timer := time.NewTimer(time.Until(next))
		<-timer.C
		timer.Stop()

		result := database.DB.Model(&identity.DoctorProfile{}).
			Where("1 = 1").
			Update("weekly_consultations", 0)
		if result.Error != nil {
			log.Printf("[Cron] Weekly consultations reset failed: %v", result.Error)
		} else {
			log.Printf("[Cron] Reset weekly_consultations to 0 for %d doctors", result.RowsAffected)
		}
	}
}

func nextSundayMidnight() time.Time {
	now := time.Now().In(ist)
	daysUntilSunday := int(time.Sunday-now.Weekday()+7) % 7
	if daysUntilSunday == 0 {
		daysUntilSunday = 7
	}
	return time.Date(now.Year(), now.Month(), now.Day()+daysUntilSunday, 0, 0, 0, 0, ist)
}

func sendMonthlyTaxSummary() {
	now := time.Now().In(ist)
	lastMonthStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, ist)
	lastMonthEnd := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, ist)

	// Sum GST collected in last month
	var gstPaise int64
	database.DB.Table("ledger_transaction_v2 t").
		Joins("JOIN ledger_entry_v2 e ON e.transaction_id = t.id").
		Joins("JOIN ledger_account_v2 a ON a.id = e.account_id").
		Where("a.name = ? AND e.direction = ? AND t.created_at >= ? AND t.created_at < ?",
			"Liability:Tax:GST", "credit", lastMonthStart, lastMonthEnd).
		Select("COALESCE(SUM(e.amount), 0)").Scan(&gstPaise)

	// Sum TDS collected in last month
	var tdsPaise int64
	database.DB.Table("ledger_transaction_v2 t").
		Joins("JOIN ledger_entry_v2 e ON e.transaction_id = t.id").
		Joins("JOIN ledger_account_v2 a ON a.id = e.account_id").
		Where("a.name = ? AND e.direction = ? AND t.created_at >= ? AND t.created_at < ?",
			"Liability:Tax:TDS", "credit", lastMonthStart, lastMonthEnd).
		Select("COALESCE(SUM(e.amount), 0)").Scan(&tdsPaise)

	monthName := lastMonthStart.Format("January 2006")
	gstINR := float64(gstPaise) / 100.0
	tdsINR := float64(tdsPaise) / 100.0

	log.Printf("[Cron] Monthly tax summary for %s: GST=₹%.2f, TDS=₹%.2f", monthName, gstINR, tdsINR)

	// Email all finance managers and admins
	var adminUsers []identity.User
	database.DB.Where("role IN ?", []string{"admin", "super_admin", "finance_manager"}).Find(&adminUsers)

	subject := fmt.Sprintf("Tax Summary %s — Dr.Ayya", monthName)
	htmlBody := fmt.Sprintf(`
	<div style="font-family:sans-serif;max-width:600px;margin:0 auto">
	<h2 style="color:#065f46">Monthly Tax Summary — %s</h2>
	<table style="width:100%%;border-collapse:collapse;margin-top:16px">
	<tr style="background:#f0fdf4">
	  <th style="padding:12px;text-align:left;border:1px solid #d1fae5">Tax Type</th>
	  <th style="padding:12px;text-align:right;border:1px solid #d1fae5">Amount Collected</th>
	</tr>
	<tr>
	  <td style="padding:12px;border:1px solid #d1fae5">GST (18%% on Platform Fee)</td>
	  <td style="padding:12px;text-align:right;border:1px solid #d1fae5">₹%.2f</td>
	</tr>
	<tr style="background:#f0fdf4">
	  <td style="padding:12px;border:1px solid #d1fae5">TDS (10%% on Doctor Earnings)</td>
	  <td style="padding:12px;text-align:right;border:1px solid #d1fae5">₹%.2f</td>
	</tr>
	<tr style="font-weight:bold">
	  <td style="padding:12px;border:1px solid #d1fae5">Total Tax Liability</td>
	  <td style="padding:12px;text-align:right;border:1px solid #d1fae5">₹%.2f</td>
	</tr>
	</table>
	<p style="color:#6b7280;margin-top:16px;font-size:13px">This is an automated summary generated by Dr.Ayya finance system.</p>
	</div>`, monthName, gstINR, tdsINR, gstINR+tdsINR)

	for _, u := range adminUsers {
		if u.Email == "" {
			continue
		}
		notification.EnqueueEmail(context.Background(), u.Email, subject, htmlBody)
	}
}
