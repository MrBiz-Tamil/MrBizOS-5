package main

import (
	"log"
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/domain/pharmacy"
	"time"

	"gorm.io/gorm"
)

func seedDispatch(db *gorm.DB) {
	// Courier profiles
	var couriers []identity.User
	db.Where("email LIKE ?", "courier%").Find(&couriers)

	for _, c := range couriers {
		var existing pharmacy.CourierProfile
		if db.Where("user_id = ?", c.ID).First(&existing).Error == nil {
			continue
		}
		db.Create(&pharmacy.CourierProfile{
			UserID:        c.ID,
			VehicleType:   "bike",
			LicenseNumber: "MH01AB1234",
			Rating:        4.7,
			TotalTrips:    45,
			IsAvailable:   true,
		})
	}

	// Courier deliveries for existing pharmacy orders
	var orders []pharmacy.PharmacyOrder
	db.Where("status IN ?", []string{"shipped", "delivered"}).Find(&orders)
	for _, order := range orders {
		if len(couriers) == 0 {
			break
		}
		var existing pharmacy.CourierDelivery
		if db.Where("order_id = ?", order.ID).First(&existing).Error == nil {
			continue
		}
		now := time.Now()
		est := now.Add(30 * time.Minute)
		cd := pharmacy.CourierDelivery{
			OrderID:               order.ID,
			CourierID:             couriers[0].ID,
			Status:                "completed",
			CurrentLatitude:       19.0760,
			CurrentLongitude:      72.8777,
			EstimatedDeliveryTime: est,
			DistanceTraveled:      5.2,
			BatteryLevel:          85,
		}
		if order.Status == "shipped" {
			cd.Status = "en_route"
		} else {
			delivered := now.Add(-1 * time.Hour)
			cd.ActualDeliveryTime = &delivered
		}
		db.Create(&cd)
	}

	log.Printf("       Created courier profiles and delivery records")
}

