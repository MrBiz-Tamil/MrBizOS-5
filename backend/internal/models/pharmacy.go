//go:build ignore
// +build ignore

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Medicine struct {
	ID                     uint   `gorm:"primaryKey" json:"id"`
	Name                   string `json:"name"`
	Brand                  string `json:"brand"`
	Category               string `json:"category"`
	Description            string `json:"description"`
	Manufacturer           string `json:"manufacturer"`
	UnitPrice              int64  `json:"unit_price"`
	IsPrescriptionRequired bool   `json:"is_prescription_required"`
	RequiresColdChain      bool   `gorm:"default:false" json:"requires_cold_chain"`
}

func (Medicine) TableName() string {
	return "medicine"
}

type Pharmacy struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Name               string    `json:"name"`
	Address            string    `json:"address"`
	City               string    `json:"city"`
	Latitude           float64   `json:"latitude"`
	Longitude          float64   `json:"longitude"`
	IsCentralWarehouse bool      `gorm:"default:false" json:"is_central_warehouse"`
	HubTier            string    `gorm:"default:'local'" json:"hub_tier"` // global, regional, local
	ParentHubID        *uint     `json:"parent_hub_id"`
	ServiceRadius      float64   `gorm:"default:10.0" json:"service_radius"` // KM
	IsActive           bool      `gorm:"default:true" json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (Pharmacy) TableName() string {
	return "pharmacy"
}

type PharmacyInventory struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	PharmacyID    uint      `json:"pharmacy_id"`
	MedicineID    uint      `json:"medicine_id"`
	BatchNumber   string    `json:"batch_number"`
	StockQuantity int       `json:"stock_quantity"`
	ReorderLevel  int       `json:"reorder_level"`
	ExpiryDate    time.Time `json:"expiry_date"`
	IsActive      bool      `gorm:"default:true" json:"is_active"`

	// Relations
	Medicine Medicine `gorm:"foreignKey:MedicineID"`
}

func (PharmacyInventory) TableName() string {
	return "pharmacyinventory"
}

// Added PharmacyOrder to match Admin Dashboard requirements
type PharmacyOrder struct {
	ID                  uint           `gorm:"primaryKey" json:"id"`
	OrderNumber         string         `gorm:"uniqueIndex" json:"order_number"`
	UserID              uint           `gorm:"index" json:"user_id"`
	PharmacyID          uint           `gorm:"index" json:"pharmacy_id"`
	Pharmacy            Pharmacy       `gorm:"foreignKey:PharmacyID" json:"pharmacy,omitempty"`
	ParentOrderID       uint           `gorm:"index" json:"parent_order_id"`
	Items               datatypes.JSON `json:"items"` // JSON [{"medicine_id": 1, "quantity": 2}]
	TotalAmount         int64          `json:"total_amount"`
	Status              string         `json:"status"` // pending, processing, shipped, delivered, cancelled
	PaymentStatus       string         `json:"payment_status"`
	DeliveryMode        DeliveryMode   `json:"delivery_mode"`
	TrackingID          string         `json:"tracking_id"`
	ShippingAddress     string         `json:"shipping_address"`
	DeliveryPhone       string         `json:"delivery_phone"`
	LedgerTransactionID *uuid.UUID     `json:"ledger_transaction_id"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

func (PharmacyOrder) TableName() string {
	return "pharmacy_orders"
}

type PharmacyOrderItem struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	OrderID      uint   `gorm:"index" json:"order_id"`
	MedicineName string `json:"medicine_name"`
	Quantity     int    `json:"quantity"`
	UnitPrice    int64  `json:"unit_price"`
	TotalPrice   int64  `json:"total_price"`
	BatchNumber  string `json:"batch_number"`
}

func (PharmacyOrderItem) TableName() string {
	return "pharmacy_order_items"
}

type InventoryTransfer struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	SourceID      uint      `gorm:"index" json:"source_id"`      // Central Warehouse
	DestinationID uint      `gorm:"index" json:"destination_id"` // City Pharmacy
	MedicineID    uint      `gorm:"index" json:"medicine_id"`
	Quantity      int       `json:"quantity"`
	Status        string    `json:"status"` // requested, in_transit, completed, cancelled
	RequestedByID uint      `json:"requested_by_id"`
	ProcessedByID *uint     `json:"processed_by_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (InventoryTransfer) TableName() string {
	return "pharmacy_inventory_transfers"
}

// Supplier represents an external vendor who provides medicines to the Central Warehouse
type Supplier struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `json:"name"`
	ContactEmail string    `json:"contact_email"`
	Phone        string    `json:"phone"`
	Address      string    `json:"address"`
	IsActive     bool      `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (Supplier) TableName() string {
	return "pharmacy_suppliers"
}

// PurchaseOrder represents an order placed by the Central Warehouse to an external Supplier
type PurchaseOrder struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	PONumber      string         `gorm:"uniqueIndex" json:"po_number"`
	SupplierID    uint           `gorm:"index" json:"supplier_id"`
	PharmacyID    uint           `gorm:"index" json:"pharmacy_id"` // Must be a Central Warehouse
	Items         datatypes.JSON `json:"items"`                    // JSON [{"medicine_id": 1, "qty": 500, "unit_cost": 10.5}]
	TotalCost     int64          `json:"total_cost"`
	Status        string         `json:"status"` // draft, ordered, received, cancelled
	ExpectedDate  time.Time      `json:"expected_date"`
	ReceivedDate  *time.Time     `json:"received_date"`
	IsAutoTrigger bool           `gorm:"default:false" json:"is_auto_trigger"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (PurchaseOrder) TableName() string {
	return "pharmacy_purchase_orders"
}

// CourierDelivery represents the live dispatch and tracking of an order
type CourierDelivery struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	OrderID               uint       `gorm:"index" json:"order_id"`   // ID of the PharmacyOrder or WellnessOrder
	CourierID             uint       `gorm:"index" json:"courier_id"` // UserID representing the courier
	Status                string     `json:"status"`                  // assigned, picked_up, en_route, arrived, completed, failed
	CurrentLatitude       float64    `json:"current_latitude"`
	CurrentLongitude      float64    `json:"current_longitude"`
	EstimatedDeliveryTime time.Time  `json:"estimated_delivery_time"`
	ActualDeliveryTime    *time.Time `json:"actual_delivery_time"`
	DeliveryNotes         string     `json:"delivery_notes"`
	CurrentAddress        string     `json:"current_address"`
	EstimatedArrival      *time.Time `json:"estimated_arrival"`
	DistanceTraveled      float64    `json:"distance_traveled"` // in KM
	BatteryLevel          int        `json:"battery_level"`     // device battery %
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (CourierDelivery) TableName() string {
	return "courier_deliveries"
}

type ColdChainMetric struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	WarehouseID uint      `gorm:"index" json:"warehouse_id"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	RecordedAt  time.Time `json:"recorded_at"`
}

func (ColdChainMetric) TableName() string {
	return "cold_chain_metrics"
}

type CourierProfile struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"uniqueIndex" json:"user_id"`
	User          User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	VehicleType   string    `json:"vehicle_type"` // e.g. bike, car, van
	LicenseNumber string    `json:"license_number"`
	Rating        float64   `gorm:"default:5.0" json:"rating"`
	TotalTrips    int       `gorm:"default:0" json:"total_trips"`
	IsAvailable   bool      `gorm:"default:true" json:"is_available"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (CourierProfile) TableName() string {
	return "courier_profiles"
}
