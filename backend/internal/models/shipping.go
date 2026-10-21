package models

import (
	"time"
)

type Shipment struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	TrackingNumber   string    `gorm:"uniqueIndex;not null" json:"tracking_number"`
	OrderID          uint      `gorm:"index" json:"order_id"` // Reference to PharmacyOrder or WellnessOrder
	OrderType        string    `json:"order_type"`           // pharmacy, wellness, retail
	CourierID        uint      `gorm:"index" json:"courier_id"`
	Status           string    `gorm:"default:'pending'" json:"status"` // pending, picked_up, in_transit, delivered, failed
	Origin           string    `json:"origin"`
	Destination      string    `json:"destination"`
	CurrentAddress   string    `json:"current_address"`
	CurrentLat       float64   `json:"current_lat"`
	CurrentLon       float64   `json:"current_lon"`
	EstimatedArrival *time.Time `json:"estimated_arrival"`
	ActualDelivery   *time.Time `json:"actual_delivery"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (Shipment) TableName() string {
	return "shipments"
}

type ShipmentUpdate struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ShipmentID uint      `gorm:"index" json:"shipment_id"`
	Status     string    `json:"status"`
	Location   string    `json:"location"`
	Notes      string    `json:"notes"`
	Timestamp  time.Time `json:"timestamp"`
}

func (ShipmentUpdate) TableName() string {
	return "shipment_updates"
}
