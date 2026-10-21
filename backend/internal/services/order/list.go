package order

import (
	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
)

// ListOrders retrieves orders for a specific user
func ListOrders(userID uint) ([]pharmacy.PharmacyOrder, error) {
	var orders []pharmacy.PharmacyOrder
	err := database.DB.Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&orders).Error
	return orders, err
}

// GetOrderDetails retrieves a specific order with its items
func GetOrderDetails(orderID uint, userID uint) (*pharmacy.PharmacyOrder, error) {
	var order pharmacy.PharmacyOrder
	err := database.DB.Where("id = ? AND user_id = ?", orderID, userID).
		First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}
