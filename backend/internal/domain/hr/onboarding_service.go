package hr

import (
	"errors"
)

type OnboardingService interface {
	GetAllOnboarding() ([]OnboardingRecord, error)
	GetOnboarding(id uint) (*OnboardingRecord, error)
	InitiateOnboarding(staffID uint) (*OnboardingRecord, error)
	UpdateDocumentStatus(id uint, docType, status string) error
	UpdateTaskStatus(id uint, taskType string, isComplete bool) error
	UpdateOnboardingStatus(id uint, status string) error
}

type onboardingService struct {
	repo OnboardingRepository
}

func NewOnboardingService(repo OnboardingRepository) OnboardingService {
	return &onboardingService{repo: repo}
}

func (s *onboardingService) GetAllOnboarding() ([]OnboardingRecord, error) {
	return s.repo.GetOnboardingRecords()
}

func (s *onboardingService) GetOnboarding(id uint) (*OnboardingRecord, error) {
	return s.repo.GetOnboardingByID(id)
}

func (s *onboardingService) InitiateOnboarding(staffID uint) (*OnboardingRecord, error) {
	// Check if already exists
	existing, err := s.repo.GetOnboardingByStaffID(staffID)
	if err == nil && existing != nil {
		return existing, nil // Already onboarding
	}

	record := &OnboardingRecord{
		StaffID: staffID,
		Status:  "pending_docs",
	}

	if err := s.repo.CreateOnboarding(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *onboardingService) UpdateDocumentStatus(id uint, docType, status string) error {
	record, err := s.repo.GetOnboardingByID(id)
	if err != nil {
		return err
	}

	switch docType {
	case "certificate":
		record.Certificate = status
	case "aadhar_id":
		record.AadharID = status
	case "pan_card":
		record.PanCard = status
	case "bank_account":
		record.BankAccount = status
	case "passport_photo":
		record.PassportPhoto = status
	default:
		return errors.New("invalid document type")
	}

	return s.repo.UpdateOnboarding(record)
}

func (s *onboardingService) UpdateTaskStatus(id uint, taskType string, isComplete bool) error {
	record, err := s.repo.GetOnboardingByID(id)
	if err != nil {
		return err
	}

	switch taskType {
	case "system_access_granted":
		record.SystemAccess = isComplete
	case "handbook_read":
		record.HandbookRead = isComplete
	case "welcome_email_sent":
		record.WelcomeEmailSent = isComplete
	default:
		return errors.New("invalid task type")
	}

	return s.repo.UpdateOnboarding(record)
}

func (s *onboardingService) UpdateOnboardingStatus(id uint, status string) error {
	record, err := s.repo.GetOnboardingByID(id)
	if err != nil {
		return err
	}
	
	record.Status = status
	return s.repo.UpdateOnboarding(record)
}
