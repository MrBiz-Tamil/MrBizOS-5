package billing

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/hospital"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/services/franchise"
	"drayya-core/internal/services/fx"
	"drayya-core/internal/services/razorpay"
	"drayya-core/internal/services/stripe"
)

type Service interface {
	CreateInvoice(appointmentID uint, amount int64, patientID uint) (*Billing, error)
	ProcessPaymentMock(invoiceID uint, paymentMethod string) (*Payment, error)
	VerifyPayment(invoiceID uint, razorpayOrderID, razorpayPaymentID, signature string) (*Payment, error)
	CreateRazorpayOrder(amount int, receipt string) (map[string]interface{}, error)
	ProcessCorporatePayment(amount int64, patientID uint) (map[string]interface{}, error)
	CreateStripeIntent(invoiceID uint, currency string) (map[string]interface{}, error)
	VerifyStripePayment(invoiceID uint, paymentIntentID string) (*Payment, error)
}

type service struct {
	repo           Repository
	financeService finance.Service
}

func NewService(repo Repository, financeService finance.Service) Service {
	return &service{
		repo:           repo,
		financeService: financeService,
	}
}

func (s *service) CreateInvoice(appointmentID uint, amount int64, patientID uint) (*Billing, error) {
	_, err := s.repo.GetBillingByAppointment(appointmentID)
	if err == nil {
		return nil, errors.New("billing record already exists")
	}

	invoice := &Billing{
		AppointmentID: &appointmentID,
		PatientID:     patientID,
		Amount:        amount,
		PaymentStatus: "pending",
		InvoiceDate:   time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.CreateBilling(invoice); err != nil {
		return nil, err
	}

	return invoice, nil
}

func (s *service) ProcessPaymentMock(invoiceID uint, paymentMethod string) (*Payment, error) {
	var payment *Payment
	var invoice *Billing
	var (
		doctorID uint
		totalAmount, commissionAmt, gstAmt, tdsAmt, docNet int64
	)

	err := s.repo.ExecuteTransaction(func(txRepo Repository) error {
		var txErr error
		invoice, txErr = txRepo.GetBillingByID(invoiceID)
		if txErr != nil {
			return errors.New("invoice not found")
		}

		if invoice.PaymentStatus == "paid" {
			return errors.New("invoice already paid")
		}

		totalAmount = invoice.Amount
		split := finance.ComputeConsultationSplit(totalAmount)
		commissionAmt = split.Commission
		gstAmt = split.GST
		tdsAmt = split.TDS
		docNet = split.DoctorNet

		newPayment := &Payment{
			InvoiceID:             &invoiceID,
			AppointmentID:         invoice.AppointmentID,
			PatientID:             invoice.PatientID,
			ConsultationFee:       totalAmount,
			PlatformFeePercentage: float64(PlatformCommissionPercent),
			PlatformCommission:    commissionAmt,
			DoctorEarnings:        docNet,
			PaymentMethod:         paymentMethod,
			Status:                "completed",
			PaidAt:                time.Now(),
			RazorpayPaymentID:     "pay_mock_" + time.Now().Format("20060102150405"),
		}

		if err := txRepo.CreatePayment(newPayment); err != nil {
			return err
		}

		invoice.PaymentStatus = "paid"
		invoice.PaymentMethod = paymentMethod

		if err := txRepo.UpdateBilling(invoice); err != nil {
			return err
		}

		payment = newPayment
		return nil
	})

	if err == nil && invoice != nil && invoice.AppointmentID != nil {
		var appt appointment.Appointment
		if txErr := database.DB.First(&appt, *invoice.AppointmentID).Error; txErr == nil && appt.DoctorID != nil {
			doctorID = *appt.DoctorID
		}
	}

	if err == nil && doctorID > 0 {
		ctx := context.Background()
		assetAcct, _ := s.financeService.GetOrCreateAccount(ctx, "Asset:Razorpay", finance.TypeAsset)
		revAcct, _ := s.financeService.GetOrCreateAccount(ctx, "Revenue:Consultations", finance.TypeRevenue)
		gstAcct, _ := s.financeService.GetOrCreateAccount(ctx, "Liability:Tax:GST", finance.TypeLiability)
		tdsAcct, _ := s.financeService.GetOrCreateAccount(ctx, "Liability:Tax:TDS", finance.TypeLiability)
		docAcct, _ := s.financeService.GetOrCreateAccount(ctx, fmt.Sprintf("Doctor:%d", doctorID), finance.TypeLiability)

		entries := []finance.EntryData{
			{AccountID: assetAcct.ID, Direction: finance.DirectionDebit, Amount: totalAmount},
			{AccountID: revAcct.ID, Direction: finance.DirectionCredit, Amount: commissionAmt},
			{AccountID: gstAcct.ID, Direction: finance.DirectionCredit, Amount: gstAmt},
			{AccountID: tdsAcct.ID, Direction: finance.DirectionCredit, Amount: tdsAmt},
			{AccountID: docAcct.ID, Direction: finance.DirectionCredit, Amount: docNet},
		}

		if _, lerr := s.financeService.RecordTransaction(ctx, fmt.Sprintf("INV-%d", invoiceID), "CONSULTATION", "Mock Consultation Payment", entries); lerr != nil {
			// Payment row is already committed as "paid"; a failed ledger write
			// leaves the books inconsistent. Surface it loudly for reconciliation
			// rather than swallowing it.
			log.Printf("[Billing] CRITICAL: invoice %d marked paid but ledger write failed: %v", invoiceID, lerr)
		}
		if payment != nil && payment.AppointmentID != nil {
			go s.triggerFranchiseRoyalty(*payment.AppointmentID, doctorID, payment.ConsultationFee)
		}
	}

	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *service) VerifyPayment(invoiceID uint, razorpayOrderID, razorpayPaymentID, signature string) (*Payment, error) {
	if !razorpay.VerifyPaymentSignature(razorpayOrderID, razorpayPaymentID, signature) {
		return nil, errors.New("payment signature verification failed")
	}

	// Idempotency check: if this razorpay_payment_id was already processed, return the
	// existing record without re-running any finance entries.
	if existing, err := s.repo.GetPaymentByRazorpayID(razorpayPaymentID); err == nil && existing != nil {
		return existing, nil
	}

	var payment *Payment
	var invoice *Billing
	var doctorID uint

	err := s.repo.ExecuteTransaction(func(txRepo Repository) error {
		var txErr error
		invoice, txErr = txRepo.GetBillingByID(invoiceID)
		if txErr != nil {
			return errors.New("invoice not found")
		}

		if invoice.PaymentStatus == "paid" {
			return errors.New("invoice already paid")
		}

		totalAmount := invoice.Amount
		split := finance.ComputeConsultationSplit(totalAmount)

		newPayment := &Payment{
			InvoiceID:             &invoiceID,
			AppointmentID:         invoice.AppointmentID,
			PatientID:             invoice.PatientID,
			ConsultationFee:       totalAmount,
			PlatformFeePercentage: float64(PlatformCommissionPercent),
			PlatformCommission:    split.Commission,
			DoctorEarnings:        split.DoctorNet,
			PaymentMethod:         "razorpay",
			Status:                "completed",
			PaidAt:                time.Now(),
			RazorpayOrderID:       razorpayOrderID,
			RazorpayPaymentID:     razorpayPaymentID,
		}

		if err := txRepo.CreatePayment(newPayment); err != nil {
			return err
		}

		invoice.PaymentStatus = "paid"
		invoice.PaymentMethod = "razorpay"

		if err := txRepo.UpdateBilling(invoice); err != nil {
			return err
		}

		payment = newPayment
		return nil
	})

	if err == nil && invoice != nil && invoice.AppointmentID != nil {
		var appt appointment.Appointment
		if txErr := database.DB.First(&appt, *invoice.AppointmentID).Error; txErr == nil && appt.DoctorID != nil {
			doctorID = *appt.DoctorID
		}
	}

	if err == nil && doctorID > 0 {
		ctx := context.Background()
		assetAcct, _ := s.financeService.GetOrCreateAccount(ctx, "Asset:Razorpay", finance.TypeAsset)
		revAcct, _ := s.financeService.GetOrCreateAccount(ctx, "Revenue:Consultations", finance.TypeRevenue)
		gstAcct, _ := s.financeService.GetOrCreateAccount(ctx, "Liability:Tax:GST", finance.TypeLiability)
		tdsAcct, _ := s.financeService.GetOrCreateAccount(ctx, "Liability:Tax:TDS", finance.TypeLiability)
		docAcct, _ := s.financeService.GetOrCreateAccount(ctx, fmt.Sprintf("Doctor:%d", doctorID), finance.TypeLiability)

		split := finance.ComputeConsultationSplit(invoice.Amount)
		entries := []finance.EntryData{
			{AccountID: assetAcct.ID, Direction: finance.DirectionDebit, Amount: invoice.Amount},
			{AccountID: revAcct.ID, Direction: finance.DirectionCredit, Amount: split.Commission},
			{AccountID: gstAcct.ID, Direction: finance.DirectionCredit, Amount: split.GST},
			{AccountID: tdsAcct.ID, Direction: finance.DirectionCredit, Amount: split.TDS},
			{AccountID: docAcct.ID, Direction: finance.DirectionCredit, Amount: split.DoctorNet},
		}

		if _, lerr := s.financeService.RecordTransaction(ctx, fmt.Sprintf("INV-%d", invoiceID), "CONSULTATION", "Razorpay Consultation Payment", entries); lerr != nil {
			log.Printf("[Billing] CRITICAL: invoice %d marked paid but ledger write failed: %v", invoiceID, lerr)
		}
	}

	if err != nil {
		return nil, err
	}

	// Post-Payment: Franchise Royalty Split
	if payment != nil && payment.AppointmentID != nil && doctorID > 0 {
		go s.triggerFranchiseRoyalty(*payment.AppointmentID, doctorID, payment.ConsultationFee)
	}

	return payment, nil
}

func (s *service) CreateRazorpayOrder(amount int, receipt string) (map[string]interface{}, error) {
	return razorpay.CreateOrder(int64(amount), receipt)
}

// CreateStripeIntent creates a Stripe PaymentIntent in the customer's currency
// for a consultation invoice. The presentment amount is computed server-side
// from the INR invoice amount; the INR price is carried as metadata.
func (s *service) CreateStripeIntent(invoiceID uint, currency string) (map[string]interface{}, error) {
	invoice, err := s.repo.GetBillingByID(invoiceID)
	if err != nil {
		return nil, errors.New("invoice not found")
	}
	if invoice.PaymentStatus == "paid" {
		return nil, errors.New("invoice already paid")
	}
	if currency == "" {
		currency = "USD"
	}
	if !fx.Supported(currency) {
		return nil, errors.New("unsupported presentment currency")
	}
	amountMinor, rate, err := fx.ToPresentment(invoice.Amount, currency)
	if err != nil {
		return nil, err
	}
	return stripe.CreatePaymentIntent(amountMinor, currency, map[string]string{
		"ref":       fmt.Sprintf("INV-%d", invoiceID),
		"inr_paise": fmt.Sprintf("%d", invoice.Amount),
		"fx_rate":   fmt.Sprintf("%.4f", rate),
	})
}

// VerifyStripePayment retrieves the PaymentIntent server-side; only when Stripe
// reports it succeeded do we post the consultation (in INR) to the ledger via
// Asset:Stripe. Idempotent on the PaymentIntent id.
func (s *service) VerifyStripePayment(invoiceID uint, paymentIntentID string) (*Payment, error) {
	pi, err := stripe.RetrievePaymentIntent(paymentIntentID)
	if err != nil {
		return nil, err
	}
	if pi.Status != "succeeded" {
		return nil, errors.New("payment not completed")
	}
	if existing, err := s.repo.GetPaymentByRazorpayID(paymentIntentID); err == nil && existing != nil {
		return existing, nil // idempotent
	}

	var payment *Payment
	var invoice *Billing
	var doctorID uint
	err = s.repo.ExecuteTransaction(func(txRepo Repository) error {
		var txErr error
		invoice, txErr = txRepo.GetBillingByID(invoiceID)
		if txErr != nil {
			return errors.New("invoice not found")
		}
		if invoice.PaymentStatus == "paid" {
			return errors.New("invoice already paid")
		}

		totalAmount := invoice.Amount
		split := finance.ComputeConsultationSplit(totalAmount)

		newPayment := &Payment{
			InvoiceID:             &invoiceID,
			AppointmentID:         invoice.AppointmentID,
			PatientID:             invoice.PatientID,
			ConsultationFee:       totalAmount,
			PlatformFeePercentage: float64(PlatformCommissionPercent),
			PlatformCommission:    split.Commission,
			DoctorEarnings:        split.DoctorNet,
			PaymentMethod:         "stripe",
			Status:                "completed",
			PaidAt:                time.Now(),
			RazorpayPaymentID:     paymentIntentID, // reuse field to store gateway ref
		}
		if err := txRepo.CreatePayment(newPayment); err != nil {
			return err
		}
		invoice.PaymentStatus = "paid"
		invoice.PaymentMethod = "stripe"
		if err := txRepo.UpdateBilling(invoice); err != nil {
			return err
		}

		payment = newPayment
		return nil
	})

	if err == nil && invoice != nil && invoice.AppointmentID != nil {
		var appt appointment.Appointment
		if txErr := database.DB.First(&appt, *invoice.AppointmentID).Error; txErr == nil && appt.DoctorID != nil {
			doctorID = *appt.DoctorID
		}
	}

	if err == nil && doctorID > 0 {
		ctx := context.Background()
		assetAcct, _ := s.financeService.GetOrCreateAccount(ctx, finance.AcctStripe, finance.TypeAsset)
		revAcct, _ := s.financeService.GetOrCreateAccount(ctx, finance.RevConsultations, finance.TypeRevenue)
		gstAcct, _ := s.financeService.GetOrCreateAccount(ctx, finance.LiabGST, finance.TypeLiability)
		tdsAcct, _ := s.financeService.GetOrCreateAccount(ctx, finance.LiabTDS, finance.TypeLiability)
		docAcct, _ := s.financeService.GetOrCreateAccount(ctx, finance.DoctorWallet(doctorID), finance.TypeLiability)

		split := finance.ComputeConsultationSplit(invoice.Amount)
		entries := []finance.EntryData{
			{AccountID: assetAcct.ID, Direction: finance.DirectionDebit, Amount: invoice.Amount},
			{AccountID: revAcct.ID, Direction: finance.DirectionCredit, Amount: split.Commission},
			{AccountID: gstAcct.ID, Direction: finance.DirectionCredit, Amount: split.GST},
			{AccountID: tdsAcct.ID, Direction: finance.DirectionCredit, Amount: split.TDS},
			{AccountID: docAcct.ID, Direction: finance.DirectionCredit, Amount: split.DoctorNet},
		}
		if _, lerr := s.financeService.RecordTransaction(ctx, fmt.Sprintf("INV-%d", invoiceID), "CONSULTATION", "Stripe Consultation Payment", entries); lerr != nil {
			log.Printf("[Billing] CRITICAL: invoice %d marked paid but ledger write failed: %v", invoiceID, lerr)
		}
	}
	if err != nil {
		return nil, err
	}

	if payment != nil && payment.AppointmentID != nil && doctorID > 0 {
		go s.triggerFranchiseRoyalty(*payment.AppointmentID, doctorID, payment.ConsultationFee)
	}

	return payment, nil
}

func (s *service) ProcessCorporatePayment(amount int64, patientID uint) (map[string]interface{}, error) {
	if amount <= 0 {
		return nil, errors.New("amount must be positive")
	}

	ctx := context.Background()
	walletName := fmt.Sprintf("PatientWallet:%d", patientID)
	walletAcct, err := s.financeService.GetOrCreateAccount(ctx, walletName, finance.TypeLiability)
	if err != nil {
		return nil, err
	}
	corpAcct, err := s.financeService.GetOrCreateAccount(ctx, "Asset:CorporateReceivable", finance.TypeAsset)
	if err != nil {
		return nil, err
	}

	// A corporate sponsor funds the patient's wallet. Real double entry:
	// the corporate receivable (asset) grows and the patient wallet (a
	// liability we now owe the patient) grows by the same amount. The previous
	// implementation debited and credited the SAME account — a no-op that moved
	// no money and left the wallet balance unchanged.
	entries := []finance.EntryData{
		{AccountID: corpAcct.ID, Direction: finance.DirectionDebit, Amount: amount},
		{AccountID: walletAcct.ID, Direction: finance.DirectionCredit, Amount: amount},
	}

	if _, err = s.financeService.RecordTransaction(
		ctx,
		fmt.Sprintf("CORP_PAY_%d_%d", patientID, time.Now().Unix()),
		"CORPORATE_PAYMENT",
		fmt.Sprintf("Corporate funding for patient %d", patientID),
		entries,
	); err != nil {
		return nil, err
	}

	// Re-read the wallet so the response reflects the post-top-up balance.
	walletAcct, _ = s.financeService.GetOrCreateAccount(ctx, walletName, finance.TypeLiability)

	return map[string]interface{}{
		"status":            "completed",
		"remaining_balance": float64(walletAcct.Balance) / 100.0,
		"transaction_id":    time.Now().Unix(),
	}, nil
}

func (s *service) triggerFranchiseRoyalty(appointmentID uint, doctorID uint, fee int64) {
	if doctorID == 0 {
		return
	}
	var docUser identity.User
	if err := database.DB.First(&docUser, doctorID).Error; err != nil || docUser.HospitalID == nil {
		return
	}
	var hosp hospital.Hospital
	if err := database.DB.First(&hosp, *docUser.HospitalID).Error; err != nil {
		return
	}
	if hosp.HospitalType == "franchise" && hosp.FranchiseID != nil {
		fs := franchise.NewService()
		if err := fs.ProcessConsultationRoyalty(appointmentID, *hosp.FranchiseID, fee); err != nil {
			log.Printf("[Billing] Failed to process franchise royalty: %v", err)
		}
	}
}
