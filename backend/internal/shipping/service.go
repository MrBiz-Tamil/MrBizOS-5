package shipping

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"drayya-core/internal/database"
	"drayya-core/internal/domain/finance"
	"drayya-core/internal/domain/pharmacy"
)

// Service manages shipping logic
type Service struct {
	providers  map[pharmacy.DeliveryMode][]ShippingProvider
	financeSvc finance.Service
}

// NewService creates a new shipping service with registered providers
func NewService(financeSvc finance.Service) *Service {
	return &Service{
		providers: map[pharmacy.DeliveryMode][]ShippingProvider{
			pharmacy.ModeQuick: {
				NewLocalDeliveryProvider(),
			},
			pharmacy.ModeStandard: {
				NewShiprocketProvider(),
			},
		},
		financeSvc: financeSvc,
	}
}

// Global instance (To be initialized in app.go)
var ShippingService *Service

// CalculateBestShipping finds the best provider for a given shipment mode
func (s *Service) CalculateBestShipping(mode pharmacy.DeliveryMode, weightKg float64, pickupPincode, destPincode string) (map[string]interface{}, error) {
	providers, ok := s.providers[mode]
	if !ok {
		return nil, errors.New("invalid delivery mode")
	}

	for _, provider := range providers {
		if provider.IsServiceable(destPincode) {
			cost, err := provider.CalculateCost(weightKg, pickupPincode, destPincode)
			if err != nil {
				log.Printf("Provider %s failed cost calc: %v", provider.Name(), err)
				continue
			}

			return map[string]interface{}{
				"provider":       provider.Name(),
				"cost":           cost,
				"pincode":        destPincode,
				"status":         "serviceable",
				"mode":           mode,
				"estimated_days": 1, // Default for quick, can be tuned
			}, nil
		}
	}

	return nil, errors.New("no service reachable for this mode/pincode")
}

func (s *Service) UpdateCourierLocation(deliveryID uint, lat, lng float64, address string) error {
	var delivery pharmacy.CourierDelivery
	if err := database.DB.First(&delivery, deliveryID).Error; err != nil {
		return err
	}

	if err := database.DB.Model(&delivery).Updates(map[string]interface{}{
		"current_latitude":  lat,
		"current_longitude": lng,
		"current_address":   address,
		"updated_at":        time.Now(),
	}).Error; err != nil {
		return err
	}

	// Also update the CourierProfile lat/lng so TryAutoAssign scorer sees up-to-date position
	UpdateCourierPosition(context.Background(), delivery.CourierID, lat, lng)

	return nil
}

func (s *Service) UpdateDeliveryStatus(deliveryID uint, status string) error {
	tx := database.DB.Begin()
	var delivery pharmacy.CourierDelivery
	if err := tx.First(&delivery, deliveryID).Error; err != nil {
		tx.Rollback()
		return err
	}

	delivery.Status = status
	if status == "completed" {
		now := time.Now()
		delivery.ActualDeliveryTime = &now

		// Increment trips for courier
		tx.Model(&pharmacy.CourierProfile{}).Where("user_id = ?", delivery.CourierID).
			UpdateColumn("total_trips", gorm.Expr("total_trips + ?", 1))
	}

	if err := tx.Save(&delivery).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update the parent PharmacyOrder status as well
	if err := tx.Model(&pharmacy.PharmacyOrder{}).Where("id = ?", delivery.OrderID).Update("status", status).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *Service) GetCourierDeliveries(courierID uint) ([]pharmacy.CourierDelivery, error) {
	var deliveries []pharmacy.CourierDelivery
	err := database.DB.Where("courier_id = ? AND status != ?", courierID, "completed").Find(&deliveries).Error
	return deliveries, err
}

func (s *Service) GetPatientDeliveries(patientID uint) ([]pharmacy.CourierDelivery, error) {
	var deliveries []pharmacy.CourierDelivery
	// Find all deliveries linked to orders belonging to this patient
	err := database.DB.Table("courier_deliveries").
		Select("courier_deliveries.*").
		Joins("JOIN pharmacy_orders ON pharmacy_orders.id = courier_deliveries.order_id").
		Where("pharmacy_orders.user_id = ? AND courier_deliveries.status NOT IN ?", 
			patientID, []string{"completed", "failed", "cancelled"}).
		Order("courier_deliveries.created_at DESC").
		Find(&deliveries).Error
	return deliveries, err
}

func (s *Service) CreateShipment(shipment *pharmacy.Shipment) error {
	err := database.DB.Create(shipment).Error
	if err != nil {
		return err
	}

	// Record Ledger: Debit Logistics Expense, Credit Courier Partner Liability
	if s.financeSvc != nil {
		// Mock cost for now, in a real scenario this would come from the provider's CalculateCost
		deliveryCost := int64(5000) // 50.00 in paise

		expenseAcct, _ := s.financeSvc.GetOrCreateAccount(context.Background(), "Expense:Logistics:Delivery", finance.TypeAsset)
		courierAcct, _ := s.financeSvc.GetOrCreateAccount(context.Background(), fmt.Sprintf("Liability:CourierPartner:%d", shipment.CourierID), finance.TypeLiability)

		entries := []finance.EntryData{
			{AccountID: expenseAcct.ID, Direction: finance.DirectionDebit, Amount: deliveryCost},
			{AccountID: courierAcct.ID, Direction: finance.DirectionCredit, Amount: deliveryCost},
		}

		_, _ = s.financeSvc.RecordTransaction(context.Background(), "SHIP-"+shipment.TrackingNumber, "LOGISTICS", fmt.Sprintf("Delivery cost for %s", shipment.TrackingNumber), entries)
	}

	return nil
}

func (s *Service) GetShipment(id string) (*pharmacy.Shipment, error) {
	var shipment pharmacy.Shipment
	err := database.DB.Where("id = ? OR tracking_number = ?", id, id).First(&shipment).Error
	return &shipment, err
}

func (s *Service) GetAllShipments() ([]pharmacy.Shipment, error) {
	var shipments []pharmacy.Shipment
	err := database.DB.Find(&shipments).Error
	return shipments, err
}

func (s *Service) AddShipmentUpdate(shipmentID uint, status, location, notes string) error {
	update := &pharmacy.ShipmentUpdate{
		ShipmentID: shipmentID,
		Status:     status,
		Location:   location,
		Notes:      notes,
		Timestamp:  time.Now(),
	}
	
	tx := database.DB.Begin()
	if err := tx.Create(update).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	if err := tx.Model(&pharmacy.Shipment{}).Where("id = ?", shipmentID).Update("status", status).Error; err != nil {
		tx.Rollback()
		return err
	}
	
	return tx.Commit().Error
}

func (s *Service) GetAvailableCouriers() ([]pharmacy.CourierProfile, error) {
	var couriers []pharmacy.CourierProfile
	err := database.DB.Where("is_available = ?", true).Find(&couriers).Error
	return couriers, err
}
