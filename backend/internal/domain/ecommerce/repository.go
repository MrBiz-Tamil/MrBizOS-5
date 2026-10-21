package ecommerce

import (
	"gorm.io/gorm"
)

type Repository interface {
	CreatePODOrder(order *PODOrder) error
	GetPODOrder(id uint) (*PODOrder, error)
	ListPODOrders() ([]PODOrder, error)
	CreateListing(listing *MarketplaceListing) error
	ListListings(marketplace string) ([]MarketplaceListing, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreatePODOrder(order *PODOrder) error {
	return r.db.Create(order).Error
}

func (r *repository) GetPODOrder(id uint) (*PODOrder, error) {
	var order PODOrder
	err := r.db.First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *repository) ListPODOrders() ([]PODOrder, error) {
	var orders []PODOrder
	err := r.db.Order("created_at DESC").Find(&orders).Error
	return orders, err
}

func (r *repository) CreateListing(listing *MarketplaceListing) error {
	return r.db.Create(listing).Error
}

func (r *repository) ListListings(marketplace string) ([]MarketplaceListing, error) {
	var listings []MarketplaceListing
	q := r.db.Where("status = ?", "ACTIVE")
	if marketplace != "" {
		q = q.Where("marketplace = ?", marketplace)
	}
	err := q.Order("created_at DESC").Find(&listings).Error
	return listings, err
}
