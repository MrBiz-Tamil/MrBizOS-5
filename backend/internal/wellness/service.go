package wellness

import (
	"context"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/store"
	"drayya-core/internal/utils/privacy"
	"time"
)

type ItemRequest struct {
	ProductID uint `json:"product_id"`
	Quantity  int  `json:"quantity"`
}

type OrderRequest struct {
	Items           []ItemRequest `json:"items"`
	DeliveryAddress string        `json:"delivery_address"`
}

// Global variable for testing or simple DI
var Service = &wellnessService{}

type wellnessService struct{}

func (s *wellnessService) CreateOrder(ctx context.Context, userID uint, req OrderRequest) (*store.WellnessOrder, error) {
	tx := database.DB.Begin()

	order := store.WellnessOrder{
		UserID:          userID,
		Status:          "pending",
		DeliveryAddress: privacy.EncryptedString(req.DeliveryAddress),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	var totalAmount int64

	for _, itemReq := range req.Items {
		var product store.WellnessProduct
		if err := tx.First(&product, itemReq.ProductID).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		orderItem := store.WellnessOrderItem{
			OrderID:   order.ID,
			ProductID: itemReq.ProductID,
			Quantity:  itemReq.Quantity,
			UnitPrice: product.SellingPrice,
		}

		if err := tx.Create(&orderItem).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		totalAmount += product.SellingPrice * int64(itemReq.Quantity)
	}

	order.TotalAmount = totalAmount
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return &order, nil
}
