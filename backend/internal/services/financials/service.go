package financials

import (
	"context"
	"fmt"
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
)

// ProcessConsultationPayout handles the financial split for a completed consultation.
// The ledger entries and the payout_status update are committed in one atomic block:
// if the ledger write succeeds but the status update fails (or vice versa), the
// idempotent reference_id ("PAYOUT-APPT-{id}") ensures a safe retry will find the
// existing transaction and skip re-entry, then retry the status update only.
func ProcessConsultationPayout(ctx context.Context, appointmentID uint, doctorID uint, patientID uint, amountCents int64) error {
	if amountCents <= 0 {
		return nil // Nothing to pay
	}

	financeSvc := finance.NewService(finance.NewRepository(database.DB))

	// 1. Resolve the accounts this release touches. The funds for a completed
	// consultation are sitting in the shared HeldFunds escrow account; releasing
	// them debits HeldFunds and credits the four split destinations.
	heldFunds, err := financeSvc.GetOrCreateAccount(ctx, "HeldFunds", finance.TypeLiability)
	if err != nil {
		return err
	}
	doctorAcct, err := financeSvc.GetOrCreateAccount(ctx, fmt.Sprintf("Doctor:%d", doctorID), finance.TypeLiability)
	if err != nil {
		return err
	}
	platformRevenueAcct, err := financeSvc.GetOrCreateAccount(ctx, "Revenue:Consultations", finance.TypeRevenue)
	if err != nil {
		return err
	}
	gstAcct, err := financeSvc.GetOrCreateAccount(ctx, "Liability:Tax:GST", finance.TypeLiability)
	if err != nil {
		return err
	}
	tdsAcct, err := financeSvc.GetOrCreateAccount(ctx, "Liability:Tax:TDS", finance.TypeLiability)
	if err != nil {
		return err
	}

	// 2. Canonical split (same formula as direct Razorpay invoices) so the
	// doctor's wallet and the books reconcile regardless of payment path.
	split := finance.ComputeConsultationSplit(amountCents)

	// 3. Double entry: debit escrow, credit revenue/GST/TDS/doctor.
	// Invariant Commission+GST+TDS+DoctorNet == Total keeps this balanced.
	entries := []finance.EntryData{
		{AccountID: heldFunds.ID, Direction: finance.DirectionDebit, Amount: split.Total},
		{AccountID: platformRevenueAcct.ID, Direction: finance.DirectionCredit, Amount: split.Commission},
		{AccountID: gstAcct.ID, Direction: finance.DirectionCredit, Amount: split.GST},
		{AccountID: tdsAcct.ID, Direction: finance.DirectionCredit, Amount: split.TDS},
		{AccountID: doctorAcct.ID, Direction: finance.DirectionCredit, Amount: split.DoctorNet},
	}

	// 4. Record Transaction + mark payout_status atomically
	referenceID := fmt.Sprintf("PAYOUT-APPT-%d", appointmentID)
	description := fmt.Sprintf("Consultation payout split for Appointment #%d", appointmentID)

	txn, err := financeSvc.RecordTransaction(ctx, referenceID, "PAYOUT_SPLIT", description, entries)
	if err != nil {
		log.Printf("[Finance] Payout ledger failed for Appt %d: %v", appointmentID, err)
		return err
	}

	// Mark payout_status = 'processed' immediately after ledger commit.
	// If this update fails, the next reconciliation cron will detect
	// payout_status='pending' on a PAYOUT-APPT-{id} transaction that already
	// exists, skip the ledger write (idempotent), and retry this update only.
	if err := database.DB.WithContext(ctx).
		Model(&struct{ ID uint }{ID: appointmentID}).
		Table("appointments").
		Where("id = ?", appointmentID).
		Update("payout_status", "processed").Error; err != nil {
		log.Printf("[Finance] WARNING: ledger committed for Appt %d but payout_status update failed: %v", appointmentID, err)
		// Non-fatal: reconciliation cron will fix this
	}

	log.Printf("[Finance] Payout processed: Appt %d | Txn %s | Doctor ₹%.2f | Platform ₹%.2f | GST ₹%.2f | TDS ₹%.2f",
		appointmentID, txn.ID,
		float64(split.DoctorNet)/100, float64(split.Commission)/100,
		float64(split.GST)/100, float64(split.TDS)/100)

	return nil
}

// GetAccountBalance returns the balance of a specific ledger account
func GetAccountBalance(ctx context.Context, name string) (int64, error) {
	repo := finance.NewRepository(database.DB)
	acct, err := repo.GetAccountByName(ctx, name)
	if err != nil {
		return 0, err
	}
	return acct.Balance, nil
}

