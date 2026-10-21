package pharmacy

import (
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/money"
	"drayya-core/internal/utils/privacy"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// DrugInteraction records known interactions between two drugs.
type DrugInteraction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	DrugAName   string    `gorm:"not null;index" json:"drug_a_name"`
	DrugBName   string    `gorm:"not null;index" json:"drug_b_name"`
	Severity    string    `gorm:"not null" json:"severity"` // "minor", "moderate", "major", "contraindicated"
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (DrugInteraction) TableName() string { return "drug_interactions" }

type DeliveryMode string

const (
	ModeQuick    DeliveryMode = "QUICK"
	ModeStandard DeliveryMode = "STANDARD"
)

// CatalogStatus is the publish state of a medicine on the public store.
type CatalogStatus string

const (
	CatalogDraft     CatalogStatus = "draft"          // created/edited, not public
	CatalogPending   CatalogStatus = "pending_review" // submitted, awaiting admin approval
	CatalogPublished CatalogStatus = "published"      // live on the public store
	CatalogArchived  CatalogStatus = "archived"       // delisted from the public store
)

// SalesChannel decides where a medicine may be sold.
type SalesChannel string

const (
	ChannelPublic       SalesChannel = "public"        // online store + counter
	ChannelPharmacyOnly SalesChannel = "pharmacy_only" // counter/POS only, never public
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

type RxValidationLevel string

const (
	RxLevelOTC      RxValidationLevel = "OTC"
	RxLevelStandard RxValidationLevel = "RX_STANDARD" // Basic prescription required
	RxLevelStrict   RxValidationLevel = "RX_STRICT"   // High regulation (e.g., narcotics)
)

type Medicine struct {
	ID                     uint              `gorm:"primaryKey" json:"id"`
	Name                   string            `json:"name"`
	Brand                  string            `json:"brand"`
	// MoleculeCode is the active pharmaceutical ingredient (API) identifier.
	// Medicines sharing the same MoleculeCode are therapeutically equivalent
	// and can be shown as cheaper generic alternatives to each other.
	// Example: "metformin-500", "amoxicillin-500", "omeprazole-20".
	MoleculeCode           string            `gorm:"index" json:"molecule_code"`
	// GenericName is the INN (International Non-proprietary Name) of the drug.
	GenericName            string            `json:"generic_name"`
	// Synonyms holds brand names and common names for smart search.
	Synonyms               string            `json:"synonyms"`
	Category               string            `json:"category"`
	Description            string                `json:"description"`
	Manufacturer           string                `json:"manufacturer"`
	Prices                 money.RegionalPricing `gorm:"type:jsonb" json:"prices"` // Replaces unit_price
	// ImageURL is the primary product photo shown on cards, search, and the
	// detail page hero. ImageURLs holds optional additional gallery photos.
	ImageURL               string                `json:"image_url,omitempty"`
	ImageURLs              datatypes.JSON        `gorm:"type:jsonb" json:"image_urls,omitempty"` // ["https://…", …]
	IsPrescriptionRequired bool                  `json:"is_prescription_required"`
	RxValidationLevel      RxValidationLevel `gorm:"default:'OTC'" json:"rx_validation_level"`
	RequiresColdChain      bool              `gorm:"default:false" json:"requires_cold_chain"`
	IsActive               bool              `gorm:"default:true" json:"is_active"`
	// Ratings — aggregate populated at query time, not stored.
	AvgRating   float32 `gorm:"-" json:"avg_rating"`
	ReviewCount int     `gorm:"-" json:"review_count"`

	// ── AYUSH product master fields (P1 of the pharmacy roadmap) ────────────────
	// The platform's catalog is Siddha + Ayurveda + Wellness — not allopathic. The
	// fields below capture what a Siddha/Ayurveda product needs that a typical
	// allopathic SKU doesn't: traditional names in Tamil/Hindi, herbal
	// composition, classical indications, dosage form, GST/HSN for invoicing,
	// and the AYUSH manufacturing licence number printed on the pack.
	NameTa             string `gorm:"index" json:"name_ta,omitempty"` // Tamil name (Siddha)
	NameHi             string `gorm:"index" json:"name_hi,omitempty"` // Hindi name (Ayurveda)
	DescriptionTa      string `gorm:"type:text" json:"description_ta,omitempty"`
	DescriptionHi      string `gorm:"type:text" json:"description_hi,omitempty"`
	// Form is the dosage form: tablet, churna (powder), kashayam (decoction),
	// tailam (oil), lehyam (paste), bhasma, capsule, syrup, wellness_item.
	Form               string `gorm:"index" json:"form,omitempty"`
	StrengthValue      string `json:"strength_value,omitempty"`         // "500"
	StrengthUnit       string `json:"strength_unit,omitempty"`          // "mg"
	PackSize           string `json:"pack_size,omitempty"`              // "60 tablets" / "100 ml"
	Ingredients        datatypes.JSON `gorm:"type:jsonb" json:"ingredients,omitempty"` // [{name, qty, unit}]
	Indications        string `gorm:"type:text" json:"indications,omitempty"`        // what it treats
	Contraindications  string `gorm:"type:text" json:"contraindications,omitempty"`  // when not to use
	DosageInstructions string `gorm:"type:text" json:"dosage_instructions,omitempty"`
	StorageInstructions string `gorm:"type:text" json:"storage_instructions,omitempty"`
	GSTPercent         float64 `gorm:"default:5" json:"gst_percent"` // 5% standard for AYUSH meds
	HSNCode            string `json:"hsn_code,omitempty"`           // tax classification
	AyushLicenseNo     string `json:"ayush_license_no,omitempty"`   // printed on the pack
	IsInhouseBrand     bool   `gorm:"default:false;index" json:"is_inhouse_brand"`

	// ── Catalog publishing & sales channel ──────────────────────────────────────
	// CatalogStatus gates public-store visibility. New medicines start as "draft";
	// a pharmacy_admin must publish them. No gorm default is set on purpose — the
	// AutoMigrate backfill relies on existing rows landing as empty so they can be
	// distinguished from genuine new drafts and bulk-published once.
	CatalogStatus CatalogStatus `gorm:"index" json:"catalog_status"`
	// SalesChannel decides where a medicine can be sold. "pharmacy_only" items are
	// never shown on the public store/search — counter/POS sale only (Rx-gated).
	SalesChannel SalesChannel `gorm:"index" json:"sales_channel"`
	PublishedAt  *time.Time   `json:"published_at,omitempty"`
	PublishedBy  uint         `json:"published_by,omitempty"`

	Variants               []ProductVariant  `gorm:"foreignKey:MedicineID" json:"variants,omitempty"`
	CreatedAt              time.Time         `json:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at"`
	DeletedAt              gorm.DeletedAt    `json:"deleted_at,omitempty" swaggertype:"string"`
}

func (Medicine) TableName() string {
	return "medicine"
}

// BeforeSave enforces consistency: if a medicine does not require a prescription,
// its RxValidationLevel must be "OTC".
func (m *Medicine) BeforeSave(tx *gorm.DB) error {
	if !m.IsPrescriptionRequired && m.RxValidationLevel != "" && m.RxValidationLevel != "OTC" {
		m.RxValidationLevel = "OTC"
	}
	return nil
}

// BeforeCreate defaults a brand-new medicine to an unpublished draft on the
// public channel. New SKUs are never auto-live — a pharmacy_admin must publish.
func (m *Medicine) BeforeCreate(tx *gorm.DB) error {
	if m.CatalogStatus == "" {
		m.CatalogStatus = CatalogDraft
	}
	if m.SalesChannel == "" {
		m.SalesChannel = ChannelPublic
	}
	return nil
}

type ProductVariant struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	MedicineID         uint      `gorm:"index" json:"medicine_id"`
	SKU                string    `gorm:"uniqueIndex;not null" json:"sku"`
	Strength           string                `json:"strength"` // e.g., "500mg"
	PackSize           string                `json:"pack_size"` // e.g., "Strip of 10"
	Prices             money.RegionalPricing `gorm:"type:jsonb" json:"prices"`
	CompareAtPrices    money.RegionalPricing `gorm:"type:jsonb" json:"compare_at_prices"` // For discounts
	PhysicalWeightGrams int                   `json:"physical_weight_grams"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (ProductVariant) TableName() string {
	return "product_variants"
}

type Prescription struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	UserID          uint           `gorm:"index" json:"user_id"`
	AppointmentID   uint           `gorm:"index" json:"appointment_id,omitempty"`
	OrderID         *uint          `gorm:"index" json:"order_id,omitempty"` // Nullable, linked during checkout
	DoctorID        *uint          `gorm:"index" json:"doctor_id,omitempty"` // If generated internally
	Diagnosis       string         `json:"diagnosis"`
	Symptoms        string         `json:"symptoms"`
	Medicines       datatypes.JSON `json:"medicines"`
	Instructions    string         `json:"instructions"`
	FileURL                string         `json:"file_url"` // Legacy: external URL (kept for back-compat)
	// DocumentID links the uploaded scan in the private vault (preferred over FileURL).
	DocumentID             *uint          `gorm:"index" json:"document_id,omitempty"`
	Status                 string         `gorm:"default:'uploaded'" json:"status"` // uploaded, under_review, verified, rejected, expired
	PharmacistNotes        string         `json:"pharmacist_notes,omitempty"`
	VerifiedByPharmacistID *uint          `json:"verified_by_pharmacist_id,omitempty" gorm:"index"`
	VerifiedAt             *time.Time     `json:"verified_at,omitempty"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
}

func (Prescription) TableName() string {
	return "prescriptions"
}

type Pharmacy struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	Name               string    `json:"name"`
	Address            string    `json:"address"`
	City               string    `json:"city"`
	Pincode            string    `json:"pincode"`
	ZoneID             *uint     `gorm:"index" json:"zone_id,omitempty"` // unified geographic zone
	Region             string    `json:"region"` // State or wider area
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
	PharmacyID    uint      `gorm:"index:idx_pharmacy_inv_lookup,priority:1" json:"pharmacy_id"`
	MedicineID    uint      `gorm:"index:idx_pharmacy_inv_lookup,priority:2" json:"medicine_id"`
	BatchNumber   string    `json:"batch_number"`
	StockQuantity int       `json:"stock_quantity"`
	ReorderLevel  int       `json:"reorder_level"`
	ExpiryDate    time.Time `json:"expiry_date"`
	IsActive      bool      `gorm:"index:idx_pharmacy_inv_lookup,priority:3;default:true" json:"is_active"`

	// Relations
	Medicine Medicine `gorm:"foreignKey:MedicineID"`
}

func (PharmacyInventory) TableName() string {
	return "pharmacyinventory"
}

type PharmacyOrder struct {
	ID                  uint                    `gorm:"primaryKey" json:"id"`
	OrderNumber         string                  `gorm:"uniqueIndex" json:"order_number"`
	UserID              uint                    `gorm:"index" json:"user_id"`
	PharmacyID          uint                    `gorm:"index" json:"pharmacy_id"`
	Pharmacy            Pharmacy                `gorm:"foreignKey:PharmacyID" json:"pharmacy,omitempty"`
	ParentOrderID       uint                    `gorm:"index" json:"parent_order_id"`
	Items               datatypes.JSON          `json:"items"` // JSON [{"medicine_id": 1, "quantity": 2}]
	Amount              int64                   `json:"amount"` // Minor units
	Currency            string                  `gorm:"default:'INR'" json:"currency"`
	Status              string                  `json:"status"` // pending, processing, packed, shipped, delivered, cancelled
	PaymentStatus       string                  `json:"payment_status"`
	PaymentMethod       string                  `json:"payment_method"` // cod, online, wallet
	DeliveryMode        DeliveryMode            `json:"delivery_mode"`
	TrackingID          string                  `json:"tracking_id"`
	ShippingAddress     privacy.EncryptedString `json:"shipping_address"`
	DeliveryPhone       privacy.EncryptedString `json:"delivery_phone"`
	DestPincode         string                  `json:"dest_pincode"`
	DestLatitude        float64                 `json:"dest_latitude"`   // patient delivery coordinates
	DestLongitude       float64                 `json:"dest_longitude"`
	SLADeadline         *time.Time              `gorm:"index" json:"sla_deadline"`    // must be delivered by this time
	RequiresColdChain   bool                    `gorm:"default:false" json:"requires_cold_chain"` // set true if any item needs refrigeration
	LedgerTransactionID *uuid.UUID              `json:"ledger_transaction_id"`
	PayoutStatus        string                  `gorm:"default:'pending'" json:"payout_status,omitempty"`
	DeliveryOTP         string                  `gorm:"size:6" json:"-"` // 6-digit OTP for delivery confirmation
	CreatedAt           time.Time               `json:"created_at"`
	UpdatedAt           time.Time               `json:"updated_at"`
}

func (PharmacyOrder) TableName() string {
	return "pharmacy_orders"
}

type PharmacyOrderItem struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	OrderID      uint   `gorm:"index" json:"order_id"`
	MedicineName string `json:"medicine_name"`
	// MedicineID and ExpiryDate landed in P6: required for pharmacovigilance.
	// When a batch is recalled (regulator notice / quality issue), we need to
	// pull every patient who received it — that's the recall query in
	// pharmacy.LookupOrderItemsByBatch.
	MedicineID   uint       `gorm:"index" json:"medicine_id"`
	Quantity     int        `json:"quantity"`
	UnitPrice    int64      `json:"unit_price"`
	TotalPrice   int64      `json:"total_price"`
	BatchNumber  string     `gorm:"index" json:"batch_number"`
	ExpiryDate   *time.Time `json:"expiry_date,omitempty"`
	PhysicalWeightGrams int  `json:"physical_weight_grams"`
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
	Amount        int64          `json:"amount"` // Replaces TotalCost
	Currency      string         `gorm:"default:'INR'" json:"currency"`
	TaxAmount     int64          `json:"tax_amount"`
	TotalWithTax  int64          `json:"total_with_tax"`
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
	ID                    uint                    `gorm:"primaryKey" json:"id"`
	OrderID               uint                    `gorm:"index" json:"order_id"`   // ID of the PharmacyOrder or WellnessOrder
	CourierID             uint                    `gorm:"index" json:"courier_id"` // UserID representing the courier
	Status                string                  `gorm:"index" json:"status"`     // assigned, picked_up, en_route, arrived, completed, failed
	CurrentLatitude       float64                 `json:"current_latitude"`
	CurrentLongitude      float64                 `json:"current_longitude"`
	PickupLatitude        float64                 `json:"pickup_latitude"`   // pharmacy lat
	PickupLongitude       float64                 `json:"pickup_longitude"`  // pharmacy lng
	DestLatitude          float64                 `json:"dest_latitude"`     // patient delivery lat
	DestLongitude         float64                 `json:"dest_longitude"`    // patient delivery lng
	EstimatedDeliveryTime time.Time               `json:"estimated_delivery_time"`
	ActualDeliveryTime    *time.Time              `json:"actual_delivery_time"`
	SLADeadline           *time.Time              `gorm:"index" json:"sla_deadline"`   // hard cutoff; breaching triggers auto-escalation
	SLABreachedAt         *time.Time              `json:"sla_breached_at"`
	BatchID               string                  `gorm:"index" json:"batch_id"`       // groups multiple stops for one courier trip
	DeliveryNotes         privacy.EncryptedString `json:"delivery_notes"`
	ProviderName          string                  `json:"provider_name"`               // e.g. "Shiprocket", "Delhivery"
	TrackingID            string                  `json:"tracking_id"`                 // AWB from provider
	ProofPhotoURL         string                  `json:"proof_photo_url"`             // POD image URL
	CurrentAddress        privacy.EncryptedString `json:"current_address"`
	EstimatedArrival      *time.Time              `json:"estimated_arrival"`
	DistanceTraveled      float64                 `json:"distance_traveled"` // in KM
	BatteryLevel          int                     `json:"battery_level"`     // device battery %
	CreatedAt             time.Time               `json:"created_at"`
	UpdatedAt             time.Time               `json:"updated_at"`
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
	ID              uint          `gorm:"primaryKey" json:"id"`
	UserID          uint          `gorm:"uniqueIndex" json:"user_id"`
	User            identity.User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	VehicleType     string        `json:"vehicle_type"` // bike, car, van, e-bike
	LicenseNumber   string        `json:"license_number"`
	Rating          float64       `gorm:"default:5.0" json:"rating"`
	TotalTrips      int           `gorm:"default:0" json:"total_trips"`
	IsAvailable     bool          `gorm:"default:true" json:"is_available"`
	ActiveDeliveries int          `gorm:"default:0" json:"active_deliveries"` // current in-flight count
	MaxDailyTrips   int           `gorm:"default:50" json:"max_daily_trips"`  // daily cap
	TodayTrips      int           `gorm:"default:0" json:"today_trips"`
	LastLatitude    float64       `json:"last_latitude"`   // last known position (updated on ping)
	LastLongitude   float64       `json:"last_longitude"`
	City            string        `gorm:"index" json:"city"` // home zone city
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

func (CourierProfile) TableName() string {
	return "courier_profiles"
}

type Shipment struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	TrackingNumber   string     `gorm:"uniqueIndex;not null" json:"tracking_number"`
	OrderID          uint       `gorm:"index" json:"order_id"` // Reference to PharmacyOrder or WellnessOrder
	OrderType        string     `json:"order_type"`            // pharmacy, wellness, retail
	CourierID        uint       `gorm:"index" json:"courier_id"`
	ProviderName     string     `json:"provider_name"`                   // e.g. "Shiprocket"
	Status           string     `gorm:"default:'pending'" json:"status"` // pending, picked_up, in_transit, delivered, failed
	Origin           string     `json:"origin"`
	Destination      string     `json:"destination"`
	CurrentAddress   string     `json:"current_address"`
	CurrentLat       float64    `json:"current_lat"`
	CurrentLon       float64    `json:"current_lon"`
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

// --- Ecommerce Analytics Tracking Models ---

// ProductInteraction logs every user interaction with a medicine/product
type ProductInteraction struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	MedicineID uint      `gorm:"index" json:"medicine_id"`
	Medicine   Medicine  `gorm:"foreignKey:MedicineID" json:"medicine,omitempty"`
	UserID     *uint     `gorm:"index" json:"user_id,omitempty"` // Nullable for guests
	SessionID  string    `gorm:"index" json:"session_id"`
	EventType  string    `gorm:"index" json:"event_type"` // view, click, add_to_cart, remove_from_cart, purchase
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

func (ProductInteraction) TableName() string {
	return "product_interactions"
}

// ProductSalesSummary is a materialized view or aggregated table for fast dashboard queries
type ProductSalesSummary struct {
	MedicineID         uint      `gorm:"primaryKey" json:"medicine_id"`
	TotalViews         int       `gorm:"default:0" json:"total_views"`
	TotalAddsToCart    int       `gorm:"default:0" json:"total_adds_to_cart"`
	TotalPurchases     int       `gorm:"default:0" json:"total_purchases"`
	TotalRevenue       int64     `gorm:"default:0" json:"total_revenue"`
	WeekOverWeekGrowth float64   `gorm:"default:0.0" json:"week_over_week_growth"` // e.g. +15.5 for 15.5%
	LastUpdatedAt      time.Time `json:"last_updated_at"`
}

func (ProductSalesSummary) TableName() string {
	return "product_sales_summaries"
}

// CartSession tracks persistent shopping carts for abandoned cart recovery
type CartSession struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        *uint     `gorm:"index" json:"user_id,omitempty"` // For logged in users
	SessionID     string    `gorm:"uniqueIndex" json:"session_id"`  // Ties back to interactions
	CartData      string    `json:"cart_data"`                      // JSON snapshot of the cart
	CartValue     int64     `json:"cart_value"`
	Status        string    `gorm:"index" json:"status"` // active, abandoned, recovered, completed
	LastActiveAt  time.Time `gorm:"index" json:"last_active_at"`
	RecoveryEmailSentAt *time.Time `json:"recovery_email_sent_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (CartSession) TableName() string {
	return "cart_sessions"
}

