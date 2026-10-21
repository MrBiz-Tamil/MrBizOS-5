package insurance

import (
	"gorm.io/gorm"
)

type Repository interface {
	CreateProvider(provider *InsuranceProvider) error
	GetProviderByCode(code string) (*InsuranceProvider, error)
	ListProviders() ([]InsuranceProvider, error)

	CreateClaim(claim *InsuranceClaim) error
	GetClaimByID(id uint) (*InsuranceClaim, error)
	GetClaimByNumber(number string) (*InsuranceClaim, error)
	UpdateClaim(id uint, updates map[string]interface{}) error
	ListClaims(uheid string, status ClaimStatus) ([]InsuranceClaim, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateProvider(provider *InsuranceProvider) error {
	return r.db.Create(provider).Error
}

func (r *repository) GetProviderByCode(code string) (*InsuranceProvider, error) {
	var provider InsuranceProvider
	err := r.db.Where("code = ?", code).First(&provider).Error
	return &provider, err
}

func (r *repository) ListProviders() ([]InsuranceProvider, error) {
	var providers []InsuranceProvider
	err := r.db.Find(&providers).Error
	return providers, err
}

func (r *repository) CreateClaim(claim *InsuranceClaim) error {
	return r.db.Create(claim).Error
}

func (r *repository) GetClaimByID(id uint) (*InsuranceClaim, error) {
	var claim InsuranceClaim
	err := r.db.First(&claim, id).Error
	return &claim, err
}

func (r *repository) GetClaimByNumber(number string) (*InsuranceClaim, error) {
	var claim InsuranceClaim
	err := r.db.Where("claim_number = ?", number).First(&claim).Error
	return &claim, err
}

func (r *repository) UpdateClaim(id uint, updates map[string]interface{}) error {
	return r.db.Model(&InsuranceClaim{}).Where("id = ?", id).Updates(updates).Error
}

func (r *repository) ListClaims(uheid string, status ClaimStatus) ([]InsuranceClaim, error) {
	var claims []InsuranceClaim
	query := r.db.Model(&InsuranceClaim{})
	if uheid != "" {
		query = query.Where("uheid = ?", uheid)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&claims).Error
	return claims, err
}
