package appointment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/presence"
	"drayya-core/internal/queue"
	"drayya-core/internal/services/financials"
	"drayya-core/internal/utils/config"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreateAppointmentRequest struct {
	PatientID      uint
	DoctorID       *uint
	Type           string // emergency, instant, consultation
	Mode           string // chat, video
	Notes          string
	FamilyMemberID *uint
	// Extra fields used by the enhanced endpoint
	StartTime string
	EndTime   string
}

type Service interface {
	CreateAppointment(ctx context.Context, req CreateAppointmentRequest) (*Appointment, error)
	GetAppointmentByID(id string) (*Appointment, error)
	UpdateAppointment(id string, updates map[string]interface{}) error
	RescheduleAppointment(id string, startTime, endTime string) error
	CancelAppointment(id string, reason string) error
	CompleteAppointment(ctx context.Context, id string) error
	StartAppointment(id string) error
	MarkNoShow(id string) error
	AddNotes(id string, notes string) error
	UpdateDoctorStatus(id string, status AppointmentStatus) error

	GetUpcomingAppointments(userID uint) ([]Appointment, error)
	GetPastAppointments(userID uint) ([]Appointment, error)
	GetUpcomingAppointmentsForDoctor(doctorID uint) ([]Appointment, error)
	GetAppointmentHistory(patientID, doctorID *uint) ([]Appointment, error)
	GetInstantPendingAppointments() ([]Appointment, error)
	// GetInstantQueueForDoctor is the doctor-aware variant of GetInstantPendingAppointments.
	// It returns unassigned pending requests + any confirmed ones assigned to this doctor
	// so the WaitingCallDashboard stays populated after dispatch pre-confirms the appointment.
	GetInstantQueueForDoctor(doctorID uint) ([]Appointment, error)
	AcceptAppointment(id string, doctorID uint) error
	DeclineAppointment(appointmentID uint, doctorUserID uint, reason string) error
}

type service struct {
	Repo           Repository
	FinanceService finance.Service
}

func NewService(repo Repository, financeService finance.Service) Service {
	return &service{
		Repo:           repo,
		FinanceService: financeService,
	}
}

func (s *service) CreateAppointment(ctx context.Context, req CreateAppointmentRequest) (*Appointment, error) {
	var appointment *Appointment

	err := s.Repo.ExecuteTransaction(ctx, func(txCtx context.Context, txRepo Repository, tx *gorm.DB) error {
		patient, err := txRepo.GetPatientInfo(req.PatientID)
		if err != nil {
			return errors.New("patient not found")
		}

		var finalFee int64 = 50000 // Default 500 INR in Paise
		isSubsidized := false

		// Premium check: Prime tier gets unlimited consultations
		if patient.Tier == "prime" {
			finalFee = 0
		}

		// Verify Doctor and Process Details
		if req.DoctorID != nil {
			doctorProfile, err := txRepo.GetDoctorProfile(*req.DoctorID)
			if err != nil {
				return errors.New("doctor not found")
			}
			// NOTE: is_verified is a display badge on this platform, not a booking gate.
			// Patients can book unverified doctors; the badge signals extra trust, not access.

			// Use doctor's actual fee (stored as paise — do NOT multiply by 100 again).
			// Default finalFee (50000 paise = ₹500) is only used when no doctor is assigned.
			if finalFee > 0 {
				if fee, ok := doctorProfile.ConsultationFees["INR"]; ok {
					finalFee = fee
				}
			}

			// Hybrid Free Logic & Growth Engine
			// Configurable limits: dailyFreeLimit (per doctor) and patientFreeLimit (per user)
			dailyLimit := int64(config.GetGrowthSetting("dailyFreeLimit", float64(2)).(float64))
			patientLimit := int64(config.GetGrowthSetting("patientFreeLimit", float64(1)).(float64))

			if int64(patient.FreeConsultsUsed) < patientLimit && doctorProfile.FreeConsultsGivenToday < int(dailyLimit) {
				// This is a subsidized growth consultation
				finalFee = 0
				isSubsidized = true

				if err := txRepo.IncrementPatientFreeConsult(req.PatientID); err != nil {
					return err
				}
				if err := txRepo.IncrementDoctorFreeConsultGiven(*req.DoctorID); err != nil {
					return err
				}
			}
		}

		// Handle Payment & Subsidy
		if finalFee > 0 || isSubsidized {
			heldFunds, _ := s.FinanceService.GetOrCreateAccount(txCtx, "HeldFunds", finance.TypeLiability)
			var entries []finance.EntryData
			var totalCharge int64
			var gstAmount int64
			var memo string

			if isSubsidized {
				// Subsidized by Platform Marketing Pool
				subsidyAmount := int64(config.GetGrowthSetting("subsidyAmount", float64(40000)).(float64))
				marketingAcct, _ := s.FinanceService.GetOrCreateAccount(txCtx, "Marketing:GrowthSubsidy", finance.TypeExpense)

				entries = append(entries, finance.EntryData{AccountID: marketingAcct.ID, Direction: finance.DirectionDebit, Amount: subsidyAmount})
				entries = append(entries, finance.EntryData{AccountID: heldFunds.ID, Direction: finance.DirectionCredit, Amount: subsidyAmount})

				finalFee = subsidyAmount // For the appointment record
				memo = "Subsidized Growth Consultation (First-time user)"
			} else {
				// Standard Patient Payment
				walletName := fmt.Sprintf("PatientWallet:%d", req.PatientID)
				walletAcct, err := s.FinanceService.GetOrCreateAccount(txCtx, walletName, finance.TypeLiability)
				if err != nil {
					return err
				}

				gstAmount, _ = s.FinanceService.CalculateTax(txCtx, finalFee, "IN", "consultation")
				totalCharge = finalFee + gstAmount

				if walletAcct.Balance < totalCharge {
					return errors.New("insufficient wallet balance")
				}

				entries = append(entries, finance.EntryData{AccountID: walletAcct.ID, Direction: finance.DirectionDebit, Amount: totalCharge})
				entries = append(entries, finance.EntryData{AccountID: heldFunds.ID, Direction: finance.DirectionCredit, Amount: finalFee})

				if gstAmount > 0 {
					gstAcct, _ := s.FinanceService.GetOrCreateAccount(txCtx, "Liability:GST", finance.TypeLiability)
					entries = append(entries, finance.EntryData{AccountID: gstAcct.ID, Direction: finance.DirectionCredit, Amount: gstAmount})
				}
				memo = fmt.Sprintf("Payment for %s consultation (fee ₹%.2f + GST ₹%.2f)", req.Type, float64(finalFee)/100, float64(gstAmount)/100)
			}

			_, err = s.FinanceService.RecordTransaction(
				txCtx,
				uuid.New().String(),
				"CONSULTATION_PAYMENT",
				memo,
				entries,
			)
			if err != nil {
				return err
			}
		}

		// Create Appointment
		startTime := time.Now()
		if req.StartTime != "" {
			parsed, err := time.Parse(time.RFC3339, req.StartTime)
			if err != nil || parsed.IsZero() {
				return errors.New("invalid start_time: must be RFC3339")
			}
			startTime = parsed
		}

		endTime := startTime.Add(30 * time.Minute)
		if req.EndTime != "" {
			parsed, err := time.Parse(time.RFC3339, req.EndTime)
			if err != nil || parsed.IsZero() {
				return errors.New("invalid end_time: must be RFC3339")
			}
			endTime = parsed
		}
		if !endTime.After(startTime) {
			return errors.New("invalid appointment time range")
		}

		var status AppointmentStatus = StatusConfirmed
		if req.DoctorID == nil {
			status = StatusPending
		}

		appointment = &Appointment{
			PatientID:        req.PatientID,
			DoctorID:         req.DoctorID,
			StartTime:        startTime,
			EndTime:          endTime,
			Status:           status,
			AppointmentType:  req.Type,
			ConsultationMode: req.Mode,
			Notes:            req.Notes,
			ConsultationFee:  finalFee,
			IsPriority:       req.Type == "emergency" || req.Type == "instant",
			FamilyMemberID:   req.FamilyMemberID,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		return txRepo.CreateAppointment(appointment)
	})

	if err != nil {
		log.Printf("    Appointment Creation Failed: %v", err)
		return nil, err
	}

	// Patient's preferred language — feeds the dispatch language-match boost.
	var patientLang string
	database.DB.Table("user").Where("id = ?", appointment.PatientID).
		Select("preferred_language").Scan(&patientLang)

	// Enqueue to Redis Stream
	err = queue.EnqueueStream(ctx, queue.AppointmentMetadata{
		AppointmentID:   int(appointment.ID),
		PatientID:       int(appointment.PatientID),
		AppointmentType: appointment.AppointmentType,
		Language:        patientLang,
		AISeverityScore: 0.0,
	})

	if err != nil {
		log.Printf("       Failed to enqueue to stream: %v", err)
	}

	return appointment, nil
}

func (s *service) GetAppointmentByID(id string) (*Appointment, error) {
	return s.Repo.GetAppointmentByID(id)
}

func (s *service) UpdateAppointment(id string, updates map[string]interface{}) error {
	rows, err := s.Repo.UpdateAppointment(id, updates)
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("appointment not found")
	}
	return nil
}

func (s *service) RescheduleAppointment(id string, startTime, endTime string) error {
	appt, err := s.Repo.GetAppointmentByID(id)
	if err != nil {
		return errors.New("appointment not found")
	}

	// Only pending or confirmed appointments may be rescheduled.
	if appt.Status != StatusPending && appt.Status != StatusConfirmed {
		return fmt.Errorf("cannot reschedule appointment in status '%s': must be pending or confirmed", appt.Status)
	}

	newStart, err := time.Parse(time.RFC3339, startTime)
	if err != nil || newStart.IsZero() {
		return errors.New("invalid start_time: must be RFC3339")
	}

	// New time must be at least 30 minutes in the future.
	if time.Until(newStart) < 30*time.Minute {
		return errors.New("new scheduled time must be at least 30 minutes in the future")
	}

	newEnd := newStart.Add(30 * time.Minute)
	if endTime != "" {
		parsed, perr := time.Parse(time.RFC3339, endTime)
		if perr != nil || parsed.IsZero() {
			return errors.New("invalid end_time: must be RFC3339")
		}
		newEnd = parsed
	}
	if !newEnd.After(newStart) {
		return errors.New("end_time must be after start_time")
	}

	// Check doctor availability for the new slot (excluding this appointment itself).
	if appt.DoctorID != nil {
		available, avErr := s.Repo.CheckDoctorAvailability(*appt.DoctorID, newStart, appt.ID)
		if avErr != nil {
			return fmt.Errorf("availability check failed: %w", avErr)
		}
		if !available {
			return errors.New("doctor is not available at the requested time (overlapping appointment within 30 minutes)")
		}
	}

	updates := map[string]interface{}{
		"start_time": newStart,
		"end_time":   newEnd,
		"status":     string(StatusRescheduled),
	}
	rows, err := s.Repo.UpdateAppointmentAtomic(id, string(appt.Status), updates)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("conflict: appointment state changed by another request, please retry")
	}
	return nil
}

func (s *service) CancelAppointment(id string, reason string) error {
	appt, err := s.Repo.GetAppointmentByID(id)
	if err != nil {
		return errors.New("appointment not found")
	}

	// State machine guard: cannot cancel terminal states.
	switch appt.Status {
	case StatusCompleted, StatusNoShow, StatusCancelled, StatusDeclined:
		return fmt.Errorf("cannot cancel an appointment in status: %s", appt.Status)
	}

	// Time-based cancellation penalty on paid appointments.
	// >24h before start  → 100% refund
	// 2-24h before start → 50% refund (50% penalty → Revenue:Cancellations)
	// <2h before start   → 0% refund (100% penalty)
	if appt.ConsultationFee > 0 && (appt.Status == StatusPending || appt.Status == StatusConfirmed) {
		ctx := context.Background()

		timeToAppt := time.Until(appt.StartTime)
		var refundAmount int64
		switch {
		case timeToAppt > 24*time.Hour:
			refundAmount = appt.ConsultationFee
		case timeToAppt > 2*time.Hour:
			refundAmount = appt.ConsultationFee / 2
		default:
			refundAmount = 0
		}
		penaltyAmount := appt.ConsultationFee - refundAmount

		heldFunds, herr := s.FinanceService.GetOrCreateAccount(ctx, "HeldFunds", finance.TypeLiability)
		if herr == nil {
			var entries []finance.EntryData
			entries = append(entries, finance.EntryData{
				AccountID: heldFunds.ID,
				Direction: finance.DirectionDebit,
				Amount:    appt.ConsultationFee,
			})

			if refundAmount > 0 {
				walletName := fmt.Sprintf("PatientWallet:%d", appt.PatientID)
				walletAcct, werr := s.FinanceService.GetOrCreateAccount(ctx, walletName, finance.TypeLiability)
				if werr == nil {
					entries = append(entries, finance.EntryData{
						AccountID: walletAcct.ID,
						Direction: finance.DirectionCredit,
						Amount:    refundAmount,
					})
				}
			}

			if penaltyAmount > 0 {
				cancRevAcct, cerr := s.FinanceService.GetOrCreateAccount(ctx, "Revenue:Cancellations", finance.TypeRevenue)
				if cerr == nil {
					entries = append(entries, finance.EntryData{
						AccountID: cancRevAcct.ID,
						Direction: finance.DirectionCredit,
						Amount:    penaltyAmount,
					})
				}
			}

			_, rerr := s.FinanceService.RecordTransaction(ctx, uuid.New().String(), "CONSULTATION_REFUND",
				fmt.Sprintf("Cancellation refund ₹%.2f / penalty ₹%.2f for appt #%s (%.0fh notice)",
					float64(refundAmount)/100, float64(penaltyAmount)/100, id, timeToAppt.Hours()),
				entries)
			if rerr != nil {
				log.Printf("    Refund failed for appointment %s: %v", id, rerr)
			}
		}
	}

	updates := map[string]interface{}{
		"status":              "cancelled",
		"cancellation_reason": reason,
	}
	rows, err2 := s.Repo.UpdateAppointmentAtomic(id, string(appt.Status), updates)
	if err2 != nil {
		return err2
	}
	if rows == 0 {
		return fmt.Errorf("conflict: appointment state changed by another request, please retry")
	}
	return nil
}

func (s *service) CompleteAppointment(ctx context.Context, id string) error {
	appt, err := s.Repo.GetAppointmentByID(id)
	if err != nil {
		return errors.New("appointment not found")
	}
	if appt.Status != StatusInConsult {
		return fmt.Errorf("cannot complete appointment with status '%s': must be in_consultation", appt.Status)
	}

	rows, err := s.Repo.UpdateAppointmentAtomic(id, string(StatusInConsult), map[string]interface{}{"status": string(StatusCompleted)})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("conflict: appointment state changed by another request, please retry")
	}

	if appt.DoctorID != nil {
		presence.DecrementSession(ctx, fmt.Sprintf("%d", *appt.DoctorID))
		queue.FinishAppointment(ctx, int(appt.ID))

		// Auto-increment consultation count on the doctor's profile.
		if err := s.Repo.IncrementDoctorConsultations(*appt.DoctorID); err != nil {
			log.Printf("       ConsultCount increment failed for doctor %d: %v", *appt.DoctorID, err)
		}

		errPayout := financials.ProcessConsultationPayout(
			ctx,
			appt.ID,
			*appt.DoctorID,
			appt.PatientID,
			appt.ConsultationFee,
		)
		if errPayout != nil {
			log.Printf("       Payout Error for Appointment %d: %v", appt.ID, errPayout)
		}
	}
	return nil
}

func (s *service) StartAppointment(id string) error {
	appt, err := s.Repo.GetAppointmentByID(id)
	if err != nil {
		return errors.New("appointment not found")
	}
	// Idempotent: already in_consultation
	if appt.Status == StatusInConsult {
		return nil
	}
	if appt.Status != StatusConfirmed {
		return fmt.Errorf("cannot start appointment with status '%s': must be confirmed", appt.Status)
	}

	// Time-window validation: allow starting up to 10 minutes early, block if more than 30 minutes late.
	now := time.Now()
	earliestStart := appt.StartTime.Add(-10 * time.Minute)
	latestStart := appt.StartTime.Add(30 * time.Minute)
	if now.Before(earliestStart) || now.After(latestStart) {
		return errors.New("appointment cannot be started outside the scheduled time window")
	}

	rows, err := s.Repo.UpdateAppointmentAtomic(id, string(StatusConfirmed), map[string]interface{}{"status": string(StatusInConsult)})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("conflict: appointment state changed by another request, please retry")
	}
	return nil
}

func (s *service) MarkNoShow(id string) error {
	appt, err := s.Repo.GetAppointmentByID(id)
	if err != nil {
		return errors.New("appointment not found")
	}
	if appt.Status != StatusConfirmed && appt.Status != StatusPending {
		return fmt.Errorf("cannot mark no-show for appointment with status '%s'", appt.Status)
	}
	// Only allow no-show if the scheduled start time has passed
	if time.Now().Before(appt.StartTime) {
		return errors.New("cannot mark no-show: appointment has not started yet")
	}

	rows, err := s.Repo.UpdateAppointmentAtomic(id, string(appt.Status), map[string]interface{}{"status": string(StatusNoShow)})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("conflict: appointment state changed by another request, please retry")
	}
	return nil
}

func (s *service) AddNotes(id string, notes string) error {
	return s.UpdateAppointment(id, map[string]interface{}{"notes": notes})
}

func (s *service) UpdateDoctorStatus(id string, status AppointmentStatus) error {
	// e.g. doctor_en_route, doctor_arrived
	rows, err := s.Repo.UpdateAppointmentStatus(id, string(status))
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("appointment not found")
	}
	return nil
}

func (s *service) GetUpcomingAppointments(userID uint) ([]Appointment, error) {
	return s.Repo.GetAppointmentsByPatientOrDoctor(userID, false, 0)
}

func (s *service) GetPastAppointments(userID uint) ([]Appointment, error) {
	return s.Repo.GetAppointmentsByPatientOrDoctor(userID, true, 50)
}

func (s *service) GetUpcomingAppointmentsForDoctor(doctorID uint) ([]Appointment, error) {
	return s.Repo.GetUpcomingAppointmentsForDoctor(doctorID)
}

func (s *service) GetAppointmentHistory(patientID, doctorID *uint) ([]Appointment, error) {
	return s.Repo.GetAppointmentHistoryCombo(patientID, doctorID)
}

func (s *service) GetInstantPendingAppointments() ([]Appointment, error) {
	return s.Repo.GetPendingInstantAppointments()
}

func (s *service) GetInstantQueueForDoctor(doctorID uint) ([]Appointment, error) {
	return s.Repo.GetInstantQueueForDoctor(doctorID)
}

// DeclineAppointment allows the assigned doctor to decline a pending appointment.
func (s *service) DeclineAppointment(appointmentID uint, doctorUserID uint, reason string) error {
	appt, err := s.Repo.GetAppointmentByIDUint(appointmentID)
	if err != nil {
		return errors.New("appointment not found")
	}

	// Only the assigned doctor may decline.
	if appt.DoctorID == nil || *appt.DoctorID != doctorUserID {
		return errors.New("unauthorized: you are not the assigned doctor for this appointment")
	}

	// Only pending appointments can be declined.
	if appt.Status != StatusPending {
		return fmt.Errorf("cannot decline appointment in status '%s': must be pending", appt.Status)
	}

	// Store reason in notes (the model has no dedicated decline_reason column).
	notesSuffix := ""
	if reason != "" {
		notesSuffix = fmt.Sprintf("\n[Declined by doctor: %s]", reason)
	} else {
		notesSuffix = "\n[Declined by doctor]"
	}

	updates := map[string]interface{}{
		"status": string(StatusDeclined),
		"notes":  appt.Notes + notesSuffix,
	}
	id := fmt.Sprintf("%d", appointmentID)
	rows, err := s.Repo.UpdateAppointmentAtomic(id, string(StatusPending), updates)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("conflict: appointment state changed by another request, please retry")
	}
	return nil
}

func (s *service) AcceptAppointment(id string, doctorID uint) error {
	appt, err := s.Repo.GetAppointmentByID(id)
	if err != nil {
		return errors.New("appointment not found")
	}

	// Allow accepting:
	// 1. Pending appointments (doctor self-assigns from the queue)
	// 2. Confirmed appointments that dispatch pre-assigned to THIS doctor
	if appt.Status == StatusConfirmed {
		// Dispatch pre-confirmed: only the assigned doctor may accept
		if appt.DoctorID != nil && *appt.DoctorID != doctorID {
			return errors.New("unauthorized: you are not the assigned doctor for this appointment")
		}
	} else if appt.Status != StatusPending {
		return fmt.Errorf("appointment cannot be accepted in status '%s'", appt.Status)
	}

	updates := map[string]interface{}{
		"status":    string(StatusConfirmed),
		"doctor_id": doctorID,
	}

	// Always use atomic update to prevent race conditions.
	// For pending appointments this prevents two doctors from accepting simultaneously.
	// For pre-confirmed appointments we still check the current status atomically.
	currentStatus := string(appt.Status)
	rows, err := s.Repo.UpdateAppointmentAtomic(id, currentStatus, updates)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("conflict: appointment is no longer available (already accepted or cancelled)")
	}
	return nil
}
