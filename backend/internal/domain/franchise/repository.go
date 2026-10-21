package franchise

import (
	"gorm.io/gorm"
)

type Repository interface {
	AutoMigrate() error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) AutoMigrate() error {
	return r.db.AutoMigrate(&Franchise{}, &FranchiseRoyaltyLog{})
}
