//go:build ignore
// +build ignore

package models

import (
	"time"
)

type DeliveryMode string

const (
	ModeQuick    DeliveryMode = "QUICK"
	ModeStandard DeliveryMode = "STANDARD"
)

type DeliveryZone struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	City           string    `gorm:"uniqueIndex;not null" json:"city"`
	IsQuickEnabled bool      `gorm:"default:true" json:"is_quick_enabled"`
	MaxRadiusKm    float64   `gorm:"default:10.0" json:"max_radius_km"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (DeliveryZone) TableName() string {
	return "delivery_zones"
}
