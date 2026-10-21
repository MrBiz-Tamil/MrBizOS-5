package payout

// Doctor payout service.
// Flow:
//  1. Find every Doctor:%d ledger wallet that carries a positive balance —
//     this liability is the doctor's accrued, post-split net earnings, credited
//     at consultation completion / invoice payment. It is the single source of
//     truth for "what we owe the doctor", so no fee re-computation happens here.
//  2. Resolve the doctor's Razorpay fund account (or create one).
//  3. Issue the payout via Razorpay for the full wallet balance.
//  4. On success, drain the wallet: Debit Doctor:%d (liability) / Credit
//     Asset:Bank — keeping the books reconciled.
//
// Triggered weekly by the cron scheduler.

import (
	"context"
	"fmt"
	"log"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/identity"
	payoutdomain "drayya-core/internal/domain/payout"
	"drayya-core/internal/services/email"
	rz "drayya-core/internal/services/razorpay"
	"drayya-core/internal/services/notification"
)

// PayoutResult summarises one doctor's payout for the caller.
type PayoutResult struct {
	DoctorUserID uint
	AmountPaise  int64
	PayoutID     string
	Err          error
}

// ProcessWeeklyPayouts pays out every doctor whose Doctor:%d wallet carries a
// positive balance and drains that wallet on success. It records a
// payoutdomain.Record for every attempt (success or failure).
func ProcessWeeklyPayouts(finSvc finance.Service) {
	log.Println("[Payout] Starting weekly doctor payout run...")

	db := database.DB
	ctx := context.Background()

	// The amount owed to a doctor is exactly their accrued wallet balance —
	// already net of commission, GST and TDS at the time each consultation was
	// settled. No fee re-computation here (that was the source of double-pay).
	var wallets []finance.LedgerAccount
	if err := db.
		Where("name LIKE ? AND type = ? AND balance > 0", "Doctor:%", finance.TypeLiability).
		Find(&wallets).Error; err != nil {
		log.Printf("[Payout] DB error fetching doctor wallets: %v", err)
		return
	}

	if len(wallets) == 0 {
		log.Println("[Payout] No pending payouts.")
		return
	}

	for _, wallet := range wallets {
		var doctorUserID uint
		if _, err := fmt.Sscanf(wallet.Name, "Doctor:%d", &doctorUserID); err != nil {
			log.Printf("[Payout] Skipping unparseable wallet %q: %v", wallet.Name, err)
			continue
		}

		amountPaise := wallet.Balance
		if amountPaise <= 0 {
			continue
		}

		// Idempotency guard: if an admin already raised a payout request for this
		// doctor that hasn't settled yet, skip — settling it will drain the wallet.
		// Paying here too would double-pay.
		var openRequests int64
		db.Model(&finance.PayoutRequest{}).
			Where("doctor_id = ? AND status IN ?", doctorUserID, []string{"pending", "approved"}).
			Count(&openRequests)
		if openRequests > 0 {
			log.Printf("[Payout] Doctor %d has %d open payout request(s) — skipping cron payout", doctorUserID, openRequests)
			continue
		}

		// Fetch doctor's user record
		var user identity.User
		if err := db.First(&user, doctorUserID).Error; err != nil {
			log.Printf("[Payout] Doctor user %d not found: %v", doctorUserID, err)
			continue
		}

		// Fetch doctor profile for bank/UPI details
		var profile identity.DoctorProfile
		if err := db.Where("user_id = ?", doctorUserID).First(&profile).Error; err != nil {
			log.Printf("[Payout] DoctorProfile for user %d not found: %v", doctorUserID, err)
			continue
		}

		// Resolve or create Razorpay fund account
		fundAccountID, err := resolveFundAccount(profile, user)
		if err != nil {
			log.Printf("[Payout] Cannot resolve fund account for doctor %d: %v — skipping", doctorUserID, err)
			continue
		}

		refID := fmt.Sprintf("PAYOUT-DOC%d-%d", doctorUserID, time.Now().Unix())
		payoutID, err := rz.CreatePayout(rz.PayoutInput{
			FundAccountID: fundAccountID,
			AmountPaise:   amountPaise,
			Currency:      "INR",
			Mode:          "IMPS",
			Purpose:       "payout",
			Narration:     fmt.Sprintf("Dr.Ayya consultation earnings Dr.%s", user.FullName),
			RefID:         refID,
		})
		if err != nil {
			log.Printf("[Payout] Razorpay payout failed for doctor %d: %v", doctorUserID, err)
			// Record failure so admins can see it in the dashboard
			db.Create(&payoutdomain.Record{
				DoctorUserID: doctorUserID,
				AmountPaise:  amountPaise,
				RefID:        refID,
				Status:       "failed",
				Notes:        fmt.Sprintf("Razorpay error: %v", err),
			})
			continue
		}

		// Drain the wallet: Debit Doctor:%d (liability ↓) / Credit Asset:Bank
		// (asset ↓, cash leaving the business). This reconciles the ledger with
		// the money that just left via Razorpay.
		doctorAcct, derr := finSvc.GetOrCreateAccount(ctx, wallet.Name, finance.TypeLiability)
		bankAcct, berr := finSvc.GetOrCreateAccount(ctx, "Asset:Bank", finance.TypeAsset)
		if derr == nil && berr == nil {
			if _, lerr := finSvc.RecordTransaction(ctx, refID, "DOCTOR_PAYOUT",
				fmt.Sprintf("Weekly payout to %s (%s)", user.FullName, payoutID),
				[]finance.EntryData{
					{AccountID: doctorAcct.ID, Direction: finance.DirectionDebit, Amount: amountPaise},
					{AccountID: bankAcct.ID, Direction: finance.DirectionCredit, Amount: amountPaise},
				}); lerr != nil {
				// Money already left via Razorpay but the ledger drain failed —
				// flag loudly so finance can reconcile manually.
				log.Printf("[Payout] CRITICAL: Razorpay payout %s sent for doctor %d but ledger drain failed: %v",
					payoutID, doctorUserID, lerr)
			}
		}

		// Record the payout in payout_records
		rec := &payoutdomain.Record{
			DoctorUserID:     doctorUserID,
			AmountPaise:      amountPaise,
			Currency:         "INR",
			RazorpayPayoutID: payoutID,
			RefID:            refID,
			Status:           "processed",
			Mode:             "IMPS",
			Notes:            "Weekly wallet payout",
		}
		db.Create(rec)

		// Notify doctor via WhatsApp + Email (both via outbox — payout emails are high-value)
		notification.WhatsAppDoctorPayout(
			user.PhoneNumber,
			user.FullName,
			fmt.Sprintf("%.0f", float64(amountPaise)/100.0),
			payoutID,
		)
		notification.EnqueueEmail(context.Background(),
			user.Email,
			"Payout Transferred — Dr.Ayya",
			email.DoctorPayoutEmail(email.PayoutEmailData{
				DoctorName: user.FullName,
				Amount:     float64(amountPaise) / 100.0,
				PayoutID:   payoutID,
				Date:       time.Now().Format("02 Jan 2006, 3:04 PM"),
			}),
		)

		log.Printf("[Payout] Paid doctor %d (user %d) ₹%.2f — Razorpay payout: %s",
			profile.ID, doctorUserID, float64(amountPaise)/100.0, payoutID)
	}

	log.Println("[Payout] Weekly payout run complete.")
}

// resolveFundAccount returns the Razorpay fund account ID for a doctor,
// creating a contact + fund account if not already stored.
func resolveFundAccount(profile identity.DoctorProfile, user identity.User) (string, error) {
	// If the doctor has pre-stored their Razorpay fund account ID, use it directly
	if profile.RazorpayFundAccountID != "" {
		return profile.RazorpayFundAccountID, nil
	}

	// Need bank or UPI to proceed
	if profile.PayoutUPI == "" && (profile.BankAccountNumber == "" || profile.BankIFSC == "") {
		return "", fmt.Errorf("no bank/UPI payout details on file for doctor user_id=%d", user.ID)
	}

	// Create contact
	contactID, err := rz.CreateContact(rz.ContactInput{
		Name:  user.FullName,
		Email: user.Email,
		Phone: user.PhoneNumber,
		Type:  "vendor",
		RefID: fmt.Sprintf("doctor:%d", user.ID),
	})
	if err != nil {
		return "", fmt.Errorf("create contact: %w", err)
	}

	// Create fund account (prefer UPI for instant transfer)
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

	// Cache the fund account ID on the doctor profile
	database.DB.Model(&profile).Update("razorpay_fund_account_id", fundAccountID)

	return fundAccountID, nil
}
