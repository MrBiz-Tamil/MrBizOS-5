package store

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
)

// SearchProducts filters pharmacy inventory
func SearchProducts(category, search, brand string, minPrice, maxPrice float64, sortBy string, limit, offset int) ([]pharmacy.PharmacyInventory, error) {
	db := database.DB
	var products []pharmacy.PharmacyInventory

	query := db.Model(&pharmacy.PharmacyInventory{}).
		Where("is_active = ? AND stock_quantity > ?", true, 0)

	// Joins would be needed if Category/Brand are in Medicine table and Inventory links to it.
	// Looking at models/pharmacy.go:
	// PharmacyInventory has MedicineID.
	// We need to join Medicine table to filter by name, brand, category if they are not cached in Inventory.
	// Python code joined `Medicine`.

	query = query.Joins("Medicine") // Preload or Join? Join for filtering.

	if category != "" {
		query = query.Where("Medicine.category = ?", category)
	}
	if brand != "" {
		query = query.Where("Medicine.brand ILIKE ?", "%"+brand+"%")
	}
	if search != "" {
		// Search in Medicine Name or Generic Name
		query = query.Where("Medicine.name ILIKE ? OR Medicine.generic_name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if minPrice > 0 {
		query = query.Where("Medicine.unit_price >= ?", minPrice)
	}
	if maxPrice > 0 {
		query = query.Where("Medicine.unit_price <= ?", maxPrice)
	}

	// Sorting
	switch sortBy {
	case "price_asc":
		query = query.Order("Medicine.unit_price asc")
	case "price_desc":
		query = query.Order("Medicine.unit_price desc")
	case "newest":
		query = query.Order("created_at desc")
	default:
		query = query.Order("Medicine.unit_price asc")
	}

	// Due to `Joins("Medicine")`, the order by Clause works automatically if column exists in the Joined table.
	err := query.Preload("Medicine").Limit(limit).Offset(offset).Find(&products).Error
	return products, err
}

// GetProductDetails fetches a single product
func GetProductDetails(id uint) (*pharmacy.PharmacyInventory, error) {
	var product pharmacy.PharmacyInventory
	err := database.DB.Preload("Medicine").First(&product, id).Error
	return &product, err
}

// GetCategories returns unique medicine categories
func GetCategories() ([]string, error) {
	var categories []string
	err := database.DB.Model(&pharmacy.Medicine{}).Distinct().Pluck("category", &categories).Error
	return categories, err
}

// CreateProduct creates a new inventory item
func CreateProduct(product pharmacy.PharmacyInventory) (*pharmacy.PharmacyInventory, error) {
	err := database.DB.Create(&product).Error
	return &product, err
}

// UpdateProduct updates product details
func UpdateProduct(id uint, updates map[string]interface{}) error {
	return database.DB.Model(&pharmacy.PharmacyInventory{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteProduct deletes a product
func DeleteProduct(id uint) error {
	return database.DB.Delete(&pharmacy.PharmacyInventory{}, id).Error
}
