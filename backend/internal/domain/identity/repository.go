package identity

import (
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	// User
	CreateUser(user *User) error
	GetUserByID(id uint) (*User, error)
	GetUserByEmail(email string) (*User, error)
	GetUserByPhone(phone string) (*User, error)
	GetUserByUHEID(uheid string) (*User, error)
	UpdateUser(id uint, updates map[string]interface{}) error

	// Patient Profile
	CreatePatientProfile(profile *PatientProfile) error
	GetPatientProfileByUserID(userID uint) (*PatientProfile, error)
	UpdatePatientProfile(userID uint, updates map[string]interface{}) error

	// Doctor Profile
	CreateDoctorProfile(profile *DoctorProfile) error
	GetDoctorProfileByUserID(userID uint) (*DoctorProfile, error)
	UpdateDoctorProfile(userID uint, updates map[string]interface{}) error
	ListDoctors(specialization string, verifiedOnly bool) ([]User, error)
	RateDoctor(doctorID uint, rating float64) error

	// Doctor Availability
	GetDoctorAvailability(doctorID uint) ([]DoctorAvailability, error)
	UpdateDoctorAvailability(doctorID uint, availability []DoctorAvailability) error

	// Family Member
	CreateFamilyMember(member *FamilyMember) error
	GetFamilyMembersByUserID(userID uint) ([]FamilyMember, error)
	GetFamilyMemberByID(id uint, userID uint) (*FamilyMember, error)
	UpdateFamilyMember(id uint, userID uint, updates map[string]interface{}) error
	DeleteFamilyMember(id uint, userID uint) error

	// Activity Log
	CreateActivityLog(log *ActivityLog) error

	// External Provider & Consent (UHEID Portability)
	CreateExternalProvider(provider *ExternalProvider) error
	GetExternalProviderByAPIKey(apiKey string) (*ExternalProvider, error)
	CreateConsent(consent *Consent) error
	GetActiveConsent(userID, providerID uint, scope ConsentScope) (*Consent, error)
	RevokeConsent(userID, providerID uint) error

	// Transaction support
	ExecuteTransaction(fn func(txRepo Repository) error) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateUser(user *User) error {
	return r.db.Create(user).Error
}

func (r *repository) GetUserByID(id uint) (*User, error) {
	var user User
	err := r.db.First(&user, id).Error
	return &user, err
}

func (r *repository) GetUserByEmail(email string) (*User, error) {
	var user User
	err := r.db.Unscoped().Where("LOWER(email) = LOWER(?)", email).First(&user).Error
	return &user, err
}

func (r *repository) GetUserByPhone(phone string) (*User, error) {
	var user User
	// handle different phone formats: try with 91, without 91, with +, etc.
	// But in DB we mostly store 10 digits or 91+digits.
	err := r.db.Unscoped().Where("phone_number = ? OR phone_number = ? OR phone_number = ?",
		phone, "91"+phone, "+"+phone).First(&user).Error
	return &user, err
}

func (r *repository) GetUserByUHEID(uheid string) (*User, error) {
	var user User
	err := r.db.Where("uheid = ?", uheid).First(&user).Error
	return &user, err
}

func (r *repository) UpdateUser(id uint, updates map[string]interface{}) error {
	return r.db.Unscoped().Model(&User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *repository) CreatePatientProfile(profile *PatientProfile) error {
	return r.db.Create(profile).Error
}

func (r *repository) GetPatientProfileByUserID(userID uint) (*PatientProfile, error) {
	var profile PatientProfile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	return &profile, err
}

func (r *repository) UpdatePatientProfile(userID uint, updates map[string]interface{}) error {
	return r.db.Model(&PatientProfile{}).Where("user_id = ?", userID).Updates(updates).Error
}

func (r *repository) CreateDoctorProfile(profile *DoctorProfile) error {
	return r.db.Create(profile).Error
}

func (r *repository) GetDoctorProfileByUserID(userID uint) (*DoctorProfile, error) {
	var profile DoctorProfile
	err := r.db.Where("user_id = ?", userID).First(&profile).Error
	return &profile, err
}

func (r *repository) UpdateDoctorProfile(userID uint, updates map[string]interface{}) error {
	return r.db.Model(&DoctorProfile{}).Where("user_id = ?", userID).Updates(updates).Error
}

func (r *repository) ListDoctors(specialization string, verifiedOnly bool) ([]User, error) {
	// Use a temporary struct to capture joined fields that are marked gorm:"-" in User
	type doctorScanRow struct {
		User
		IsOnlineShadow       bool    `gorm:"column:is_online"`
		SpecializationShadow string  `gorm:"column:specialization"`
		CurrentStatusShadow  string  `gorm:"column:current_status"`
		AverageRatingShadow  float32 `gorm:"column:average_rating"`
	}

	var rows []doctorScanRow
	query := r.db.Table("\"user\"").
		Select("\"user\".*, doctorprofile.specialization, doctorprofile.is_online, doctorprofile.current_status, doctorprofile.average_rating").
		Joins("JOIN doctorprofile ON \"user\".id = doctorprofile.user_id").
		Where("\"user\".role IN ?", []string{"doctor", "chief_doctor", "hospital_head_doctor"})

	if specialization != "" {
		query = query.Where("LOWER(doctorprofile.specialization) = LOWER(?)", specialization)
	}

	if verifiedOnly {
		query = query.Where("doctorprofile.is_verified = ?", true)
	}

	err := query.Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// Map scan rows back to User objects with populated shadow fields
	doctors := make([]User, len(rows))
	for i, row := range rows {
		doctors[i] = row.User
		doctors[i].IsOnline = row.IsOnlineShadow
		doctors[i].Specialization = row.SpecializationShadow
		doctors[i].CurrentStatus = row.CurrentStatusShadow
		doctors[i].AverageRating = row.AverageRatingShadow
		doctors[i].PasswordHash = "" // Safety first
	}

	return doctors, nil
}

func (r *repository) RateDoctor(doctorID uint, rating float64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var profile DoctorProfile
		if err := tx.Where("user_id = ?", doctorID).First(&profile).Error; err != nil {
			return err
		}

		// Calculate new average rating
		totalRating := profile.AverageRating * float64(profile.TotalConsultations)
		profile.TotalConsultations++
		profile.AverageRating = (totalRating + rating) / float64(profile.TotalConsultations)

		return tx.Save(&profile).Error
	})
}

func (r *repository) GetDoctorAvailability(doctorID uint) ([]DoctorAvailability, error) {
	var availability []DoctorAvailability
	err := r.db.Where("doctor_id = ?", doctorID).Order("day_of_week ASC").Find(&availability).Error
	return availability, err
}

func (r *repository) UpdateDoctorAvailability(doctorID uint, availability []DoctorAvailability) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("doctor_id = ?", doctorID).Delete(&DoctorAvailability{}).Error; err != nil {
			return err
		}
		if len(availability) > 0 {
			for i := range availability {
				availability[i].DoctorID = doctorID
			}
			if err := tx.Create(&availability).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *repository) CreateFamilyMember(member *FamilyMember) error {
	return r.db.Create(member).Error
}

func (r *repository) GetFamilyMembersByUserID(userID uint) ([]FamilyMember, error) {
	var members []FamilyMember
	err := r.db.Where("user_id = ?", userID).Find(&members).Error
	return members, err
}

func (r *repository) GetFamilyMemberByID(id uint, userID uint) (*FamilyMember, error) {
	var member FamilyMember
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&member).Error
	return &member, err
}

func (r *repository) UpdateFamilyMember(id uint, userID uint, updates map[string]interface{}) error {
	return r.db.Model(&FamilyMember{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates).Error
}

func (r *repository) DeleteFamilyMember(id uint, userID uint) error {
	return r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&FamilyMember{}).Error
}

func (r *repository) CreateActivityLog(log *ActivityLog) error {
	return r.db.Create(log).Error
}

func (r *repository) CreateExternalProvider(provider *ExternalProvider) error {
	return r.db.Create(provider).Error
}

func (r *repository) GetExternalProviderByAPIKey(apiKey string) (*ExternalProvider, error) {
	var provider ExternalProvider
	err := r.db.Where("api_key = ? AND is_active = ?", apiKey, true).First(&provider).Error
	return &provider, err
}

func (r *repository) CreateConsent(consent *Consent) error {
	return r.db.Create(consent).Error
}

func (r *repository) GetActiveConsent(userID, providerID uint, scope ConsentScope) (*Consent, error) {
	var consent Consent
	now := time.Now()
	err := r.db.Where("user_id = ? AND external_provider_id = ? AND scope = ? AND expires_at > ? AND is_revoked = ?",
		userID, providerID, scope, now, false).First(&consent).Error
	return &consent, err
}

func (r *repository) RevokeConsent(userID, providerID uint) error {
	return r.db.Model(&Consent{}).Where("user_id = ? AND external_provider_id = ?", userID, providerID).
		Update("is_revoked", true).Error
}

func (r *repository) ExecuteTransaction(fn func(txRepo Repository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(&repository{db: tx})
	})
}
