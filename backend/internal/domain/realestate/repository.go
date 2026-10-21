package realestate

import (
	"gorm.io/gorm"
)

type Repository interface {
	CreateProperty(prop *Property) error
	UpdateProperty(prop *Property) error
	GetProperty(id uint) (*Property, error)
	ListProperties(city string, txType string, maxPrice float64) ([]Property, error)
	CreateSiteVisit(visit *SiteVisit) error
	ListSiteVisits(agentID uint, buyerID uint) ([]SiteVisit, error)
	CreateDeal(deal *PropertyDeal) error
	GetDeal(id uint) (*PropertyDeal, error)
	UpdateDeal(deal *PropertyDeal) error
	ListDeals() ([]PropertyDeal, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateProperty(prop *Property) error {
	return r.db.Create(prop).Error
}

func (r *repository) UpdateProperty(prop *Property) error {
	return r.db.Save(prop).Error
}

func (r *repository) GetProperty(id uint) (*Property, error) {
	var prop Property
	err := r.db.First(&prop, id).Error
	if err != nil {
		return nil, err
	}
	return &prop, nil
}

func (r *repository) ListProperties(city string, txType string, maxPrice float64) ([]Property, error) {
	var props []Property
	q := r.db.Where("status = ?", "AVAILABLE")
	if city != "" {
		q = q.Where("city = ?", city)
	}
	if txType != "" {
		q = q.Where("transaction_type = ?", txType)
	}
	if maxPrice > 0 {
		q = q.Where("price <= ?", maxPrice)
	}
	err := q.Order("created_at DESC").Find(&props).Error
	return props, err
}

func (r *repository) CreateSiteVisit(visit *SiteVisit) error {
	return r.db.Create(visit).Error
}

func (r *repository) ListSiteVisits(agentID uint, buyerID uint) ([]SiteVisit, error) {
	var visits []SiteVisit
	q := r.db.Preload("Property")
	if agentID > 0 {
		q = q.Where("agent_id = ?", agentID)
	}
	if buyerID > 0 {
		q = q.Where("buyer_id = ?", buyerID)
	}
	err := q.Order("scheduled_at ASC").Find(&visits).Error
	return visits, err
}

func (r *repository) CreateDeal(deal *PropertyDeal) error {
	return r.db.Create(deal).Error
}

func (r *repository) GetDeal(id uint) (*PropertyDeal, error) {
	var deal PropertyDeal
	err := r.db.Preload("Property").First(&deal, id).Error
	if err != nil {
		return nil, err
	}
	return &deal, nil
}

func (r *repository) UpdateDeal(deal *PropertyDeal) error {
	return r.db.Save(deal).Error
}

func (r *repository) ListDeals() ([]PropertyDeal, error) {
	var deals []PropertyDeal
	err := r.db.Preload("Property").Order("created_at DESC").Find(&deals).Error
	return deals, err
}
