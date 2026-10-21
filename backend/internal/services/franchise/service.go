package franchise

import (
	"context"
	"fmt"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/appointment"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/franchise"
	"drayya-core/internal/services/financials"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	DB             *gorm.DB
	FinanceService finance.Service
}

func NewService() *Service {
	return &Service{
		DB:             database.DB,
		FinanceService: finance.NewService(finance.NewRepository(database.DB)),
	}
}

// ProcessConsultationRoyalty handles the financial split for a consultation
func (s *Service) ProcessConsultationRoyalty(appointmentID uint, franchiseID uint, totalAmount int64) error {
	var f franchise.Franchise
	if err := s.DB.First(&f, franchiseID).Error; err != nil {
		return err
	}

	if !f.IsActive {
		return fmt.Errorf("franchise %d is inactive", franchiseID)
	}

	royaltyPaise := int64(float64(totalAmount) * f.RoyaltyRate / 100.0)

	// Ledger accounts
	revenueAcctName := "Revenue:Consultations"
	franchiseAcctName := fmt.Sprintf("FranchiseWallet:%d", f.OwnerID)

	revenueAcct, err := s.FinanceService.GetOrCreateAccount(context.Background(), revenueAcctName, finance.TypeRevenue)
	if err != nil {
		return err
	}

	franchiseAcct, err := s.FinanceService.GetOrCreateAccount(context.Background(), franchiseAcctName, finance.TypeLiability)
	if err != nil {
		return err
	}

	// Prepare entries
	// In double entry: Revenue (Credit increases) -> Liability (Credit increases)
	// Wait, Revenue is credited when earned. If we split a part of it to Franchise,
	// we are effectively reducing our Net Revenue or creating a payout obligation.
	// Correct Flow:
	// Total Revenue (Credit) 100
	// Split:
	//   Company Revenue (Credit) 90
	//   Franchise Payable (Credit) 10

	entries := []finance.EntryData{
		{
			AccountID: revenueAcct.ID,
			Direction: finance.DirectionDebit, // Decreasing the gross revenue account to allocate split
			Amount:    royaltyPaise,
		},
		{
			AccountID: franchiseAcct.ID,
			Direction: finance.DirectionCredit, // Increasing Franchise payoff obligation
			Amount:    royaltyPaise,
		},
	}

	txn, err := s.FinanceService.RecordTransaction(
		context.Background(),
		fmt.Sprintf("FRANCHISE_ROYALTY_APP_%d", appointmentID),
		"FRANCHISE_ROYALTY",
		fmt.Sprintf("Royalty for appointment %d", appointmentID),
		entries,
	)
	if err != nil {
		return err
	}

	// Log royalty for reporting
	log := franchise.FranchiseRoyaltyLog{
		FranchiseID:   franchiseID,
		AppointmentID: appointmentID,
		TotalAmount:   totalAmount,
		RoyaltyAmount: royaltyPaise,
		TransactionID: txn.ID.String(),
		CreatedAt:     time.Now(),
	}

	return s.DB.Create(&log).Error
}

func (s *Service) WithdrawFranchiseEarnings(franchiseID uint, amount int64) error {
	var f franchise.Franchise
	if err := s.DB.Preload("Owner").First(&f, franchiseID).Error; err != nil {
		return err
	}

	amountPaise := amount // Already in cents

	// In double entry:
	// Debit: FranchiseWallet (reducing liability)
	// Credit: CentralBank / Cash (reducing asset)
	franchiseAcctName := fmt.Sprintf("FranchiseWallet:%d", f.OwnerID)
	franchiseAcct, err := s.FinanceService.GetOrCreateAccount(context.Background(), franchiseAcctName, finance.TypeLiability)
	if err != nil {
		return err
	}

	bankAcct, err := s.FinanceService.GetOrCreateAccount(context.Background(), "Bank:Operations", finance.TypeAsset)
	if err != nil {
		return err
	}

	entries := []finance.EntryData{
		{
			AccountID: franchiseAcct.ID,
			Direction: finance.DirectionDebit, // Liability goes down
			Amount:    amountPaise,
		},
		{
			AccountID: bankAcct.ID,
			Direction: finance.DirectionCredit, // Asset goes down
			Amount:    amountPaise,
		},
	}

	refID := fmt.Sprintf("FRANCHISE_PAYOUT_%d_%d", franchiseID, time.Now().Unix())
	_, err = s.FinanceService.RecordTransaction(context.Background(), refID, "FRANCHISE_PAYOUT", "Withdrawal of franchise earnings", entries)
	return err
}

func (s *Service) GetAgreements(status string) ([]franchise.Franchise, error) {
	var results []franchise.Franchise
	query := s.DB.Preload("Owner")
	if status != "" {
		query = query.Where("is_active = ?", status == "active")
	}
	err := query.Find(&results).Error
	return results, err
}

func (s *Service) GetPerformance(franchiseID uint) (interface{}, error) {
	var f franchise.Franchise
	if err := s.DB.First(&f, franchiseID).Error; err != nil {
		return nil, err
	}

	var totalRevenue, totalRoyalty int64
	s.DB.Model(&franchise.FranchiseRoyaltyLog{}).
		Where("franchise_id = ?", franchiseID).
		Select("COALESCE(SUM(total_amount), 0), COALESCE(SUM(royalty_amount), 0)").
		Row().Scan(&totalRevenue, &totalRoyalty)

	// Get current balance from ledger
	franchiseAcctName := fmt.Sprintf("FranchiseWallet:%d", f.OwnerID)
	balance, _ := financials.GetAccountBalance(context.Background(), franchiseAcctName)

	return map[string]interface{}{
		"franchise_id":    franchiseID,
		"total_revenue":   float64(totalRevenue) / 100.0,
		"total_royalty":   float64(totalRoyalty) / 100.0,
		"current_balance": float64(balance) / 100.0,
		"unit_count":      1, // Simplified for now
	}, nil
}

func (s *Service) CreateFranchise(f franchise.Franchise) (*franchise.Franchise, error) {
	if err := s.DB.Create(&f).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

type EnterpriseAnalytics struct {
	OwnedRevenue     int64 `json:"owned_revenue"`
	FranchiseRevenue int64 `json:"franchise_revenue"`
	OwnedAppts       int64 `json:"owned_appts"`
	FranchiseAppts   int64 `json:"franchise_appts"`
}

// ManagedUnit represents a hospital/clinic unit managed by a user
type ManagedUnit struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Location string `json:"location"`
	IsActive bool   `json:"is_active"`
}

// EnterpriseClient represents a B2B enterprise client
type EnterpriseClient struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
	Type   string `json:"type"`
}

func (s *Service) GetManagedUnits(userID uint) ([]ManagedUnit, error) {
	var units []ManagedUnit
	// Get hospitals where user is a manager/head
	rows, err := s.DB.Raw(`
		SELECT h.id, h.name, COALESCE(h.hospital_type, 'owned') as type,
		       COALESCE(h.address, '') as location, h.is_active
		FROM hospitals h
		JOIN hospital_doctors hd ON hd.hospital_id = h.id
		WHERE hd.doctor_id = ? AND (hd.role = 'head' OR hd.role = 'manager')
	`, userID).Rows()
	if err != nil {
		return []ManagedUnit{}, nil
	}
	defer rows.Close()
	for rows.Next() {
		var u ManagedUnit
		rows.Scan(&u.ID, &u.Name, &u.Type, &u.Location, &u.IsActive)
		units = append(units, u)
	}
	if units == nil {
		return []ManagedUnit{}, nil
	}
	return units, nil
}

func (s *Service) GetEnterpriseClients() ([]EnterpriseClient, error) {
	var franchises []franchise.Franchise
	if err := s.DB.Find(&franchises).Error; err != nil {
		return []EnterpriseClient{}, nil
	}
	clients := make([]EnterpriseClient, len(franchises))
	for i, f := range franchises {
		clients[i] = EnterpriseClient{ID: f.ID, Name: f.Name, Type: "franchise"}
	}
	return clients, nil
}

func (s *Service) GetEnterpriseAnalytics() (EnterpriseAnalytics, error) {
	var stats EnterpriseAnalytics

	// Owned Hospitals (HospitalType = 'owned' or nil)
	s.DB.Model(&appointment.Appointment{}).
		Joins("JOIN hospital_doctors ON hospital_doctors.doctor_id = appointments.doctor_id").
		Joins("JOIN hospitals ON hospitals.id = hospital_doctors.hospital_id").
		Where("hospitals.hospital_type = ? OR hospitals.hospital_type IS NULL", "owned").
		Count(&stats.OwnedAppts)

	// Franchise Hospitals
	s.DB.Model(&appointment.Appointment{}).
		Joins("JOIN hospital_doctors ON hospital_doctors.doctor_id = appointments.doctor_id").
		Joins("JOIN hospitals ON hospitals.id = hospital_doctors.hospital_id").
		Where("hospitals.hospital_type = ?", "franchise").
		Count(&stats.FranchiseAppts)

	// Sum Revenue from Royalty Logs (Franchise Dr.Ayya Cut)
	s.DB.Model(&franchise.FranchiseRoyaltyLog{}).
		Select("COALESCE(SUM(royalty_amount), 0)").
		Scan(&stats.FranchiseRevenue)

	// Sum Revenue from non-franchise appointments (placeholder logic)
	// In a real system, we'd query the ledger for MasterRevenue entries marked STANDARD_PAYOUT
	s.DB.Table("ledger_entry_v2").
		Joins("JOIN ledger_account_v2 ON ledger_account_v2.id = ledger_entry_v2.account_id").
		Joins("JOIN ledger_transaction_v2 ON ledger_transaction_v2.id = ledger_entry_v2.transaction_id").
		Where("ledger_account_v2.name = ? AND ledger_transaction_v2.type = ?", "MasterRevenue", "STANDARD_PAYOUT").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&stats.OwnedRevenue)

	return stats, nil
}
