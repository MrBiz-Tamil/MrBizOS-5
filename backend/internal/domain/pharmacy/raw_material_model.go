package pharmacy

import (
	"time"

	"gorm.io/gorm"
)

// ──────────────────────────────────────────────
// RAW MATERIAL / BULK DRUG CATALOG
// Active Pharmaceutical Ingredients (APIs), excipients, packaging materials
// ──────────────────────────────────────────────

type RawMaterial struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	Name                string    `gorm:"not null" json:"name"`                        // "Paracetamol API"
	CASNumber           string    `gorm:"index" json:"cas_number"`                     // Chemical Abstracts Service number
	GenericName         string    `json:"generic_name"`                                // INN name
	Category            string    `gorm:"index" json:"category"`                       // "api", "excipient", "packaging", "solvent"
	TherapeuticCategory string    `json:"therapeutic_category"`                         // "analgesic", "antidiabetic"
	GradeType           string    `gorm:"not null;default:'IP'" json:"grade_type"`     // "IP" (Indian Pharmacopoeia), "BP", "USP", "EP"
	PhysicalForm        string    `json:"physical_form"`                               // "powder", "crystal", "granule", "liquid", "paste"
	StorageCondition    string    `gorm:"default:'ambient'" json:"storage_condition"`  // "ambient", "cold_chain", "controlled", "hygroscopic"
	RequiresColdChain   bool      `gorm:"default:false" json:"requires_cold_chain"`
	IsControlled        bool      `gorm:"default:false" json:"is_controlled"`          // NDPS Act / Schedule X
	HSNCode             string    `json:"hsn_code"`                                    // GST/customs classification
	DefaultUOM          string    `gorm:"default:'kg'" json:"default_uom"`             // kg, liters, grams
	MinOrderQtyKg       float64   `json:"min_order_qty_kg"`                            // Supplier minimum order
	ShelfLifeMonths     int       `json:"shelf_life_months"`
	IsActive            bool      `gorm:"default:true" json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"deleted_at,omitempty" swaggertype:"string"`
}

func (RawMaterial) TableName() string { return "raw_materials" }

// ──────────────────────────────────────────────
// APPROVED VENDOR LIST (AVL)
// Links suppliers to specific raw materials they're certified to provide
// ──────────────────────────────────────────────

type ApprovedVendor struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	SupplierID      uint       `gorm:"index;not null" json:"supplier_id"`
	Supplier        Supplier   `gorm:"foreignKey:SupplierID" json:"supplier,omitempty"`
	RawMaterialID   uint       `gorm:"index;not null" json:"raw_material_id"`
	RawMaterial     RawMaterial `gorm:"foreignKey:RawMaterialID" json:"raw_material,omitempty"`
	GMPCertExpiry   *time.Time `json:"gmp_cert_expiry"`                               // GMP certificate validity
	GMPCertURL      string     `json:"gmp_cert_url"`
	CoAOnFile       bool       `gorm:"default:false" json:"coa_on_file"`              // Certificate of Analysis available
	CountryOfOrigin string     `json:"country_of_origin"`
	LeadTimeDays    int        `gorm:"default:7" json:"lead_time_days"`
	MinOrderQty     float64    `json:"min_order_qty"`
	UnitOfMeasure   string     `gorm:"default:'kg'" json:"unit_of_measure"`
	PricePerUnit    int64      `json:"price_per_unit"`                                // Paise per kg/unit
	Currency        string     `gorm:"default:'INR'" json:"currency"`
	Rating          float64    `gorm:"default:5.0" json:"rating"`                     // 1-5 vendor rating
	IsPreferred     bool       `gorm:"default:false" json:"is_preferred"`
	IsActive        bool       `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (ApprovedVendor) TableName() string { return "approved_vendors" }

// ──────────────────────────────────────────────
// RAW MATERIAL INVENTORY
// Tracks actual stock of raw materials per warehouse with lot/batch tracking
// ──────────────────────────────────────────────

type RawMaterialInventory struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	RawMaterialID   uint       `gorm:"index;not null" json:"raw_material_id"`
	RawMaterial     RawMaterial `gorm:"foreignKey:RawMaterialID" json:"raw_material,omitempty"`
	WarehouseID     uint       `gorm:"index;not null" json:"warehouse_id"`             // FK → Pharmacy.ID
	LotNumber       string     `gorm:"index;not null" json:"lot_number"`
	BatchNumber     string     `json:"batch_number"`                                   // Supplier's batch
	Quantity        float64    `json:"quantity"`                                        // Current remaining
	OriginalQty     float64    `json:"original_qty"`                                   // As received
	UnitOfMeasure   string     `gorm:"default:'kg'" json:"unit_of_measure"`
	SupplierID      uint       `gorm:"index" json:"supplier_id"`
	CoADocumentURL  string     `json:"coa_document_url"`
	ManufactureDate *time.Time `json:"manufacture_date"`
	ExpiryDate      time.Time  `json:"expiry_date"`
	RetestDate      *time.Time `json:"retest_date"`                                    // Some APIs have retest instead of expiry
	QAStatus        string     `gorm:"default:'quarantine';index" json:"qa_status"`    // quarantine, approved, rejected, retest_due
	QAApprovedByID  *uint      `json:"qa_approved_by_id"`
	QAApprovedAt    *time.Time `json:"qa_approved_at"`
	QANotes         string     `json:"qa_notes"`
	CostPerUnit     int64      `json:"cost_per_unit"`                                  // Paise per kg/unit
	ReorderLevel    float64    `gorm:"default:10.0" json:"reorder_level"`
	IsActive        bool       `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (RawMaterialInventory) TableName() string { return "raw_material_inventory" }

// ──────────────────────────────────────────────
// RAW MATERIAL INDENT
// Demand signal from manufacturing planner requesting raw materials
// ──────────────────────────────────────────────

type RawMaterialIndent struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	IndentNumber      string     `gorm:"uniqueIndex;not null" json:"indent_number"`   // "IND-2026-00034"
	RawMaterialID     uint       `gorm:"index;not null" json:"raw_material_id"`
	RawMaterial       RawMaterial `gorm:"foreignKey:RawMaterialID" json:"raw_material,omitempty"`
	ProductionOrderID *uint      `gorm:"index" json:"production_order_id"`            // Optional: linked to specific production order
	RequiredQty       float64    `json:"required_qty"`
	UnitOfMeasure     string     `gorm:"default:'kg'" json:"unit_of_measure"`
	RequiredByDate    time.Time  `json:"required_by_date"`
	WarehouseID       uint       `gorm:"index;not null" json:"warehouse_id"`          // Which hub needs it
	Status            string     `gorm:"default:'pending';index" json:"status"`       // pending, approved, rfq_sent, po_created, received, cancelled
	Priority          int        `gorm:"default:5" json:"priority"`                    // 1=urgent
	RequestedByID     uint       `json:"requested_by_id"`
	ApprovedByID      *uint      `json:"approved_by_id"`
	ApprovedAt        *time.Time `json:"approved_at"`
	Notes             string     `json:"notes"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (RawMaterialIndent) TableName() string { return "raw_material_indents" }

// ──────────────────────────────────────────────
// REQUEST FOR QUOTATION (RFQ)
// Sent to multiple vendors for price comparison
// ──────────────────────────────────────────────

type RFQ struct {
	ID            uint             `gorm:"primaryKey" json:"id"`
	RFQNumber     string           `gorm:"uniqueIndex;not null" json:"rfq_number"`   // "RFQ-2026-00012"
	RawMaterialID uint             `gorm:"index;not null" json:"raw_material_id"`
	RawMaterial   RawMaterial      `gorm:"foreignKey:RawMaterialID" json:"raw_material,omitempty"`
	IndentID      *uint            `gorm:"index" json:"indent_id"`                    // Optional: linked to indent
	RequiredQty   float64          `json:"required_qty"`
	UnitOfMeasure string           `gorm:"default:'kg'" json:"unit_of_measure"`
	Deadline      time.Time        `json:"deadline"`                                  // Response deadline
	Status        string           `gorm:"default:'open';index" json:"status"`        // open, closed, awarded, cancelled
	CreatedByID   uint             `json:"created_by_id"`
	Notes         string           `json:"notes"`
	Quotations    []VendorQuotation `gorm:"foreignKey:RFQID" json:"quotations,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

func (RFQ) TableName() string { return "rfqs" }

// ──────────────────────────────────────────────
// VENDOR QUOTATION
// Individual vendor responses to an RFQ
// ──────────────────────────────────────────────

type VendorQuotation struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	RFQID         uint       `gorm:"index;not null" json:"rfq_id"`
	SupplierID    uint       `gorm:"index;not null" json:"supplier_id"`
	Supplier      Supplier   `gorm:"foreignKey:SupplierID" json:"supplier,omitempty"`
	UnitPrice     int64      `json:"unit_price"`                                      // Paise per kg/unit
	Currency      string     `gorm:"default:'INR'" json:"currency"`
	LeadTimeDays  int        `json:"lead_time_days"`
	MinOrderQty   float64    `json:"min_order_qty"`
	ValidUntil    time.Time  `json:"valid_until"`
	CoAAvailable  bool       `gorm:"default:false" json:"coa_available"`
	Notes         string     `json:"notes"`
	IsSelected    bool       `gorm:"default:false" json:"is_selected"`                // Winner of the RFQ
	SubmittedAt   *time.Time `json:"submitted_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (VendorQuotation) TableName() string { return "vendor_quotations" }

// ──────────────────────────────────────────────
// RAW MATERIAL TRANSACTION LOG
// Immutable audit trail for every movement of raw materials
// ──────────────────────────────────────────────

type RawMaterialTransaction struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	InventoryID     uint      `gorm:"index;not null" json:"inventory_id"`
	TransactionType string    `gorm:"not null;index" json:"transaction_type"` // grn_receipt, qa_release, manufacturing_issue, return_to_stock, destruction, sampling, adjustment
	Quantity        float64   `json:"quantity"`                               // Positive = in, Negative = out
	BalanceAfter    float64   `json:"balance_after"`                          // Running balance
	UnitOfMeasure   string    `gorm:"default:'kg'" json:"unit_of_measure"`
	ReferenceType   string    `json:"reference_type"`                         // grn, production_order, qa_sample, destruction_record, indent
	ReferenceID     *uint     `json:"reference_id"`
	PerformedByID   uint      `gorm:"not null" json:"performed_by_id"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"` // Immutable — no UpdatedAt
}

func (RawMaterialTransaction) TableName() string { return "raw_material_transactions" }

// ──────────────────────────────────────────────
// GOODS RECEIPT NOTE (GRN)
// Tracks inbound goods (from suppliers, inter-warehouse, manufacturing)
// ──────────────────────────────────────────────

type GoodsReceiptNote struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	GRNNumber       string    `gorm:"uniqueIndex;not null" json:"grn_number"`       // "GRN-2026-00123"
	PurchaseOrderID *uint     `gorm:"index" json:"purchase_order_id"`               // Nullable: internal production has no PO
	WarehouseID     uint      `gorm:"index;not null" json:"warehouse_id"`
	SourceType      string    `gorm:"not null" json:"source_type"`                  // supplier, manufacturing, inter_warehouse, customer_return
	SupplierID      *uint     `json:"supplier_id"`
	ReceivedByID    uint      `gorm:"not null" json:"received_by_id"`
	ReceivedAt      time.Time `json:"received_at"`
	Status          string    `gorm:"default:'pending_inspection';index" json:"status"` // pending_inspection, qa_in_progress, accepted, partially_accepted, rejected
	VehicleNumber   string    `json:"vehicle_number"`
	InvoiceNumber   string    `json:"invoice_number"`
	InvoiceAmount   int64     `json:"invoice_amount"`                               // Paise
	Notes           string    `json:"notes"`
	Items           []GRNItem `gorm:"foreignKey:GRNID" json:"items,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (GoodsReceiptNote) TableName() string { return "goods_receipt_notes" }

type GRNItem struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	GRNID           uint       `gorm:"index;not null" json:"grn_id"`
	ItemType        string     `gorm:"not null" json:"item_type"`                    // medicine, raw_material
	ItemID          uint       `gorm:"not null" json:"item_id"`                      // FK → Medicine.ID or RawMaterial.ID
	ItemName        string     `json:"item_name"`                                    // Denormalized for display
	OrderedQty      float64    `json:"ordered_qty"`
	ReceivedQty     float64    `json:"received_qty"`
	AcceptedQty     float64    `json:"accepted_qty"`                                 // After QA
	RejectedQty     float64    `json:"rejected_qty"`
	UnitOfMeasure   string     `gorm:"default:'units'" json:"unit_of_measure"`
	BatchNumber     string     `json:"batch_number"`
	ExpiryDate      *time.Time `json:"expiry_date"`
	ManufactureDate *time.Time `json:"manufacture_date"`
	UnitCost        int64      `json:"unit_cost"`                                    // Paise
	QAStatus        string     `gorm:"default:'pending'" json:"qa_status"`           // pending, passed, failed, conditional
	QANotes         string     `json:"qa_notes"`
	CoADocumentURL  string     `json:"coa_document_url"`
	ArrivalTempC    *float64   `json:"arrival_temp_c"`                               // For cold chain items
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (GRNItem) TableName() string { return "grn_items" }

// ──────────────────────────────────────────────
// ENHANCED PURCHASE ORDER FIELDS
// Additional fields to extend existing PurchaseOrder for multi-city
// ──────────────────────────────────────────────

type PurchaseOrderExtension struct {
	ID                   uint       `gorm:"primaryKey" json:"id"`
	PurchaseOrderID      uint       `gorm:"uniqueIndex;not null" json:"purchase_order_id"`
	DestinationHubID     *uint      `json:"destination_hub_id"`                         // Which city hub receives the goods
	DestinationCityName  string     `json:"destination_city_name"`
	SupplierAckAt        *time.Time `json:"supplier_ack_at"`                            // Supplier confirmation timestamp
	ExpectedDeliveryDate *time.Time `json:"expected_delivery_date"`                     // Supplier-confirmed ETA
	GRNID                *uint      `json:"grn_id"`                                     // Links to GRN when received
	InvoiceNumber        string     `json:"invoice_number"`                             // For 3-way match
	PaymentStatus        string     `gorm:"default:'pending'" json:"payment_status"`    // pending, matched, paid, disputed
	ApprovalLevel        string     `json:"approval_level"`                             // auto, manager, regional_head, cfo
	ApprovedByID         *uint      `json:"approved_by_id"`
	ApprovedAt           *time.Time `json:"approved_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func (PurchaseOrderExtension) TableName() string { return "purchase_order_extensions" }

// ──────────────────────────────────────────────
// AGGREGATION/DASHBOARD TYPES
// ──────────────────────────────────────────────

// RawMaterialSummary for dashboard display.
type RawMaterialSummary struct {
	RawMaterialID   uint    `json:"raw_material_id"`
	Name            string  `json:"name"`
	Category        string  `json:"category"`
	TotalQty        float64 `json:"total_qty"`
	UnitOfMeasure   string  `json:"unit_of_measure"`
	QAApprovedQty   float64 `json:"qa_approved_qty"`
	QuarantineQty   float64 `json:"quarantine_qty"`
	WarehouseCount  int     `json:"warehouse_count"`
	NearestExpiry   string  `json:"nearest_expiry"`
	LowStock        bool    `json:"low_stock"`
}

// CityWorkflowDashboard is the multi-city modal top-level data structure.
type CityWorkflowDashboard struct {
	CityName       string                  `json:"city_name"`
	WarehouseID    uint                    `json:"warehouse_id"`
	Manufacturing  ProductionDashboard     `json:"manufacturing"`
	Procurement    ProcurementDashboard    `json:"procurement"`
	RawMaterials   RawMaterialDashboard    `json:"raw_materials"`
	InventoryStats InventoryDashboardStats `json:"inventory_stats"`
}

type ProcurementDashboard struct {
	PendingPOs      int64 `json:"pending_pos"`
	AwaitingGRN     int64 `json:"awaiting_grn"`
	ReorderAlerts   int64 `json:"reorder_alerts"`
	TotalPOValue    int64 `json:"total_po_value"`
}

type RawMaterialDashboard struct {
	OpenIndents       int64 `json:"open_indents"`
	RFQsPending       int64 `json:"rfqs_pending"`
	InQALab           int64 `json:"in_qa_lab"`
	TotalMaterials    int64 `json:"total_materials"`
	LowStockMaterials int64 `json:"low_stock_materials"`
}

type InventoryDashboardStats struct {
	TotalSKUs       int64   `json:"total_skus"`
	LowStockItems   int64   `json:"low_stock_items"`
	ExpiringItems   int64   `json:"expiring_items"`
	ColdChainOK     bool    `json:"cold_chain_ok"`
	InventoryValue  float64 `json:"inventory_value"`
}
