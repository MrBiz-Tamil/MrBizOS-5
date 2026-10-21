// Package zone holds the platform's single geographic primitive (ServiceZone)
// and the pincode resolvers shared across domains. The Fiber HTTP handlers that
// used to live here were removed together with the REST server; the model and
// resolver helpers remain and are consumed directly by migrations and domain
// services.
package zone

import (
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ServiceZone is the platform's single geographic primitive. Hybrid hierarchy:
// a city-level zone (ParentID nil) can hold pincode-level sub-zones (ParentID
// set). Every operational unit — doctors, labs, pharmacies, stores, hospitals,
// franchises — maps to a zone via its zone_id.
type ServiceZone struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Name  string `gorm:"not null" json:"name"` // "Chennai" or "South Chennai"
	City  string `gorm:"index" json:"city"`
	State string `json:"state"`
	// ParentID nil → city-level zone; set → pincode sub-zone of that city.
	ParentID  *uint          `gorm:"index" json:"parent_id,omitempty"`
	Kind      string         `gorm:"default:'subzone'" json:"kind"` // city | subzone
	Pincodes  datatypes.JSON `gorm:"type:jsonb" json:"pincodes"`    // sub-zones carry pincodes
	CenterLat float64        `json:"center_lat"`
	CenterLng float64        `json:"center_lng"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (ServiceZone) TableName() string { return "service_zones" }

// AutoMigrate creates/updates the service_zones table.
func AutoMigrate(db *gorm.DB) error { return db.AutoMigrate(&ServiceZone{}) }

// ResolveByPincode finds the sub-zone whose pincode list contains the given
// pincode. Returns the matched zone and true, or false if no zone covers it.
// Shared across domains so every unit resolves geography the same way.
func ResolveByPincode(db *gorm.DB, pincode string) (ServiceZone, bool) {
	pincode = strings.TrimSpace(pincode)
	if pincode == "" {
		return ServiceZone{}, false
	}
	var z ServiceZone
	err := db.Where("is_active = ? AND pincodes LIKE ?", true, "%\""+pincode+"\"%").First(&z).Error
	if err != nil {
		return ServiceZone{}, false
	}
	return z, true
}

// ResolveIDByPincode is the convenience form returning just the zone id (0 if none).
func ResolveIDByPincode(db *gorm.DB, pincode string) uint {
	if z, ok := ResolveByPincode(db, pincode); ok {
		return z.ID
	}
	return 0
}
