package lab

import (
	"drayya-core/internal/domain/identity"
	"time"
)

// ──────────────────────────────────────────────────────────────────────────────
// Test catalog
// ──────────────────────────────────────────────────────────────────────────────

// LabTest is a single diagnostic test or a bundled health package.
type LabTest struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Name              string    `gorm:"not null" json:"name"`
	Category          string    `json:"category"`        // Hematology, Biochemistry, etc.
	HealthConcern     string    `json:"health_concern"`  // Diabetes, Thyroid, Heart, etc.
	Description       string    `json:"description"`
	SampleType        string    `json:"sample_type"`     // blood, urine, stool, multiple
	FastingRequired   bool      `gorm:"default:false" json:"fasting_required"`
	ReportHours       int       `gorm:"default:24" json:"report_hours"`
	ReferenceRange    string    `json:"reference_range"`
	Unit              string    `json:"unit"`
	BasePrice         int64     `json:"base_price"`           // paise (MRP / default price)
	OriginalPricePaise int64    `json:"original_price_paise"` // MRP before platform discount
	DiscountPct       int       `gorm:"default:0" json:"discount_pct"`
	IsPackage         bool      `gorm:"default:false" json:"is_package"`
	IsPopular         bool      `gorm:"default:false" json:"is_popular"`
	Icon              string    `json:"icon"` // emoji
	IsActive          bool      `gorm:"default:true" json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (LabTest) TableName() string { return "lab_tests" }

// LabPackageItem maps a package test to its component individual tests.
type LabPackageItem struct {
	PackageTestID   uint    `gorm:"primaryKey;index" json:"package_test_id"`
	ComponentTestID uint    `gorm:"primaryKey;index" json:"component_test_id"`
	ComponentTest   LabTest `gorm:"foreignKey:ComponentTestID" json:"component_test,omitempty"`
}

func (LabPackageItem) TableName() string { return "lab_package_items" }

// ──────────────────────────────────────────────────────────────────────────────
// Partner network
// ──────────────────────────────────────────────────────────────────────────────

// LabPartner is the diagnostic chain brand (Thyrocare, DRTC, SRL, etc.)
type LabPartner struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null;uniqueIndex" json:"name"`
	Slug        string    `gorm:"uniqueIndex" json:"slug"`
	LogoURL     string    `json:"logo_url"`
	IsNABL      bool      `gorm:"default:true" json:"is_nabl"`
	IsCAP       bool      `json:"is_cap"`
	Rating      float32   `gorm:"default:4.5" json:"rating"`
	Description string    `json:"description"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

func (LabPartner) TableName() string { return "lab_partners" }

// LabCenter is an individual collection point belonging to a partner.
type LabCenter struct {
	ID               uint        `gorm:"primaryKey" json:"id"`
	PartnerID        uint        `gorm:"index" json:"partner_id"`
	Partner          *LabPartner `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	Name             string      `json:"name"`
	Address          string      `json:"address"`
	Pincode          string      `gorm:"index" json:"pincode"`
	ZoneID           *uint       `gorm:"index" json:"zone_id,omitempty"` // unified geographic zone
	District         string      `gorm:"index" json:"district"`
	Taluk            string      `json:"taluk"`
	Lat              *float64    `json:"lat,omitempty"`
	Lng              *float64    `json:"lng,omitempty"`
	Phone            string      `json:"phone"`
	OffersHomePickup bool        `gorm:"default:true" json:"offers_home_pickup"`
	OffersLabVisit   bool        `gorm:"default:true" json:"offers_lab_visit"`
	OpeningTime      string      `gorm:"default:'07:00'" json:"opening_time"`
	ClosingTime      string      `gorm:"default:'20:00'" json:"closing_time"`
	IsActive         bool        `gorm:"default:true" json:"is_active"`
	CreatedAt        time.Time   `json:"created_at"`
}

func (LabCenter) TableName() string { return "lab_centers" }

// LabCenterPricing links a lab center to a test with its specific pricing.
type LabCenterPricing struct {
	ID                    uint `gorm:"primaryKey" json:"id"`
	LabCenterID           uint `gorm:"index;not null" json:"lab_center_id"`
	LabTestID             uint `gorm:"index;not null" json:"lab_test_id"`
	PricePaise            int64 `json:"price_paise"`
	HomePickupChargePaise int64 `json:"home_pickup_charge_paise"`
	TurnaroundHours       int   `gorm:"default:24" json:"turnaround_hours"`
	IsAvailable           bool  `gorm:"default:true" json:"is_available"`
}

func (LabCenterPricing) TableName() string { return "lab_center_pricing" }

// ──────────────────────────────────────────────────────────────────────────────
// Patient bookings
// ──────────────────────────────────────────────────────────────────────────────

// LabBooking is a patient's appointment to get a test done at a partner center.
// Status flow: pending → confirmed → sample_collected → processing → report_ready → completed
type LabBooking struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	PatientID             uint       `gorm:"index" json:"patient_id"`
	LabCenterID           uint       `gorm:"index" json:"lab_center_id"`
	LabCenter             *LabCenter `gorm:"foreignKey:LabCenterID" json:"lab_center,omitempty"`
	LabTestID             uint       `gorm:"index" json:"lab_test_id"`
	LabTest               *LabTest   `gorm:"foreignKey:LabTestID" json:"lab_test,omitempty"`
	BookingType           string     `gorm:"default:'home_pickup'" json:"booking_type"` // home_pickup | lab_visit
	Status                string     `gorm:"default:'pending'" json:"status"`
	PreferredDate         string     `json:"preferred_date"`
	PreferredTimeSlot     string     `json:"preferred_time_slot"`
	PickupAddress         string     `json:"pickup_address"`
	PickupPincode         string     `json:"pickup_pincode"`
	AmountPaise           int64      `json:"amount_paise"`
	HomePickupChargePaise int64      `json:"home_pickup_charge_paise"`
	TotalPaise            int64      `json:"total_paise"`
	PaymentStatus         string     `gorm:"default:'pending'" json:"payment_status"`
	PaymentRef            string     `json:"payment_ref"`
	PhlebotomistID        *uint      `json:"phlebotomist_id,omitempty"`
	ReportFileURL         string     `json:"report_file_url"`
	AISummary             string     `json:"ai_summary"`
	AIRecommendation      string     `json:"ai_recommendation"`
	CancelledAt           *time.Time `json:"cancelled_at,omitempty"`
	CancelReason          string     `json:"cancel_reason,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func (LabBooking) TableName() string { return "lab_bookings" }

// ──────────────────────────────────────────────────────────────────────────────
// Hospital-internal models (kept for OPD/IPD use)
// ──────────────────────────────────────────────────────────────────────────────

// LabRequest is used internally by hospitals when doctors order tests for admitted patients.
type LabRequest struct {
	ID            uint          `gorm:"primaryKey" json:"id"`
	HospitalID    uint          `gorm:"index" json:"hospital_id"`
	PatientID     uint          `gorm:"index" json:"patient_id"`
	Patient       identity.User `gorm:"foreignKey:PatientID" json:"patient"`
	DoctorID      uint          `gorm:"index" json:"doctor_id"`
	Doctor        identity.User `gorm:"foreignKey:DoctorID" json:"doctor"`
	AppointmentID *uint         `gorm:"index" json:"appointment_id,omitempty"`
	AdmissionID   *uint         `gorm:"index" json:"admission_id,omitempty"`
	Status        string        `gorm:"default:'pending'" json:"status"`
	Priority      string        `gorm:"default:'normal'" json:"priority"`
	OrderedAt     time.Time     `json:"ordered_at"`
	CollectedAt   *time.Time    `json:"collected_at"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	Tests         []LabTest     `gorm:"many2many:lab_request_items;" json:"tests"`
}

func (LabRequest) TableName() string { return "lab_requests" }

type LabResult struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	RequestID    uint      `gorm:"index" json:"request_id"`
	TestID       uint      `gorm:"index" json:"test_id"`
	Test         LabTest   `gorm:"foreignKey:TestID" json:"test"`
	Value        string    `json:"value"`
	Unit         string    `json:"unit"`
	ReferenceMin *float64  `json:"reference_min"`
	ReferenceMax *float64  `json:"reference_max"`
	Flag         string    `json:"flag"` // normal | high | low | critical
	Comments     string    `json:"comments"`
	TechnicianID uint      `json:"technician_id"`
	VerifiedByID *uint     `json:"verified_by_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (LabResult) TableName() string { return "lab_results" }

type Specimen struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	RequestID     uint      `gorm:"index" json:"request_id"`
	Type          string    `json:"type"`
	Barcode       string    `gorm:"uniqueIndex" json:"barcode"`
	Status        string    `gorm:"default:'pending'" json:"status"`
	CollectedAt   time.Time `json:"collected_at"`
	CollectedByID uint      `json:"collected_by_id"`
	Notes         string    `json:"notes"`
}

func (Specimen) TableName() string { return "lab_specimens" }
