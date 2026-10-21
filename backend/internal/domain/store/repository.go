package store

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository interface {
	// Products
	GetProductByID(ctx context.Context, id uint) (*WellnessProduct, error)
	GetProducts(ctx context.Context, category string) ([]WellnessProduct, error) // Added for service compatibility
	SearchProducts(ctx context.Context, query string, category string, brand string, minPrice, maxPrice float64, sort string, limit, offset int) ([]WellnessProduct, error)
	CreateProduct(ctx context.Context, p *WellnessProduct) error
	UpdateProduct(ctx context.Context, id uint, updates map[string]interface{}) error
	DeleteProduct(ctx context.Context, id uint) error
	GetCategories(ctx context.Context) ([]string, error)
	GetInventory(ctx context.Context, productID, pharmacyID uint) (*WellnessInventory, error)
	UpdateInventory(ctx context.Context, productID, pharmacyID uint, qty int) error

	// Orders
	CreateOrder(ctx context.Context, o *WellnessOrder) error
	GetOrderByID(ctx context.Context, id uint) (*WellnessOrder, error)
	GetOrders(ctx context.Context, userID uint) ([]WellnessOrder, error) // Renamed for service compatibility
	UpdateOrderStatus(ctx context.Context, id uint, status string) error
	MarkOrderPaid(ctx context.Context, id uint, paymentRef string) error
	GetServiceableArea(ctx context.Context, pincode string) (*ServiceableArea, error)

	AutoMigrate() error
}

type defaultRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &defaultRepository{db: db}
}

func (r *defaultRepository) AutoMigrate() error {
	return r.db.AutoMigrate(
		&WellnessProduct{},
		&WellnessInventory{},
		&WellnessOrder{},
		&WellnessOrderItem{},
		&City{},
		&ServiceableArea{},
	)
}

func (r *defaultRepository) GetProductByID(ctx context.Context, id uint) (*WellnessProduct, error) {
	var p WellnessProduct
	err := r.db.WithContext(ctx).First(&p, id).Error
	return &p, err
}

func (r *defaultRepository) GetProducts(ctx context.Context, category string) ([]WellnessProduct, error) {
	var products []WellnessProduct
	query := r.db.WithContext(ctx).Model(&WellnessProduct{}).Where("is_active = ?", true)
	if category != "" {
		query = query.Where("category = ?", category)
	}
	err := query.Find(&products).Error
	return products, err
}

func (r *defaultRepository) SearchProducts(ctx context.Context, search string, category string, brand string, minPrice, maxPrice float64, sortBy string, limit, offset int) ([]WellnessProduct, error) {
	var products []WellnessProduct
	query := r.db.WithContext(ctx).Model(&WellnessProduct{}).Where("is_active = ?", true)

	if search != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if brand != "" {
		query = query.Where("brand = ?", brand)
	}
	if minPrice > 0 {
		query = query.Where("selling_price >= ?", int64(minPrice*100))
	}
	if maxPrice > 0 {
		query = query.Where("selling_price <= ?", int64(maxPrice*100))
	}

	switch sortBy {
	case "price_asc":
		query = query.Order("selling_price asc")
	case "price_desc":
		query = query.Order("selling_price desc")
	case "newest":
		query = query.Order("created_at desc")
	default:
		query = query.Order("created_at desc")
	}

	err := query.Limit(limit).Offset(offset).Find(&products).Error
	return products, err
}

func (r *defaultRepository) CreateProduct(ctx context.Context, p *WellnessProduct) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *defaultRepository) UpdateProduct(ctx context.Context, id uint, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&WellnessProduct{}).Where("id = ?", id).Updates(updates).Error
}

func (r *defaultRepository) DeleteProduct(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&WellnessProduct{}, id).Error
}

func (r *defaultRepository) GetCategories(ctx context.Context) ([]string, error) {
	var cats []string
	err := r.db.WithContext(ctx).Model(&WellnessProduct{}).Distinct().Pluck("category", &cats).Error
	return cats, err
}

func (r *defaultRepository) CreateOrder(ctx context.Context, o *WellnessOrder) error {
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *defaultRepository) GetOrderByID(ctx context.Context, id uint) (*WellnessOrder, error) {
	var o WellnessOrder
	err := r.db.WithContext(ctx).Preload("Items.Product").First(&o, id).Error
	return &o, err
}

func (r *defaultRepository) GetOrders(ctx context.Context, userID uint) ([]WellnessOrder, error) {
	var os []WellnessOrder
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Preload("Items.Product").Order("created_at desc").Find(&os).Error
	return os, err
}

func (r *defaultRepository) UpdateOrderStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).Model(&WellnessOrder{}).Where("id = ?", id).Update("status", status).Error
}

func (r *defaultRepository) MarkOrderPaid(ctx context.Context, id uint, paymentRef string) error {
	return r.db.WithContext(ctx).Model(&WellnessOrder{}).Where("id = ?", id).Updates(map[string]interface{}{
		"payment_status": "paid",
		"status":         "processing",
		"payment_ref":    paymentRef,
		"updated_at":     time.Now(),
	}).Error
}

func (r *defaultRepository) GetInventory(ctx context.Context, productID, pharmacyID uint) (*WellnessInventory, error) {
	var inv WellnessInventory
	err := r.db.WithContext(ctx).Where("product_id = ? AND pharmacy_id = ?", productID, pharmacyID).First(&inv).Error
	return &inv, err
}

func (r *defaultRepository) UpdateInventory(ctx context.Context, productID, pharmacyID uint, qty int) error {
	return r.db.WithContext(ctx).Model(&WellnessInventory{}).
		Where("product_id = ? AND pharmacy_id = ?", productID, pharmacyID).
		Update("stock_quantity", qty).Error
}

func (r *defaultRepository) GetServiceableArea(ctx context.Context, pincode string) (*ServiceableArea, error) {
	var sa ServiceableArea
	err := r.db.WithContext(ctx).Where("pincode = ?", pincode).First(&sa).Error
	return &sa, err
}
