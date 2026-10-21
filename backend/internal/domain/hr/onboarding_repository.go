package hr

import "gorm.io/gorm"

type OnboardingRepository interface {
	GetOnboardingRecords() ([]OnboardingRecord, error)
	GetOnboardingByID(id uint) (*OnboardingRecord, error)
	GetOnboardingByStaffID(staffID uint) (*OnboardingRecord, error)
	CreateOnboarding(record *OnboardingRecord) error
	UpdateOnboarding(record *OnboardingRecord) error
}

type onboardingRepository struct {
	db *gorm.DB
}

func NewOnboardingRepository(db *gorm.DB) OnboardingRepository {
	return &onboardingRepository{db: db}
}

func (r *onboardingRepository) GetOnboardingRecords() ([]OnboardingRecord, error) {
	var records []OnboardingRecord
	err := r.db.Preload("StaffProfile").Find(&records).Error
	return records, err
}

func (r *onboardingRepository) GetOnboardingByID(id uint) (*OnboardingRecord, error) {
	var record OnboardingRecord
	err := r.db.Preload("StaffProfile").First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *onboardingRepository) GetOnboardingByStaffID(staffID uint) (*OnboardingRecord, error) {
	var record OnboardingRecord
	err := r.db.Where("staff_id = ?", staffID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *onboardingRepository) CreateOnboarding(record *OnboardingRecord) error {
	return r.db.Create(record).Error
}

func (r *onboardingRepository) UpdateOnboarding(record *OnboardingRecord) error {
	return r.db.Save(record).Error
}
