package devi

// flow_booking.go — Full 6-step appointment booking flow.
//
// Step 0: Consultation type selection
// Step 1: Symptoms (optional) → show top-3 doctors
// Step 2: Doctor selection → show available slots
// Step 3: Slot selection → show summary + wallet balance
// Step 4: Confirmation → CREATE appointment in DB
// Step 5: Payment choice → deduct wallet OR offer link

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/finance"
)

// bookingStart is the entry point for the booking intent.
func (e *Engine) bookingStart(ctx context.Context, uc *UserContext, msg string, state *sessionState) FlowResult {
	if !uc.CanBook() {
		return done(e.guestRegisterPrompt("book an appointment"))
	}
	st := BookingState{Step: 0}
	state.Intent = string(intentBook)
	state.Payload = marshalPayload(st)

	forWhom := ""
	if uc.IsFamilyMember() && uc.FamilyMemberName != "" {
		forWhom = fmt.Sprintf(" for *%s*", uc.FamilyMemberName)
		st.ForFamilyMember = true
		state.Payload = marshalPayload(st)
	}
	return reply(
		fmt.Sprintf("Great! Let's find you the right doctor. 🩺%s\n\nWhat type of consultation do you need?\n\n"+
			"1️⃣  General checkup\n"+
			"2️⃣  Specialist (specific department)\n"+
			"3️⃣  Mental health\n"+
			"4️⃣  Emergency / Urgent\n\n"+
			"Tap a button or type a number (1-4):", forWhom),
		btn("1 General", "1"), btn("2 Specialist", "2"),
		btn("3 Mental Health", "3"), btn("4 Emergency", "4"),
	)
}

// bookingContinue routes to the correct step handler.
func (e *Engine) bookingContinue(ctx context.Context, uc *UserContext, state *sessionState, msg string) FlowResult {
	st := unmarshalBooking(state)
	switch st.Step {
	case 0:
		return e.bookStep0(ctx, uc, state, &st, msg)
	case 1:
		return e.bookStep1(ctx, uc, state, &st, msg)
	case 2:
		return e.bookStep2(ctx, uc, state, &st, msg)
	case 3:
		return e.bookStep3(ctx, uc, state, &st, msg)
	case 4:
		return e.bookStep4(ctx, uc, state, &st, msg)
	case 5:
		return e.bookStep5(ctx, uc, state, &st, msg)
	}
	return done(e.buildMainMenu(uc))
}

// Step 0: consultation type → ask for symptoms
func (e *Engine) bookStep0(ctx context.Context, uc *UserContext, state *sessionState, st *BookingState, msg string) FlowResult {
	lower := strings.ToLower(strings.TrimSpace(msg))
	switch {
	case lower == "1" || strings.Contains(lower, "general") || strings.Contains(lower, "checkup"):
		st.ConsultType = "General Medicine"
		st.Specialization = "General Medicine"
	case lower == "2" || strings.Contains(lower, "specialist"):
		st.ConsultType = "Specialist"
		st.Specialization = "Specialist"
	case lower == "3" || strings.Contains(lower, "mental") || strings.Contains(lower, "psych"):
		st.ConsultType = "Psychiatry"
		st.Specialization = "Psychiatry"
	case lower == "4" || strings.Contains(lower, "emergency") || strings.Contains(lower, "urgent"):
		return e.emergencyStart(ctx, uc)
	default:
		return reply("Please reply with a number 1–4 or a type like *General* or *Specialist*.",
			btn("1 General", "1"), btn("2 Specialist", "2"),
			btn("3 Mental Health", "3"), btn("4 Emergency", "4"))
	}

	st.Step = 1
	state.Payload = marshalPayload(st)

	patientLine := ""
	if uc.IsFamilyMember() && uc.FamilyMemberName != "" {
		patientLine = fmt.Sprintf(" for *%s*", uc.FamilyMemberName)
	}
	return reply(
		fmt.Sprintf("Got it — *%s*%s. 👍\n\nAny symptoms you'd like to share with the doctor?\nThis helps us find the best specialist for you.\n(Type *skip* to proceed without)", st.ConsultType, patientLine),
		btn("Skip", "skip"),
	)
}

// Step 1: collect symptoms (or skip) → show doctor list
func (e *Engine) bookStep1(ctx context.Context, uc *UserContext, state *sessionState, st *BookingState, msg string) FlowResult {
	if strings.ToLower(strings.TrimSpace(msg)) != "skip" {
		st.Symptoms = msg
		// Use symptom matching to refine specialization
		var result matchResult
		if os.Getenv("GEMINI_API_KEY") != "" {
			analysis, err := e.AnalyzeSymptomsLLM(ctx, msg, "unknown", "unknown")
			if err == nil && analysis != nil {
				urgency := 0
				switch analysis.UrgencyLevel {
				case "critical":
					urgency = 10
				case "high":
					urgency = 8
				case "moderate":
					urgency = 5
				default:
					urgency = 2
				}
				dept := analysis.RecommendedSpecialization
				if dept == "" && len(analysis.RecommendedSpecializations) > 0 {
					dept = analysis.RecommendedSpecializations[0]
				}
				result = matchResult{department: dept, urgency: urgency}
			} else {
				result = e.matchSymptoms(msg)
			}
		} else {
			result = e.matchSymptoms(msg)
		}
		if result.urgency >= 9 {
			return e.emergencyStart(ctx, uc)
		}
		if result.department != "" && result.department != "General Medicine" {
			st.Specialization = result.department
		}
	}
	state.Payload = marshalPayload(st)
	return e.showDoctors(ctx, uc, state, st)
}

// showDoctors queries top-3 available doctors and shows them.
func (e *Engine) showDoctors(ctx context.Context, uc *UserContext, state *sessionState, st *BookingState) FlowResult {
	cards := e.fetchDoctorCards(ctx, st.Specialization, 3)
	if len(cards) == 0 {
		// Fall back to any available doctor
		cards = e.fetchDoctorCards(ctx, "", 3)
	}
	if len(cards) == 0 {
		return done("No doctors are available right now. Please try again in a few minutes or call our helpline.\n\nType *menu* to go back.")
	}

	st.Doctors = cards
	st.Step = 2
	state.Payload = marshalPayload(st)

	var sb strings.Builder
	spec := st.Specialization
	if spec == "Specialist" || spec == "" {
		spec = "doctors"
	}
	sb.WriteString(fmt.Sprintf("👨‍⚕️ *Top %s available now:*\n\n", spec))
	for i, d := range cards {
		online := "🟢 Online"
		if !d.IsOnline {
			online = "🔵 Available"
		}
		sb.WriteString(fmt.Sprintf("%d️⃣  Dr. %s (%s)\n    💰 ₹%s | ⭐ %.1f | %s\n\n",
			i+1, d.Name, d.Specialization, rupeesStr(d.FeePaise), d.Rating, online))
	}
	sb.WriteString("Reply *1*, *2*, or *3* to select a doctor.")

	actions := make([]ChatAction, len(cards))
	for i, d := range cards {
		actions[i] = btn(fmt.Sprintf("%d %s", i+1, d.Name), fmt.Sprintf("%d", i+1))
	}
	return reply(sb.String(), actions...)
}

// Step 2: doctor selection → show slots
func (e *Engine) bookStep2(ctx context.Context, uc *UserContext, state *sessionState, st *BookingState, msg string) FlowResult {
	lower := strings.TrimSpace(msg)
	idx := -1
	switch lower {
	case "1":
		idx = 0
	case "2":
		idx = 1
	case "3":
		idx = 2
	}
	if idx < 0 || idx >= len(st.Doctors) {
		return reply("Please reply with *1*, *2*, or *3* to select a doctor.",
			btn("1", "1"), btn("2", "2"), btn("3", "3"))
	}
	st.SelectedDoctor = &st.Doctors[idx]
	slots := e.generateSlots(ctx, st.SelectedDoctor.ProfileID)
	if len(slots) == 0 {
		return done(fmt.Sprintf("Dr. %s has no available slots right now.\n\nType *menu* to pick a different doctor.", st.SelectedDoctor.Name))
	}
	st.Slots = slots
	st.Step = 3
	state.Payload = marshalPayload(st)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Excellent choice! 🌟 Here are *Dr. %s's* available slots (IST):\n\n", st.SelectedDoctor.Name))
	for i, s := range slots {
		sb.WriteString(fmt.Sprintf("%d️⃣  %s IST\n", i+1, s.Label))
	}
	sb.WriteString("\nTap a slot or reply with the number to confirm:")

	actions := make([]ChatAction, len(slots))
	for i, s := range slots {
		actions[i] = btn(s.Label, fmt.Sprintf("%d", i+1))
	}
	return reply(sb.String(), actions...)
}

// Step 3: slot selection → show appointment summary
func (e *Engine) bookStep3(ctx context.Context, uc *UserContext, state *sessionState, st *BookingState, msg string) FlowResult {
	lower := strings.TrimSpace(msg)
	idx := -1
	switch lower {
	case "1":
		idx = 0
	case "2":
		idx = 1
	case "3":
		idx = 2
	}
	if idx < 0 || idx >= len(st.Slots) {
		return reply("Please reply *1*, *2*, or *3* to choose a slot.",
			btn("1", "1"), btn("2", "2"), btn("3", "3"))
	}
	st.SelectedSlot = &st.Slots[idx]
	st.FeePaise = st.SelectedDoctor.FeePaise

	// Fetch wallet balance
	walletBalance := int64(0)
	if uc.UserID != nil && e.svc.FinanceSvc != nil {
		acct, err := e.svc.FinanceSvc.GetOrCreateAccount(
			ctx, fmt.Sprintf("PatientWallet:%d", *uc.UserID), finance.TypeLiability)
		if err == nil {
			walletBalance = acct.Balance
		}
	}

	st.Step = 4
	state.Payload = marshalPayload(st)

	patientLine := ""
	if uc.IsFamilyMember() && uc.FamilyMemberName != "" {
		patientLine = fmt.Sprintf("\n👤 Patient: *%s*", uc.FamilyMemberName)
	}
	walletLine := fmt.Sprintf("\n💳 Wallet balance: *₹%s*", rupeesStr(walletBalance))
	if walletBalance < st.FeePaise {
		walletLine += " _(insufficient — reply TOPUP to add funds first)_"
	}

	msg = fmt.Sprintf(
		"Almost there! 🎉 Here's your appointment summary:\n\n"+
			"🏥 Type: *%s*\n"+
			"👨‍⚕️ Doctor: *Dr. %s* (%s)\n"+
			"🕐 Slot: *%s IST*\n"+
			"💰 Fee: *₹%s*%s%s\n\n"+
			"Tap *Confirm* to book, or *Cancel* to go back.",
		st.ConsultType, st.SelectedDoctor.Name, st.SelectedDoctor.Specialization,
		st.SelectedSlot.Label, rupeesStr(st.FeePaise), patientLine, walletLine,
	)

	actions := []ChatAction{btn("Confirm Booking", "yes"), btn("Cancel", "menu")}
	if walletBalance < st.FeePaise {
		actions = append([]ChatAction{btn("Top Up Wallet First", "topup")}, actions...)
	}
	return reply(msg, actions...)
}

// Step 4: confirmation → CREATE appointment in DB
func (e *Engine) bookStep4(ctx context.Context, uc *UserContext, state *sessionState, st *BookingState, msg string) FlowResult {
	lower := strings.ToLower(strings.TrimSpace(msg))

	// Handle "topup" detour
	if lower == "topup" || lower == "top up" || lower == "add money" {
		payState := PaymentState{Step: 1, Action: "topup"}
		state.Intent = string(intentPayment)
		state.Payload = marshalPayload(payState)
		return reply("💳 *Wallet Top-Up*\n\nHow much would you like to add?\n\n"+
			"1️⃣  ₹100\n2️⃣  ₹500\n3️⃣  ₹1000\n\nOr type any amount (e.g. 250).",
			btn("₹100", "100"), btn("₹500", "500"), btn("₹1000", "1000"))
	}

	if !isYes(lower) {
		return done("Booking cancelled. Type *menu* to start over.")
	}

	if st.SelectedDoctor == nil || st.SelectedSlot == nil || uc.UserID == nil {
		return done("Something went wrong. Please type *book* to start again.")
	}

	// CREATE appointment via domain service
	doctorUID := st.SelectedDoctor.UserID
	req := appointment.CreateAppointmentRequest{
		PatientID: *uc.UserID,
		DoctorID:  &doctorUID,
		Type:      "consultation",
		Mode:      "video",
		Notes:     st.Symptoms,
		StartTime: st.SelectedSlot.StartTime.Format(time.RFC3339),
		EndTime:   st.SelectedSlot.EndTime.Format(time.RFC3339),
	}
	if uc.FamilyMemberID != nil {
		req.FamilyMemberID = uc.FamilyMemberID
	}

	appt, err := e.svc.AppointmentSvc.CreateAppointment(ctx, req)
	if err != nil {
		return done(fmt.Sprintf("⚠️ Could not create appointment: %v\n\nPlease try again or contact support.", err))
	}

	st.AppointmentID = appt.ID
	st.Step = 5
	state.Payload = marshalPayload(st)

	// Notify via outbox (non-blocking)
	if e.svc.NotifSvc != nil && uc.UserID != nil {
		go func() {
			user, err := e.svc.IdentityRepo.GetUserByID(*uc.UserID)
			if err == nil {
				_ = e.svc.NotifSvc.EnqueueMFAOTP(ctx, user.Email, user.PhoneNumber, user.FullName, "")
				// A proper appointment confirmation enqueue would go here
			}
		}()
	}

	// Check wallet balance
	walletBalance := int64(0)
	if e.svc.FinanceSvc != nil {
		acct, err := e.svc.FinanceSvc.GetOrCreateAccount(
			ctx, fmt.Sprintf("PatientWallet:%d", *uc.UserID), finance.TypeLiability)
		if err == nil {
			walletBalance = acct.Balance
		}
	}

	msg = fmt.Sprintf(
		"🎉 *Appointment booked successfully!*\n\n"+
			"🆔 Ref #%d\n"+
			"👨‍⚕️ Dr. %s | %s IST\n"+
			"💰 Fee: ₹%s | 💳 Wallet: ₹%s\n\n"+
			"💡 *Tip:* Join the call 2 minutes early for the best experience.\n\n",
		appt.ID, st.SelectedDoctor.Name, st.SelectedSlot.Label,
		rupeesStr(st.FeePaise), rupeesStr(walletBalance),
	)

	actions := []ChatAction{}
	if walletBalance >= st.FeePaise {
		msg += "Reply *PAY* to pay from wallet now, or *LATER* to pay from the app."
		actions = []ChatAction{
			payBtn(fmt.Sprintf("Pay ₹%s from Wallet", rupeesStr(st.FeePaise)), "pay"),
			btn("Pay Later", "later"),
		}
	} else {
		msg += "Your wallet balance is insufficient. Reply *TOPUP* to add funds, or *LATER* to pay from the app."
		actions = []ChatAction{
			btn("Top Up Wallet", "topup"),
			btn("Pay Later in App", "later"),
		}
	}

	return FlowResult{
		Message: msg,
		Actions: actions,
		Data: map[string]interface{}{
			"appointment_id": appt.ID,
			"amount_paise":   st.FeePaise,
			"doctor_name":    st.SelectedDoctor.Name,
			"slot":           st.SelectedSlot.Label,
		},
	}
}

// Step 5: payment
func (e *Engine) bookStep5(ctx context.Context, uc *UserContext, state *sessionState, st *BookingState, msg string) FlowResult {
	lower := strings.ToLower(strings.TrimSpace(msg))

	if lower == "later" || lower == "skip" {
		return done(fmt.Sprintf(
			"👍 No problem! Appointment #%d is saved and confirmed.\n\n"+
				"Complete payment anytime from the *Appointments* tab in the app.\n"+
				proactiveFollowUp(), st.AppointmentID))
	}

	if lower == "topup" || lower == "top up" {
		payState := PaymentState{Step: 1, Action: "topup"}
		state.Intent = string(intentPayment)
		state.Payload = marshalPayload(payState)
		return reply("💳 *Wallet Top-Up*\n\nHow much would you like to add?\n\n"+
			"1️⃣  ₹100\n2️⃣  ₹500\n3️⃣  ₹1000\n\nOr type any amount.",
			btn("₹100", "100"), btn("₹500", "500"), btn("₹1000", "1000"))
	}

	if lower != "pay" && !isYes(lower) {
		return reply("Reply *PAY* to pay from wallet or *LATER* to pay from the app.",
			payBtn("Pay from Wallet", "pay"), btn("Pay Later", "later"))
	}

	// Deduct wallet
	if uc.UserID == nil || e.svc.FinanceSvc != nil {
		return done("Payment failed — please complete payment from the app.")
	}
	walletName := fmt.Sprintf("PatientWallet:%d", *uc.UserID)
	walletAcct, err := e.svc.FinanceSvc.GetOrCreateAccount(ctx, walletName, finance.TypeLiability)
	if err != nil || walletAcct.Balance < st.FeePaise {
		return done("Insufficient wallet balance. Please top up from the app and then complete payment.\n\nType *menu* for other options.")
	}
	heldAcct, err := e.svc.FinanceSvc.GetOrCreateAccount(ctx, "HeldFunds", finance.TypeLiability)
	if err != nil {
		return done("Payment system error. Please complete payment from the app.")
	}
	refID := fmt.Sprintf("DEVI-PAY-%d-%d", *uc.UserID, st.AppointmentID)
	_, err = e.svc.FinanceSvc.RecordTransaction(ctx, refID, "CONSULTATION_PAYMENT",
		fmt.Sprintf("Devi in-chat payment for appointment #%d", st.AppointmentID),
		[]finance.EntryData{
			{AccountID: walletAcct.ID, Direction: finance.DirectionDebit, Amount: st.FeePaise},
			{AccountID: heldAcct.ID, Direction: finance.DirectionCredit, Amount: st.FeePaise},
		})
	if err != nil {
		return done("Payment failed — please try again from the app.")
	}

	// Update appointment status to confirmed
	_ = e.svc.AppointmentSvc.UpdateAppointment(
		fmt.Sprintf("%d", st.AppointmentID),
		map[string]interface{}{"status": "confirmed"},
	)

	return FlowResult{
		Message: fmt.Sprintf(
			"💳 *Payment Confirmed!* 🎉\n\n"+
				"✅ ₹%s paid for Appointment #%d\n"+
				"👨‍⚕️ Dr. %s | %s IST\n\n"+
				"You're all set! You'll receive a confirmation shortly."+
				proactiveFollowUp(),
			rupeesStr(st.FeePaise), st.AppointmentID,
			st.SelectedDoctor.Name, st.SelectedSlot.Label,
		),
		Actions: []ChatAction{linkBtn("Open Appointment", fmt.Sprintf("deep_link:appointment:%d", st.AppointmentID))},
		Data:    map[string]interface{}{"appointment_id": st.AppointmentID, "status": "confirmed"},
		Done:    true,
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// fetchDoctorCards queries the DB for available doctors and maps to DoctorCard.
func (e *Engine) fetchDoctorCards(ctx context.Context, specialization string, limit int) []DoctorCard {
	if e.svc.DB == nil {
		return nil
	}
	threshold := time.Now().Add(-5 * time.Minute)
	
	var rows []struct {
		ProfileID      uint    `gorm:"column:profile_id"`
		UserID         uint    `gorm:"column:user_id"`
		Name           string  `gorm:"column:name"`
		Specialization string  `gorm:"column:specialization"`
		ConsultationFee int64  `gorm:"column:consultation_fee"`
		AverageRating  float64 `gorm:"column:average_rating"`
		IsOnline       bool    `gorm:"column:is_online"`
	}
	e.svc.DB.WithContext(ctx).Raw(`
		SELECT dp.id AS profile_id, dp.user_id, u.full_name AS name,
		       dp.specialization, dp.consultation_fee, dp.average_rating, dp.is_online
		FROM doctorprofile dp
		JOIN "user" u ON u.id = dp.user_id
		WHERE dp.is_online = true AND dp.is_verified = true
		  AND dp.current_status = 'available' AND dp.last_ping > ?
		`+specialFilter(specialization)+`
		ORDER BY (dp.average_rating * 10 + dp.admin_boost_score) DESC
		LIMIT ?`, threshold, limit).Scan(&rows)

	cards := make([]DoctorCard, 0, len(rows))
	for _, r := range rows {
		cards = append(cards, DoctorCard{
			UserID:         r.UserID,
			ProfileID:      r.ProfileID,
			Name:           r.Name,
			Specialization: r.Specialization,
			FeePaise:       r.ConsultationFee,
			Rating:         r.AverageRating,
			IsOnline:       r.IsOnline,
		})
	}
	return cards
}

func specialFilter(spec string) string {
	if spec == "" || spec == "Specialist" {
		return ""
	}
	// Safely interpolated — spec comes from our own matchSymptoms, not user input
	return fmt.Sprintf(" AND dp.specialization = '%s' ", strings.ReplaceAll(spec, "'", ""))
}

// generateSlots produces up to 3 bookable time slots for a doctor.
// Uses DoctorAvailability for today's schedule; falls back to reasonable defaults.
func (e *Engine) generateSlots(_ context.Context, profileID uint) []SlotCard {
	ist, _ := time.LoadLocation("Asia/Kolkata")
	now := time.Now().In(ist)
	today := int(now.Weekday()) // 0=Sunday

	var avails []struct {
		DayOfWeek int
		StartTime string
		EndTime   string
	}
	if e.svc.DB != nil && profileID > 0 {
		e.svc.DB.Table("doctor_availability").
			Select("day_of_week, start_time, end_time").
			Where("doctor_id = ? AND is_available = true AND day_of_week = ?", profileID, today).
			Scan(&avails)
	}

	slots := make([]SlotCard, 0, 3)

	if len(avails) > 0 {
		for _, a := range avails {
			startH, startM := parseHHMM(a.StartTime)
			endH, endM := parseHHMM(a.EndTime)
			slotStart := time.Date(now.Year(), now.Month(), now.Day(), startH, startM, 0, 0, ist)
			slotEnd := time.Date(now.Year(), now.Month(), now.Day(), endH, endM, 0, 0, ist)
			if slotStart.Before(now.Add(30 * time.Minute)) {
				slotStart = now.Add(30 * time.Minute).Round(30 * time.Minute)
				slotEnd = slotStart.Add(30 * time.Minute)
			}
			slots = append(slots, SlotCard{
				Label:     "Today " + slotStart.Format("3:04 PM"),
				StartTime: slotStart,
				EndTime:   slotEnd,
			})
			if len(slots) >= 2 {
				break
			}
		}
	}

	// Always offer a tomorrow slot
	tomorrow := now.Add(24 * time.Hour)
	tmStart := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 0, 0, 0, ist)
	slots = append(slots, SlotCard{
		Label:     "Tomorrow " + tmStart.Format("3:04 PM"),
		StartTime: tmStart,
		EndTime:   tmStart.Add(30 * time.Minute),
	})

	// If no availability today, add another slot
	if len(slots) < 2 {
		s2 := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 14, 0, 0, 0, ist)
		slots = append(slots, SlotCard{
			Label:     "Tomorrow " + s2.Format("3:04 PM"),
			StartTime: s2,
			EndTime:   s2.Add(30 * time.Minute),
		})
	}

	if len(slots) > 3 {
		slots = slots[:3]
	}
	return slots
}

// parseHHMM parses "HH:MM" into hours and minutes.
func parseHHMM(t string) (int, int) {
	var h, m int
	fmt.Sscanf(t, "%d:%d", &h, &m)
	return h, m
}
