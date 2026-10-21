package insurance

import (
	"context"
	"errors"
	"fmt"
	"drayya-core/internal/domain/finance"
	"time"
)

type Service interface {
	SubmitClaim(claim InsuranceClaim) (*InsuranceClaim, error)
	ApproveClaim(claimID uint, remarks string) error
	RejectClaim(claimID uint, remarks string) error
	SettleClaim(claimID uint) error
	GetClaim(claimID uint) (*InsuranceClaim, error)
	ListClaims(uheid string, status ClaimStatus) ([]InsuranceClaim, error)
	
	CreateProvider(name, code string) (*InsuranceProvider, error)
	ListProviders() ([]InsuranceProvider, error)
}

type service struct {
	repo       Repository
	financeSvc finance.Service
}

func NewService(repo Repository, financeSvc finance.Service) Service {
	return &service{repo: repo, financeSvc: financeSvc}
}

func (s *service) CreateProvider(name, code string) (*InsuranceProvider, error) {
	provider := &InsuranceProvider{
		Name: name,
		Code: code,
	}
	err := s.repo.CreateProvider(provider)
	return provider, err
}

func (s *service) ListProviders() ([]InsuranceProvider, error) {
	return s.repo.ListProviders()
}

func (s *service) SubmitClaim(claim InsuranceClaim) (*InsuranceClaim, error) {
	claim.ClaimNumber = fmt.Sprintf("CLM-%d-%s", time.Now().Unix(), claim.UHEID)
	claim.Status = ClaimPending
	err := s.repo.CreateClaim(&claim)
	return &claim, err
}

func (s *service) ApproveClaim(claimID uint, remarks string) error {
	claim, err := s.repo.GetClaimByID(claimID)
	if err != nil {
		return err
	}

	if claim.Status != ClaimPending {
		return errors.New("only pending claims can be approved")
	}

	updates := map[string]interface{}{
		"status":  ClaimApproved,
		"remarks": remarks,
	}

	return s.repo.UpdateClaim(claimID, updates)
}

func (s *service) RejectClaim(claimID uint, remarks string) error {
	return s.repo.UpdateClaim(claimID, map[string]interface{}{
		"status":  ClaimRejected,
		"remarks": remarks,
	})
}

func (s *service) SettleClaim(claimID uint) error {
	claim, err := s.repo.GetClaimByID(claimID)
	if err != nil {
		return err
	}

	if claim.Status != ClaimApproved {
		return errors.New("only approved claims can be settled")
	}

	if s.financeSvc == nil {
		return errors.New("finance service not available")
	}

	// 1. Create Ledger Transaction
	// Debit Asset:Receivable (Insurer)
	// Credit Liability:Provider (Hospital)
	
	insurerName := fmt.Sprintf("Insurer:%d", claim.InsurerID)
	providerName := fmt.Sprintf("ExternalProvider:%d", claim.ProviderID)
	
	acctInsurer, _ := s.financeSvc.GetOrCreateAccount(context.Background(), insurerName, finance.TypeAsset)
	acctProvider, _ := s.financeSvc.GetOrCreateAccount(context.Background(), providerName, finance.TypeLiability)

	entries := []finance.EntryData{
		{AccountID: acctInsurer.ID, Direction: finance.DirectionDebit, Amount: claim.Amount},
		{AccountID: acctProvider.ID, Direction: finance.DirectionCredit, Amount: claim.Amount},
	}

	tx, err := s.financeSvc.RecordTransaction(context.Background(), claim.ClaimNumber, "INSURANCE_SETTLEMENT", 
		fmt.Sprintf("Settlement for claim %s", claim.ClaimNumber), entries)
	if err != nil {
		return err
	}

	// 2. Update Claim Status
	return s.repo.UpdateClaim(claimID, map[string]interface{}{
		"status":                ClaimSettled,
		"escrow_transaction_id": tx.ID.String(),
	})
}

func (s *service) GetClaim(claimID uint) (*InsuranceClaim, error) {
	return s.repo.GetClaimByID(claimID)
}

func (s *service) ListClaims(uheid string, status ClaimStatus) ([]InsuranceClaim, error) {
	return s.repo.ListClaims(uheid, status)
}
