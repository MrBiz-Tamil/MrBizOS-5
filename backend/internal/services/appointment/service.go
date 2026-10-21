//go:build ignore
// +build ignore

// Deprecated: Migrated to internal/domain/appointment
package appointment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"drayya-core/internal/database"
	"drayya-core/internal/ledger"
	"drayya-core/internal/models"
	"drayya-core/internal/queue"
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
}

type Service struct {
	DB            *gorm.DB
	LedgerService *ledger.Service
}

func NewService() *Service {
	return &Service{
		DB:            database.DB,
		LedgerService: ledger.NewService(),
	}
}

func (s *Service) CreateAppointment(ctx context.Context, req CreateAppointmentRequest) (*models.Appointment, error) {
	// 1. Validations & Pricing
	var amount int64 = 50000 // Default 500 INR in Paise
	var doctorProfile models.DoctorProfile

	// Premium check: Prime tier gets unlimited consultations
	var user models.User
	if err := s.DB.First(&user, req.PatientID).Error; err == nil {
		if user.Tier == "prime" {
			amount = 0
		}
	}

	if amount > 0 && req.DoctorID != nil {
		if err := s.DB.Where("user_id = ?", *req.DoctorID).First(&doctorProfile).Error; err == nil {
			amount = int64(doctorProfile.ConsultationFee * 100)
		}
	}

	// 2. Atomic Transaction: Payment + DB Entry + Enqueue
	var appointment *models.Appointment
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		// a. Payment via Ledger (Simplified for SOS)
		if amount > 0 {
			walletName := fmt.Sprintf("PatientWallet:%d", req.PatientID)
			walletAcct, err := s.LedgerService.GetOrCreateAccount(walletName, models.TypeLiability)
			if err != nil {
				return err
			}

			if walletAcct.Balance < amount {
				return errors.New("insufficient wallet balance")
			}

			heldFunds, _ := s.LedgerService.GetOrCreateAccount("HeldFunds", models.TypeLiability)

			entries := []ledger.EntryData{
				{AccountID: walletAcct.ID, Direction: models.DirectionDebit, Amount: amount},
				{AccountID: heldFunds.ID, Direction: models.DirectionCredit, Amount: amount},
			}

			_, err = s.LedgerService.RecordTransaction(
				uuid.New().String(),
				"CONSULTATION_PAYMENT",
				fmt.Sprintf("Payment for %s consultation", req.Type),
				entries,
			)
			if err != nil {
				return err
			}
		}

		// b. Create Appointment
		appointment = &models.Appointment{
			PatientID: req.PatientID,
			DoctorID:  req.DoctorID,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(30 * time.Minute),
			Status: func() models.AppointmentStatus {
				if req.DoctorID == nil {
					return models.StatusPending
				}
				return models.StatusConfirmed
			}(),
			AppointmentType:  req.Type,
			ConsultationMode: req.Mode,
			Notes:            req.Notes,
			IsPriority:       req.Type == "emergency" || req.Type == "instant",
			FamilyMemberID:   req.FamilyMemberID,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if err := tx.Create(appointment).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 3. Enqueue to Redis Stream (Outside DB Tx)
	err = queue.EnqueueStream(ctx, queue.AppointmentMetadata{
		AppointmentID:   int(appointment.ID),
		PatientID:       int(appointment.PatientID),
		AppointmentType: appointment.AppointmentType,
		AISeverityScore: 0.0, // Default for now
	})

	if err != nil {
		log.Printf("       Failed to enqueue to stream: %v", err)
	}

	return appointment, nil
}

