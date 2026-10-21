package handlers

// Payout HTTP handlers.
//
// Admin endpoints (require admin role):
//   GET  /admin/payouts               — paginated list of all payout records
//   GET  /admin/payouts/summary       — per-doctor totals (paid, pending, last payout)
//   GET  /admin/payouts/doctor/:id    — payout history for one doctor
//   POST /admin/payouts/trigger       — on-demand payout run (all pending or one doctor)
//
// Doctor endpoint (requires doctor role):
//   PUT  /doctors/me/payout-details   — update UPI / bank account for payouts

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	payoutdomain "drayya-core/internal/domain/payout"
	"drayya-core/internal/middleware"
	"drayya-core/internal/services/email"
	"drayya-core/internal/services/notification"
	rz "drayya-core/internal/services/razorpay"
	"drayya-core/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

var (
	upiRegex  = regexp.MustCompile(`^[a-zA-Z0-9.\-_]{2,256}@[a-zA-Z]{2,64}$`)
	ifscRegex = regexp.MustCompile(`^[A-Z]{4}0[A-Z0-9]{6}$`)
	panRegex  = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]$`) // e.g. ABCDE1234F
)

type PayoutHandler struct {
	finSvc finance.Service
}

func NewPayoutHandler(finSvc finance.Service) *PayoutHandler {
	return &PayoutHandler{finSvc: finSvc}
}

// ListPayouts returns all payout records, newest first.
// GET /admin/payouts?status=processed&page=1&limit=50
func (h *PayoutHandler) ListPayouts(c *fiber.Ctx) error {
	db := database.DB
	status := c.Query("status")
	limit := c.QueryInt("limit", 50)
	page := c.QueryInt("page", 1)
	if limit > 200 {
		limit = 200
	}
	offset := (page - 1) * limit

	q := db.Model(&payoutdomain.Record{}).Order("created_at DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	q.Count(&total)

	var records []payoutdomain.Record
	q.Offset(offset).Limit(limit).Find(&records)

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"records": records,
		"total":   total,
		"page":    page,
		"limit":   limit,
	}, "Payout records fetched")
}

// PayoutSummary returns an aggregated view per doctor.
// GET /admin/payouts/summary
func (h *PayoutHandler) PayoutSummary(c *fiber.Ctx) error {
	db := database.DB

	type doctorSummary struct {
		DoctorUserID    uint    `json:"doctor_user_id"`
		FullName        string  `json:"full_name"`
		TotalPaidPaise  int64   `json:"total_paid_paise"`
		TotalPaidINR    float64 `json:"total_paid_inr"`
		PendingPaise    int64   `json:"pending_paise"`
		PendingINR      float64 `json:"pending_inr"`
		LastPayoutAt    *time.Time `json:"last_payout_at"`
		PayoutCount     int64   `json:"payout_count"`
	}

	// Aggregate paid amounts per doctor from payout_records
	type aggRow struct {
		DoctorUserID uint
		TotalPaid    int64
		Count        int64
		LastAt       *time.Time
	}
	var agg []aggRow
	db.Raw(`
		SELECT doctor_user_id,
		       SUM(amount_paise) AS total_paid,
		       COUNT(*)          AS count,
		       MAX(created_at)   AS last_at
		FROM payout_records
		WHERE status = 'processed'
		GROUP BY doctor_user_id
	`).Scan(&agg)

	// Build map doctor_id → aggregated
	aggMap := map[uint]*aggRow{}
	for i := range agg {
		aggMap[agg[i].DoctorUserID] = &agg[i]
	}

	// Find all doctors with pending (unpaid) completed appointments. Group by
	// (doctor, fee) and apply the canonical finance.ComputeConsultationSplit so the
	// pending net matches exactly what TriggerPayout will pay (previously a flat
	// 60% SQL expression that disagreed with the ledger).
	type pendingFeeRow struct {
		DoctorID        uint
		ConsultationFee int64
		Cnt             int64
	}
	var pendingFees []pendingFeeRow
	db.Raw(`
		SELECT doctor_id, consultation_fee, COUNT(*) AS cnt
		FROM appointments
		WHERE status = 'completed' AND payout_status = 'pending' AND doctor_id IS NOT NULL
		GROUP BY doctor_id, consultation_fee
	`).Scan(&pendingFees)
	pendMap := map[uint]int64{}
	for _, p := range pendingFees {
		net := finance.ComputeConsultationSplit(p.ConsultationFee).DoctorNet
		pendMap[p.DoctorID] += net * p.Cnt
	}

	// Collect all involved doctor IDs
	doctorIDs := map[uint]struct{}{}
	for id := range aggMap {
		doctorIDs[id] = struct{}{}
	}
	for id := range pendMap {
		doctorIDs[id] = struct{}{}
	}

	// Fetch doctor names
	var users []identity.User
	ids := make([]uint, 0, len(doctorIDs))
	for id := range doctorIDs {
		ids = append(ids, id)
	}
	if len(ids) > 0 {
		db.Where("id IN ?", ids).Find(&users)
	}
	nameMap := map[uint]string{}
	for _, u := range users {
		nameMap[u.ID] = u.FullName
	}

	summaries := make([]doctorSummary, 0, len(doctorIDs))
	for docID := range doctorIDs {
		s := doctorSummary{
			DoctorUserID: docID,
			FullName:     nameMap[docID],
			PendingPaise: pendMap[docID],
			PendingINR:   float64(pendMap[docID]) / 100.0,
		}
		if a, ok := aggMap[docID]; ok {
			s.TotalPaidPaise = a.TotalPaid
			s.TotalPaidINR = float64(a.TotalPaid) / 100.0
			s.LastPayoutAt = a.LastAt
			s.PayoutCount = a.Count
		}
		summaries = append(summaries, s)
	}

	return response.JSON(c, fiber.StatusOK, summaries, "Payout summary fetched")
}

// DoctorPayoutHistory returns all payout records for one doctor.
// GET /admin/payouts/doctor/:id
func (h *PayoutHandler) DoctorPayoutHistory(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil || id == 0 {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ID", "Invalid doctor user_id")
	}
	var records []payoutdomain.Record
	database.DB.Where("doctor_user_id = ?", id).Order("created_at DESC").Find(&records)
	return response.JSON(c, fiber.StatusOK, records, "Doctor payout history fetched")
}

// TriggerPayout runs an on-demand payout.
// POST /admin/payouts/trigger   body: { doctor_user_id?: number }
// If doctor_user_id is omitted, all pending payouts are processed.
// A pessimistic lock (SELECT FOR UPDATE SKIP LOCKED) prevents double-payouts
// when multiple admin calls race.
func (h *PayoutHandler) TriggerPayout(c *fiber.Ctx) error {
	var req struct {
		DoctorUserID *uint `json:"doctor_user_id"`
	}
	c.BodyParser(&req) // optional body

	ctx := context.Background()

	var results []fiber.Map

	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		query := tx.
			Set("gorm:query_option", "FOR UPDATE SKIP LOCKED").
			Where("status = ? AND payout_status = ?", "completed", "pending").
			Preload("Doctor")
		if req.DoctorUserID != nil && *req.DoctorUserID != 0 {
			query = query.Where("doctor_id = ?", *req.DoctorUserID)
		}

		var appts []appointment.Appointment
		if err := query.Find(&appts).Error; err != nil {
			return err
		}
		if len(appts) == 0 {
			return nil // nothing to do; results stays empty
		}

		// Group by doctor
		type doctorGroup struct {
			DoctorUserID   uint
			TotalPaise     int64
			AppointmentIDs []uint
		}
		groups := map[uint]*doctorGroup{}
		for _, a := range appts {
			if a.DoctorID == nil {
				continue
			}
			docID := *a.DoctorID
			if _, ok := groups[docID]; !ok {
				groups[docID] = &doctorGroup{DoctorUserID: docID}
			}
			// Canonical split (commission + GST + TDS) — the single source of truth.
			// Previously a flat 60% was used here, which disagreed with the ledger's
			// 58.14% net and made payouts impossible to reconcile.
			groups[docID].TotalPaise += finance.ComputeConsultationSplit(a.ConsultationFee).DoctorNet
			groups[docID].AppointmentIDs = append(groups[docID].AppointmentIDs, a.ID)
		}

		for doctorUserID, grp := range groups {
			if grp.TotalPaise <= 0 {
				continue
			}
			var user identity.User
			if err := tx.First(&user, doctorUserID).Error; err != nil {
				results = append(results, fiber.Map{"doctor_user_id": doctorUserID, "status": "failed", "error": "user not found"})
				continue
			}
			var profile identity.DoctorProfile
			if err := tx.Where("user_id = ?", doctorUserID).First(&profile).Error; err != nil {
				results = append(results, fiber.Map{"doctor_user_id": doctorUserID, "status": "failed", "error": "profile not found"})
				continue
			}

			refID := fmt.Sprintf("PAYOUT-DOC%d-%d", doctorUserID, time.Now().UnixNano())
			fundAcctID, err := resolveFundAccountForHandler(profile, user)
			if err != nil {
				tx.Create(&payoutdomain.Record{
					DoctorUserID: doctorUserID, AmountPaise: grp.TotalPaise,
					RefID: refID, Status: "failed",
					Notes: "No fund account: " + err.Error(),
				})
				results = append(results, fiber.Map{"doctor_user_id": doctorUserID, "status": "failed", "error": err.Error()})
				continue
			}

			payoutID, err := rz.CreatePayout(rz.PayoutInput{
				FundAccountID: fundAcctID, AmountPaise: grp.TotalPaise,
				Currency: "INR", Mode: "IMPS", Purpose: "payout",
				Narration: fmt.Sprintf("Dr.Ayya earnings Dr.%s", user.FullName),
				RefID:     refID,
			})
			if err != nil {
				tx.Create(&payoutdomain.Record{
					DoctorUserID: doctorUserID, AmountPaise: grp.TotalPaise,
					RefID: refID, Status: "failed",
					Notes: "Razorpay: " + err.Error(),
				})
				results = append(results, fiber.Map{"doctor_user_id": doctorUserID, "status": "failed", "error": err.Error()})
				continue
			}

			// Ledger entries
			heldAcct, _ := h.finSvc.GetOrCreateAccount(ctx, "HeldFunds", finance.TypeLiability)
			docAcct, _ := h.finSvc.GetOrCreateAccount(ctx,
				fmt.Sprintf("DoctorEarnings:%d", doctorUserID), finance.TypeAsset)
			if heldAcct != nil && docAcct != nil {
				h.finSvc.RecordTransaction(ctx, refID, "DOCTOR_PAYOUT",
					fmt.Sprintf("On-demand payout to %s (%s)", user.FullName, payoutID),
					[]finance.EntryData{
						{AccountID: heldAcct.ID, Direction: finance.DirectionDebit, Amount: grp.TotalPaise},
						{AccountID: docAcct.ID, Direction: finance.DirectionCredit, Amount: grp.TotalPaise},
					})
			}

			tx.Model(&appointment.Appointment{}).
				Where("id IN ?", grp.AppointmentIDs).
				Update("payout_status", "processed")

			tx.Create(&payoutdomain.Record{
				DoctorUserID:     doctorUserID,
				AmountPaise:      grp.TotalPaise,
				RazorpayPayoutID: payoutID,
				RefID:            refID,
				Status:           "processed",
				Mode:             "IMPS",
				AppointmentCount: len(grp.AppointmentIDs),
				Notes:            fmt.Sprintf("On-demand payout — %d appointments", len(grp.AppointmentIDs)),
			})

			notification.WhatsAppDoctorPayout(user.PhoneNumber, user.FullName,
				fmt.Sprintf("%.0f", float64(grp.TotalPaise)/100.0), payoutID)
			notification.EnqueueEmail(context.Background(),
				user.Email,
				"Payout Transferred — Dr.Ayya",
				email.DoctorPayoutEmail(email.PayoutEmailData{
					DoctorName:       user.FullName,
					Amount:           float64(grp.TotalPaise) / 100.0,
					PayoutID:         payoutID,
					AppointmentCount: len(grp.AppointmentIDs),
					Date:             time.Now().Format("02 Jan 2006, 3:04 PM"),
				}),
			)

			results = append(results, fiber.Map{
				"doctor_user_id":     doctorUserID,
				"doctor_name":        user.FullName,
				"amount_inr":         float64(grp.TotalPaise) / 100.0,
				"razorpay_payout_id": payoutID,
				"status":             "processed",
			})
		}
		return nil
	})

	if txErr != nil {
		return response.Error(c, fiber.StatusInternalServerError, "PAYOUT_TX_ERROR", txErr.Error())
	}

	if len(results) == 0 {
		return response.JSON(c, fiber.StatusOK, fiber.Map{"processed": 0}, "No pending payouts")
	}

	return response.JSON(c, fiber.StatusOK, fiber.Map{
		"processed": len(results),
		"results":   results,
	}, "Payout run complete")
}

// UpdatePayoutDetails lets a doctor update their UPI / bank details.
// PUT /doctors/me/payout-details
func UpdatePayoutDetails(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uint)
	if !ok || userID == 0 {
		return response.Error(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	}
	var req struct {
		PayoutUPI         string `json:"payout_upi"`
		BankAccountNumber string `json:"bank_account_number"`
		BankIFSC          string `json:"bank_ifsc"`
		PAN               string `json:"pan"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
	}
	if req.PayoutUPI == "" && (req.BankAccountNumber == "" || req.BankIFSC == "") && req.PAN == "" {
		return response.Error(c, fiber.StatusBadRequest, "MISSING_DETAILS",
			"Provide payout_upi, both bank_account_number and bank_ifsc, or pan")
	}

	// Validate UPI format
	if req.PayoutUPI != "" && !upiRegex.MatchString(req.PayoutUPI) {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_UPI", "Invalid UPI ID format (expected format: name@bank)")
	}
	// Validate IFSC format
	if req.BankIFSC != "" && !ifscRegex.MatchString(req.BankIFSC) {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_IFSC", "Invalid IFSC code format (expected: 4 letters + 0 + 6 alphanumeric)")
	}
	// Validate bank account number length
	if req.BankAccountNumber != "" && (len(req.BankAccountNumber) < 9 || len(req.BankAccountNumber) > 18) {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_ACCOUNT", "Bank account number must be 9-18 digits")
	}
	// Validate PAN format
	if req.PAN != "" && !panRegex.MatchString(req.PAN) {
		return response.Error(c, fiber.StatusBadRequest, "INVALID_PAN", "Invalid PAN format (expected: 5 letters + 4 digits + 1 letter, e.g. ABCDE1234F)")
	}

	updates := map[string]interface{}{
		"razorpay_fund_account_id": "", // reset cached fund account so it's recreated
	}
	if req.PayoutUPI != "" {
		updates["payout_upi"] = req.PayoutUPI
	}
	if req.BankAccountNumber != "" {
		updates["bank_account_number"] = req.BankAccountNumber
		updates["bank_ifsc"] = req.BankIFSC
	}
	if req.PAN != "" {
		// New/changed PAN resets verification — finance re-verifies before filing.
		updates["pan"] = req.PAN
		updates["pan_verified"] = false
	}

	result := database.DB.Model(&identity.DoctorProfile{}).
		Where("user_id = ?", userID).
		Updates(updates)
	if result.Error != nil {
		return response.Error(c, fiber.StatusInternalServerError, "UPDATE_FAILED", result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return response.Error(c, fiber.StatusNotFound, "PROFILE_NOT_FOUND", "Doctor profile not found")
	}
	return response.JSON(c, fiber.StatusOK, nil, "Payout details updated")
}

// resolveFundAccountForHandler is a copy of the function in payout/service.go
// without the circular import. Resolves or creates a Razorpay fund account.
func resolveFundAccountForHandler(profile identity.DoctorProfile, user identity.User) (string, error) {
	if profile.RazorpayFundAccountID != "" {
		return profile.RazorpayFundAccountID, nil
	}
	if profile.PayoutUPI == "" && (profile.BankAccountNumber == "" || profile.BankIFSC == "") {
		return "", fmt.Errorf("no bank/UPI payout details for doctor user_id=%d", user.ID)
	}
	contactID, err := rz.CreateContact(rz.ContactInput{
		Name: user.FullName, Email: user.Email, Phone: user.PhoneNumber,
		Type: "vendor", RefID: fmt.Sprintf("doctor:%d", user.ID),
	})
	if err != nil {
		return "", fmt.Errorf("create contact: %w", err)
	}
	var fundInput rz.FundAccountInput
	fundInput.ContactID = contactID
	if profile.PayoutUPI != "" {
		fundInput.AccountType = "vpa"
		fundInput.VPA = profile.PayoutUPI
	} else {
		fundInput.AccountType = "bank_account"
		fundInput.Name = user.FullName
		fundInput.IFSC = profile.BankIFSC
		fundInput.AccountNo = profile.BankAccountNumber
	}
	fundAccountID, err := rz.CreateFundAccount(fundInput)
	if err != nil {
		return "", fmt.Errorf("create fund account: %w", err)
	}
	database.DB.Model(&profile).Update("razorpay_fund_account_id", fundAccountID)
	return fundAccountID, nil
}

// RegisterPayoutRoutes wires all payout endpoints.
func RegisterPayoutRoutes(router fiber.Router, finSvc finance.Service) {
	h := NewPayoutHandler(finSvc)

	// Admin payout management
	admin := router.Group("/admin", middleware.Protected(), middleware.AdminOnly())
	admin.Get("/payouts", h.ListPayouts)
	admin.Get("/payouts/summary", h.PayoutSummary)
	admin.Get("/payouts/doctor/:id", h.DoctorPayoutHistory)
	admin.Post("/payouts/trigger", h.TriggerPayout)

	// Doctor self-service
	router.Put("/doctors/me/payout-details",
		middleware.Protected(),
		UpdatePayoutDetails,
	)
}
