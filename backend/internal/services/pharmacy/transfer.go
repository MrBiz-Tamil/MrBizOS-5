//go:build ignore
// +build ignore

package pharmacy

import (
	"errors"
	"drayya-core/internal/database"
	"drayya-core/internal/domain/pharmacy"
	"drayya-core/internal/models"
	"time"

	"gorm.io/gorm"
)

type TransferService struct {
	DB *gorm.DB
}

func NewTransferService() *TransferService {
	return &TransferService{DB: database.DB}
}

func (s *TransferService) RequestTransfer(req pharmacy.InventoryTransfer) (*pharmacy.InventoryTransfer, error) {
	req.Status = "requested"
	req.CreatedAt = time.Now()
	if err := s.DB.Create(&req).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *TransferService) ProcessTransfer(transferID uint, processedBy uint, status string) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var transfer pharmacy.InventoryTransfer
		if err := tx.First(&transfer, transferID).Error; err != nil {
			return err
		}

		if transfer.Status == "completed" || transfer.Status == "cancelled" {
			return errors.New("cannot process a completed or cancelled transfer")
		}

		transfer.Status = status
		transfer.ProcessedByID = &processedBy
		transfer.UpdatedAt = time.Now()

		if status == "completed" {
			// Hub & Spoke Logic: Decrement Source (Hub) and Increment Destination (Spoke)

			// 1. Source (Hub)
			var sourceInv pharmacy.PharmacyInventory
			if err := tx.Where("pharmacy_id = ? AND medicine_id = ?", transfer.SourceID, transfer.MedicineID).First(&sourceInv).Error; err != nil {
				return errors.New("source inventory not found")
			}
			if sourceInv.StockQuantity < transfer.Quantity {
				return errors.New("insufficient stock in source pharmacy")
			}
			sourceInv.StockQuantity -= transfer.Quantity
			if err := tx.Save(&sourceInv).Error; err != nil {
				return err
			}

			// 2. Destination (Spoke)
			var destInv pharmacy.PharmacyInventory
			err := tx.Where("pharmacy_id = ? AND medicine_id = ?", transfer.DestinationID, transfer.MedicineID).First(&destInv).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// Create new inventory record in destination if not exists
					destInv = pharmacy.PharmacyInventory{
						PharmacyID:    transfer.DestinationID,
						MedicineID:    transfer.MedicineID,
						StockQuantity: transfer.Quantity,
						IsActive:      true,
					}
					if err := tx.Create(&destInv).Error; err != nil {
						return err
					}
				} else {
					return err
				}
			} else {
				destInv.StockQuantity += transfer.Quantity
				if err := tx.Save(&destInv).Error; err != nil {
					return err
				}
			}
		}

		return tx.Save(&transfer).Error
	})
}

