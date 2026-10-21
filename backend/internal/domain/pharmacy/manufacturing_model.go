package pharmacy

import (
	"time"

	"gorm.io/gorm"
)

// ──────────────────────────────────────────────
// BILL OF MATERIALS (BOM)
// Links finished medicines to their raw material ingredients
// ──────────────────────────────────────────────

type BillOfMaterials struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	MedicineID uint      `gorm:"index;not null" json:"medicine_id"`
	Medicine   Medicine  `gorm:"foreignKey:MedicineID" json:"medicine,omitempty"`
	VariantID  *uint     `json:"variant_id"`
	Version    int       `gorm:"default:1" json:"version"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	Notes      string    `json:"notes"`
	Items      []BOMItem `gorm:"foreignKey:BOMID" json:"items,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (BillOfMaterials) TableName() string { return "bill_of_materials" }

type BOMItem struct {
	ID             uint        `gorm:"primaryKey" json:"id"`
	BOMID          uint        `gorm:"index;not null" json:"bom_id"`
	RawMaterialID  uint        `gorm:"index;not null" json:"raw_material_id"`
	RawMaterial    RawMaterial `gorm:"foreignKey:RawMaterialID" json:"raw_material,omitempty"`
	QtyPerUnit     float64     `json:"qty_per_unit"`      // grams of raw material per finished unit
	UnitOfMeasure  string      `gorm:"default:'grams'" json:"unit_of_measure"`
	WastagePercent float64     `gorm:"default:2.0" json:"wastage_percent"` // Expected yield loss %
	IsOptional     bool        `gorm:"default:false" json:"is_optional"`
}

func (BOMItem) TableName() string { return "bom_items" }

// ──────────────────────────────────────────────
// PRODUCTION ORDER
// Tracks manufacturing runs from scheduling through QA release
// ──────────────────────────────────────────────

// ProductionStatus represents the manufacturing state machine:
// draft → scheduled → material_reserved → in_production → qa_pending → qa_passed → released
//
//	                                                    └→ qa_failed → quarantine
type ProductionStatus string

const (
	ProdStatusDraft            ProductionStatus = "draft"
	ProdStatusScheduled        ProductionStatus = "scheduled"
	ProdStatusMaterialReserved ProductionStatus = "material_reserved"
	ProdStatusInProduction     ProductionStatus = "in_production"
	ProdStatusQAPending        ProductionStatus = "qa_pending"
	ProdStatusQAPassed         ProductionStatus = "qa_passed"
	ProdStatusQAFailed         ProductionStatus = "qa_failed"
	ProdStatusReleased         ProductionStatus = "released"
	ProdStatusCancelled        ProductionStatus = "cancelled"
)

// AllowedProductionTransitions defines valid state machine transitions.
var AllowedProductionTransitions = map[ProductionStatus][]ProductionStatus{
	ProdStatusDraft:            {ProdStatusScheduled, ProdStatusCancelled},
	ProdStatusScheduled:        {ProdStatusMaterialReserved, ProdStatusCancelled},
	ProdStatusMaterialReserved: {ProdStatusInProduction, ProdStatusCancelled},
	ProdStatusInProduction:     {ProdStatusQAPending, ProdStatusCancelled},
	ProdStatusQAPending:        {ProdStatusQAPassed, ProdStatusQAFailed},
	ProdStatusQAPassed:         {ProdStatusReleased},
	ProdStatusQAFailed:         {ProdStatusDraft}, // Can retry
}

type ProductionOrder struct {
	ID              uint             `gorm:"primaryKey" json:"id"`
	PONumber        string           `gorm:"uniqueIndex;not null" json:"po_number"` // "MFG-2026-00045"
	MedicineID      uint             `gorm:"index;not null" json:"medicine_id"`
	Medicine        Medicine         `gorm:"foreignKey:MedicineID" json:"medicine,omitempty"`
	BOMID           uint             `gorm:"index;not null" json:"bom_id"`
	BOM             BillOfMaterials  `gorm:"foreignKey:BOMID" json:"bom,omitempty"`
	TargetQty       int              `json:"target_qty"`       // Units to produce
	ActualYieldQty  *int             `json:"actual_yield_qty"` // Actual produced after manufacturing
	WarehouseID     uint             `gorm:"index;not null" json:"warehouse_id"` // Where manufacturing happens
	Warehouse       Pharmacy         `gorm:"foreignKey:WarehouseID" json:"warehouse,omitempty"`
	DestCityID      *uint            `json:"dest_city_id"`         // Target city for distribution
	DestCityName    string           `json:"dest_city_name"`       // Denormalized for display
	BatchNumber     string           `json:"batch_number"`         // Generated batch for finished goods
	Status          ProductionStatus `gorm:"default:'draft';index" json:"status"`
	Priority        int              `gorm:"default:5" json:"priority"` // 1=urgent, 10=low
	ScheduledDate   *time.Time       `json:"scheduled_date"`
	StartedAt       *time.Time       `json:"started_at"`
	CompletedAt     *time.Time       `json:"completed_at"`
	QAInspectorID   *uint            `json:"qa_inspector_id"`
	QAResult        string           `json:"qa_result"` // pass, fail, conditional
	QANotes         string           `json:"qa_notes"`
	QACompletedAt   *time.Time       `json:"qa_completed_at"`
	CreatedByID     uint             `json:"created_by_id"`
	Notes           string           `json:"notes"`
	Issuances       []MaterialIssuance `gorm:"foreignKey:ProductionOrderID" json:"issuances,omitempty"`
	Logs            []ManufacturingLog `gorm:"foreignKey:ProductionOrderID" json:"logs,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

func (ProductionOrder) TableName() string { return "production_orders" }

// IsTransitionAllowed checks if a status transition is valid.
func (p *ProductionOrder) IsTransitionAllowed(newStatus ProductionStatus) bool {
	allowed, exists := AllowedProductionTransitions[p.Status]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == newStatus {
			return true
		}
	}
	return false
}

// ──────────────────────────────────────────────
// MANUFACTURING LOG
// Audit trail for every stage of production
// ──────────────────────────────────────────────

type ManufacturingLog struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	ProductionOrderID uint      `gorm:"index;not null" json:"production_order_id"`
	Stage             string    `gorm:"not null" json:"stage"` // mixing, granulation, compression, coating, packaging, qa_sampling
	OperatorID        uint      `json:"operator_id"`
	TemperatureC      *float64  `json:"temperature_c"`
	HumidityPct       *float64  `json:"humidity_pct"`
	Notes             string    `json:"notes"`
	StartedAt         time.Time `json:"started_at"`
	CompletedAt       *time.Time `json:"completed_at"`
	CreatedAt         time.Time `json:"created_at"`
}

func (ManufacturingLog) TableName() string { return "manufacturing_logs" }

// ──────────────────────────────────────────────
// MATERIAL ISSUANCE
// Tracks raw materials pulled from inventory for production
// ──────────────────────────────────────────────

type MaterialIssuance struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	ProductionOrderID uint       `gorm:"index;not null" json:"production_order_id"`
	RawMaterialID     uint       `gorm:"index;not null" json:"raw_material_id"`
	RawMaterial       RawMaterial `gorm:"foreignKey:RawMaterialID" json:"raw_material,omitempty"`
	InventoryID       uint       `gorm:"index;not null" json:"inventory_id"` // Which specific lot
	IssuedQty         float64    `json:"issued_qty"`
	UnitOfMeasure     string     `gorm:"default:'grams'" json:"unit_of_measure"`
	ReturnedQty       float64    `gorm:"default:0" json:"returned_qty"` // If excess returned
	IssuedByID        uint       `json:"issued_by_id"`
	IssuedAt          time.Time  `json:"issued_at"`
	ReturnedAt        *time.Time `json:"returned_at"`
	CreatedAt         time.Time  `json:"created_at"`
}

func (MaterialIssuance) TableName() string { return "material_issuances" }

// ──────────────────────────────────────────────
// FINISHED GOODS BATCH
// Created when a production order passes QA and is released
// ──────────────────────────────────────────────

type FinishedGoodsBatch struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	ProductionOrderID uint      `gorm:"index;not null" json:"production_order_id"`
	MedicineID        uint      `gorm:"index;not null" json:"medicine_id"`
	BatchNumber       string    `gorm:"index;not null" json:"batch_number"`
	QtyProduced       int       `json:"qty_produced"`
	ExpiryDate        time.Time `json:"expiry_date"`
	ManufactureDate   time.Time `json:"manufacture_date"`
	HubID             uint      `gorm:"index" json:"hub_id"` // Which warehouse it's stored in
	Status            string    `gorm:"default:'available'" json:"status"` // available, distributed, recalled, expired
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at,omitempty" swaggertype:"string"`
}

func (FinishedGoodsBatch) TableName() string { return "finished_goods_batches" }

// ──────────────────────────────────────────────
// HELPER TYPES
// ──────────────────────────────────────────────

// MaterialShortage is returned when BOM validation finds insufficient raw material.
type MaterialShortage struct {
	RawMaterialID   uint    `json:"raw_material_id"`
	RawMaterialName string  `json:"raw_material_name"`
	RequiredQty     float64 `json:"required_qty"`
	AvailableQty    float64 `json:"available_qty"`
	ShortageQty     float64 `json:"shortage_qty"`
	UnitOfMeasure   string  `json:"unit_of_measure"`
}

// ProductionDashboard holds aggregated stats for the manufacturing dashboard.
type ProductionDashboard struct {
	ActiveOrders    int64   `json:"active_orders"`
	InQAReview      int64   `json:"in_qa_review"`
	PlannedThisWeek int64   `json:"planned_this_week"`
	ReleasedToday   int64   `json:"released_today"`
	TotalYieldToday int     `json:"total_yield_today"`
	AvgYieldPct     float64 `json:"avg_yield_pct"`
}
