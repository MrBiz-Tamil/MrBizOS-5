package franchise

import (
	"context"
	"errors"
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/franchise"
	"drayya-core/internal/domain/hospital"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PayoutService struct {
	DB             *gorm.DB
	FinanceService finance.Service
}

func NewPayoutService() *PayoutService {
	return &PayoutService{
		DB:             database.DB,
		FinanceService: finance.NewService(finance.NewRepository(database.DB)),
	}
}

// ProcessAppointmentPayout executes the revenue split when a franchise appointment is completed.
func (s *PayoutService) ProcessAppointmentPayout(appointmentID uint, totalAmount int64) error {
	var appointment appointment.Appointment
	// Fetch appointment with doctor profile and hospital to check if it's a franchise
	if err := s.DB.Preload("Doctor").Preload("Doctor.DoctorProfile").First(&appointment, appointmentID).Error; err != nil {
		return err
	}

	// For franchise logic, we assume the doctor is linked to a hospital (this might require joining through a schedule or direct link)
	// For this enterprise implementation, we'll fetch the hospital if available.
	// Note: In an actual robust setup, an appointment would explicitly link to a HospitalID/ClinicID where it occurred.
	// For now, let's assume we find the franchise via the Doctor or a direct HospitalID link if added later.
	// Let's query if this doctor belongs to a franchise hospital.
	var hospitalObj hospital.Hospital
	// Mock logic: assuming we have a way to link appointment -> hospital.
	// Since we added FranchiseID to Hospital, we need to know WHICH hospital this happened at.
	// If the system doesn't directly link appointment to hospital yet, we'll return nil for now
	// or log a warning that the link is missing.

	// Assuming DoctorProfile has a PrimaryHospitalID or similar, or we just look up the first clinic they belong to (for demo purposes)
	err := s.DB.Joins("JOIN hospital_doctors ON hospital_doctors.hospital_id = hospitals.id").
		Where("hospital_doctors.doctor_id = ? AND hospitals.hospital_type = ?", appointment.DoctorID, "franchise").
		Preload("Franchise").
		First(&hospitalObj).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Not a franchise appointment, normal payout applies (100% to Master Revenue or Doctor Split)
			return s.processStandardPayout(appointment.PatientID, *appointment.DoctorID, totalAmount)
		}
		return err
	}

	if hospitalObj.Franchise == nil {
		return errors.New("franchise data missing for franchise hospital")
	}

	// --- Calculate Royalty Split ---
	royaltyRateInt := int64(hospitalObj.Franchise.RoyaltyRate) // percentage as integer
	royaltyPaise := (totalAmount * royaltyRateInt) / 100
	franchisePaise := totalAmount - royaltyPaise
	totalPaise := totalAmount

	return s.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Ledger Entries
		heldFundsAcct, err := s.FinanceService.GetOrCreateAccount(context.Background(), "HeldFunds", finance.TypeLiability)
		if err != nil {
			return err
		}

		masterRevenueAcct, err := s.FinanceService.GetOrCreateAccount(context.Background(), "MasterRevenue", finance.TypeRevenue)
		if err != nil {
			return err
		}

		franchiseWalletName := fmt.Sprintf("FranchiseWallet:%d", hospitalObj.Franchise.OwnerID)
		franchiseWalletAcct, err := s.FinanceService.GetOrCreateAccount(context.Background(), franchiseWalletName, finance.TypeLiability)
		if err != nil {
			return err
		}

		entries := []finance.EntryData{
			{AccountID: heldFundsAcct.ID, Direction: finance.DirectionDebit, Amount: totalPaise},            // Release from escrow
			{AccountID: masterRevenueAcct.ID, Direction: finance.DirectionCredit, Amount: royaltyPaise},     // Dr.Ayya Cut
			{AccountID: franchiseWalletAcct.ID, Direction: finance.DirectionCredit, Amount: franchisePaise}, // Franchise Cut
		}

		txnID := uuid.New().String()
		_, err = s.FinanceService.RecordTransaction(
			context.Background(),
			txnID,
			"FRANCHISE_PAYOUT",
			fmt.Sprintf("Payout for appointment %d at franchise %s", appointmentID, hospitalObj.Name),
			entries,
		)
		if err != nil {
			return err
		}

		// 2. Record Royalty Log
		logEntry := franchise.FranchiseRoyaltyLog{
			FranchiseID:   hospitalObj.Franchise.ID,
			AppointmentID: appointmentID,
			TotalAmount:   totalAmount,
			RoyaltyAmount: royaltyPaise,
			TransactionID: txnID,
		}

		if err := tx.Create(&logEntry).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *PayoutService) processStandardPayout(patientID, doctorID uint, amount int64) error {
	// Simple standard payout: from HeldFunds to Master Revenue
	heldFundsAcct, err := s.FinanceService.GetOrCreateAccount(context.Background(), "HeldFunds", finance.TypeLiability)
	if err != nil {
		return err
	}

	masterRevenueAcct, err := s.FinanceService.GetOrCreateAccount(context.Background(), "MasterRevenue", finance.TypeRevenue)
	if err != nil {
		return err
	}

	amountPaise := amount // Already in cents

	entries := []finance.EntryData{
		{AccountID: heldFundsAcct.ID, Direction: finance.DirectionDebit, Amount: amountPaise},
		{AccountID: masterRevenueAcct.ID, Direction: finance.DirectionCredit, Amount: amountPaise},
	}

	_, err = s.FinanceService.RecordTransaction(
		context.Background(),
		uuid.New().String(),
		"STANDARD_PAYOUT",
		"Payout for appointment",
		entries,
	)
	return err
}
