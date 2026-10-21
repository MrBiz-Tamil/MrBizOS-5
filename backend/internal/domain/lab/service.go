package lab

import (
	"context"
	"errors"
	"fmt"
	"time"

	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/loyalty"
)

// Service defines all business operations for the lab domain.
type Service interface {
	// Catalog
	ListAvailableTests(ctx context.Context, category, concern, q string) ([]LabTest, error)
	ListPackages(ctx context.Context) ([]LabTest, error)
	GetHealthConcerns(ctx context.Context) ([]map[string]interface{}, error)
	GetTestByID(ctx context.Context, id uint) (*LabTest, error)

	// Partner centers
	FindLabCenters(ctx context.Context, pincode, district string, testID uint) ([]labCenterResult, error)

	// Patient bookings
	CreateBooking(ctx context.Context, patientID uint, req BookingRequest) (*LabBooking, error)
	GetMyBookings(ctx context.Context, patientID uint, status string) ([]LabBooking, error)
	GetBookingByID(ctx context.Context, id, patientID uint) (*LabBooking, error)
	CancelBooking(ctx context.Context, id, patientID uint, reason string) error
	ConfirmLabPayment(ctx context.Context, bookingID uint, method, gatewayRef string) (*LabBooking, error)

	// Lab portal
	GetPortalBookings(ctx context.Context, labCenterID uint, status string) ([]LabBooking, error)
	MarkSampleCollected(ctx context.Context, bookingID uint) error
	SubmitResult(ctx context.Context, bookingID uint, req ResultRequest) error
	VerifyReport(ctx context.Context, bookingID uint) error

	// Hospital internal
	OrderTests(ctx context.Context, patientID, doctorID uint, testIDs []uint, priority string) (*LabRequest, error)
	RecordSpecimen(ctx context.Context, requestID uint, specimenType, barcode string, technicianID uint) (*Specimen, error)
	SubmitLabResult(ctx context.Context, requestID, testID uint, value, unit string, technicianID uint) (*LabResult, error)
	GetPatientReports(ctx context.Context, patientID uint) ([]LabRequest, error)
}

// BookingRequest carries the data needed to create a LabBooking.
type BookingRequest struct {
	LabCenterID           uint   `json:"lab_center_id"`
	LabTestID             uint   `json:"lab_test_id"`
	BookingType           string `json:"booking_type"` // home_pickup | lab_visit
	PreferredDate         string `json:"preferred_date"`
	PreferredTimeSlot     string `json:"preferred_time_slot"`
	PickupAddress         string `json:"pickup_address"`
	PickupPincode         string `json:"pickup_pincode"`
	TestPrice             int64  `json:"test_price"`             // paise
	HomePickupCharge      int64  `json:"home_pickup_charge"`     // paise
}

// ResultRequest carries report data submitted by a lab technician.
type ResultRequest struct {
	ReportFileURL    string `json:"report_file_url"`
	AISummary        string `json:"ai_summary"`
	AIRecommendation string `json:"ai_recommendation"`
}

type service struct {
	repo       Repository
	financeSvc finance.Service
}

func NewService(repo Repository, financeSvc finance.Service) Service {
	return &service{repo: repo, financeSvc: financeSvc}
}

// ── Catalog ───────────────────────────────────────────────────────────────────

func (s *service) ListAvailableTests(ctx context.Context, category, concern, q string) ([]LabTest, error) {
	return s.repo.ListLabTests(ctx, category, concern, q)
}

func (s *service) ListPackages(ctx context.Context) ([]LabTest, error) {
	return s.repo.ListPackages(ctx)
}

func (s *service) GetHealthConcerns(ctx context.Context) ([]map[string]interface{}, error) {
	return s.repo.GetHealthConcerns(ctx)
}

func (s *service) GetTestByID(ctx context.Context, id uint) (*LabTest, error) {
	return s.repo.GetLabTestByID(ctx, id)
}

// ── Partner centers ───────────────────────────────────────────────────────────

func (s *service) FindLabCenters(ctx context.Context, pincode, district string, testID uint) ([]labCenterResult, error) {
	if pincode == "" && district == "" {
		return nil, errors.New("pincode or district is required")
	}
	return s.repo.FindLabCenters(ctx, pincode, district, testID)
}

// ── Patient bookings ──────────────────────────────────────────────────────────

func (s *service) CreateBooking(ctx context.Context, patientID uint, req BookingRequest) (*LabBooking, error) {
	if req.LabCenterID == 0 {
		return nil, errors.New("lab_center_id is required")
	}
	if req.LabTestID == 0 {
		return nil, errors.New("lab_test_id is required")
	}
	if req.PreferredDate == "" {
		return nil, errors.New("preferred_date is required")
	}
	if req.BookingType == "" {
		req.BookingType = "home_pickup"
	}
	if req.BookingType == "home_pickup" && req.PickupAddress == "" {
		return nil, errors.New("pickup_address is required for home pickup")
	}

	// SECURITY: the price is resolved server-side from the center's pricing (or
	// the test catalog as fallback). Client-supplied amounts in the request are
	// IGNORED — otherwise a patient could book any test at a price they choose.
	var testPrice, pickupCharge int64
	if pricing, err := s.repo.GetCenterPricing(ctx, req.LabCenterID, req.LabTestID); err == nil && pricing != nil && pricing.PricePaise > 0 {
		testPrice = pricing.PricePaise
		pickupCharge = pricing.HomePickupChargePaise
	} else {
		test, terr := s.repo.GetLabTestByID(ctx, req.LabTestID)
		if terr != nil || test == nil || test.BasePrice <= 0 {
			return nil, errors.New("price unavailable for this test/center")
		}
		testPrice = test.BasePrice
	}

	total := testPrice
	if req.BookingType == "home_pickup" {
		total += pickupCharge
	}

	booking := &LabBooking{
		PatientID:             patientID,
		LabCenterID:           req.LabCenterID,
		LabTestID:             req.LabTestID,
		BookingType:           req.BookingType,
		Status:                "pending",
		PreferredDate:         req.PreferredDate,
		PreferredTimeSlot:     req.PreferredTimeSlot,
		PickupAddress:         req.PickupAddress,
		PickupPincode:         req.PickupPincode,
		AmountPaise:           testPrice,
		HomePickupChargePaise: pickupCharge,
		TotalPaise:            total,
		PaymentStatus:         "pending",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	if err := s.repo.CreateBooking(ctx, booking); err != nil {
		return nil, err
	}
	return booking, nil
}

func (s *service) GetMyBookings(ctx context.Context, patientID uint, status string) ([]LabBooking, error) {
	return s.repo.GetBookingsByPatient(ctx, patientID, status)
}

func (s *service) GetBookingByID(ctx context.Context, id, patientID uint) (*LabBooking, error) {
	b, err := s.repo.GetBookingByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Only the owning patient can view their booking
	if patientID != 0 && b.PatientID != patientID {
		return nil, errors.New("booking not found")
	}
	return b, nil
}

func (s *service) CancelBooking(ctx context.Context, id, patientID uint, reason string) error {
	b, err := s.repo.GetBookingByID(ctx, id)
	if err != nil {
		return err
	}
	if b.PatientID != patientID {
		return errors.New("booking not found")
	}
	if b.Status != "pending" && b.Status != "confirmed" {
		return fmt.Errorf("cannot cancel a booking with status '%s'", b.Status)
	}
	return s.repo.CancelBooking(ctx, id, reason)
}

// ConfirmLabPayment posts a paid booking to the central ledger using the
// marketplace split (platform commission + GST + lab-partner payable) and marks
// the booking paid. Idempotent: a booking already paid (or a duplicate
// PostRevenue reference) does not double-post. method is "wallet" or "razorpay".
func (s *service) ConfirmLabPayment(ctx context.Context, bookingID uint, method, gatewayRef string) (*LabBooking, error) {
	b, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if b.PaymentStatus == "paid" {
		return b, nil // already settled
	}
	if b.TotalPaise <= 0 {
		return nil, errors.New("booking has no payable amount")
	}
	if b.LabCenter == nil || b.LabCenter.PartnerID == 0 {
		return nil, errors.New("cannot resolve lab partner for booking")
	}
	partnerID := b.LabCenter.PartnerID

	// Marketplace split (GST inclusive of total; commission off the net).
	total := b.TotalPaise
	gst := total * finance.LabGSTPercent / (100 + finance.LabGSTPercent)
	net := total - gst
	commission := net * finance.LabCommissionPct / 100
	provider := net - commission

	// Funding source. The ledger always books the INR price; for Stripe the
	// customer was charged in their currency but Stripe settles to our INR bank,
	// so the INR amount lands in Asset:Stripe.
	fundingAccount := finance.AcctRazorpay
	fundingType := finance.TypeAsset
	switch method {
	case "wallet":
		fundingAccount = finance.PatientWallet(b.PatientID)
		fundingType = finance.TypeLiability
		acct, err := s.financeSvc.GetOrCreateAccount(ctx, fundingAccount, finance.TypeLiability)
		if err != nil {
			return nil, err
		}
		if acct.Balance < total {
			return nil, fmt.Errorf("insufficient wallet balance: have %d paise, need %d", acct.Balance, total)
		}
	case "stripe":
		fundingAccount = finance.AcctStripe
	}

	if _, err := s.financeSvc.PostRevenue(ctx, fmt.Sprintf("LAB-%d", b.ID), "LAB",
		fmt.Sprintf("Lab booking #%d payment", b.ID),
		finance.RevenueInput{
			FundingAccount: fundingAccount,
			FundingType:    fundingType,
			Total:          total,
			Credits: []finance.RevenueLeg{
				{Account: finance.RevLab, Type: finance.TypeRevenue, Amount: commission},
				{Account: finance.LiabGST, Type: finance.TypeLiability, Amount: gst},
				{Account: finance.ProviderWallet("Lab", partnerID), Type: finance.TypeLiability, Amount: provider},
			},
		}); err != nil {
		return nil, fmt.Errorf("ledger posting failed: %w", err)
	}

	if err := s.repo.MarkBookingPaid(ctx, b.ID, gatewayRef); err != nil {
		return nil, err
	}

	// Award loyalty points (best-effort; never blocks payment).
	_, _ = loyalty.EarnPoints(b.PatientID, total, "lab", b.ID)

	b.PaymentStatus = "paid"
	b.Status = "confirmed"
	b.PaymentRef = gatewayRef
	return b, nil
}

// ── Lab portal ────────────────────────────────────────────────────────────────

func (s *service) GetPortalBookings(ctx context.Context, labCenterID uint, status string) ([]LabBooking, error) {
	return s.repo.GetPortalBookings(ctx, labCenterID, status)
}

func (s *service) MarkSampleCollected(ctx context.Context, bookingID uint) error {
	return s.repo.MarkSampleCollected(ctx, bookingID)
}

func (s *service) SubmitResult(ctx context.Context, bookingID uint, req ResultRequest) error {
	return s.repo.SubmitBookingResult(ctx, bookingID, req.ReportFileURL, req.AISummary, req.AIRecommendation)
}

func (s *service) VerifyReport(ctx context.Context, bookingID uint) error {
	return s.repo.VerifyBookingReport(ctx, bookingID)
}

// ── Hospital internal ─────────────────────────────────────────────────────────

func (s *service) OrderTests(ctx context.Context, patientID, doctorID uint, testIDs []uint, priority string) (*LabRequest, error) {
	var tests []LabTest
	for _, tid := range testIDs {
		t, err := s.repo.GetLabTestByID(ctx, tid)
		if err != nil {
			return nil, fmt.Errorf("lab test %d not found", tid)
		}
		tests = append(tests, *t)
	}
	lr := &LabRequest{
		PatientID: patientID,
		DoctorID:  doctorID,
		Status:    "pending",
		Priority:  priority,
		OrderedAt: time.Now(),
		Tests:     tests,
	}
	if err := s.repo.CreateLabRequest(ctx, lr); err != nil {
		return nil, err
	}
	return lr, nil
}

func (s *service) RecordSpecimen(ctx context.Context, requestID uint, specimenType, barcode string, technicianID uint) (*Specimen, error) {
	sp := &Specimen{
		RequestID:     requestID,
		Type:          specimenType,
		Barcode:       barcode,
		Status:        "collected",
		CollectedAt:   time.Now(),
		CollectedByID: technicianID,
	}
	if err := s.repo.RecordSpecimenCollection(ctx, sp); err != nil {
		return nil, err
	}
	s.repo.UpdateLabRequestStatus(ctx, requestID, "collected")
	return sp, nil
}

func (s *service) SubmitLabResult(ctx context.Context, requestID, testID uint, value, unit string, technicianID uint) (*LabResult, error) {
	res := &LabResult{
		RequestID:    requestID,
		TestID:       testID,
		Value:        value,
		Unit:         unit,
		TechnicianID: technicianID,
		CreatedAt:    time.Now(),
	}
	if err := s.repo.CreateLabResult(ctx, res); err != nil {
		return nil, err
	}
	s.repo.UpdateLabRequestStatus(ctx, requestID, "completed")
	return res, nil
}

func (s *service) GetPatientReports(ctx context.Context, patientID uint) ([]LabRequest, error) {
	return s.repo.GetLabRequestsByPatient(ctx, patientID)
}
