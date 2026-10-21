package pharmacy

import (
	"time"

	"gorm.io/gorm"
)

// DrugRecall records a regulatory recall notice for a medicine batch.
type DrugRecall struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	DrugName         string         `gorm:"size:300;not null" json:"drug_name"`
	BatchNumber      string         `gorm:"size:100;not null;index" json:"batch_number"`
	ManufacturerName string         `gorm:"size:300" json:"manufacturer_name"`
	RecallClass      string         `gorm:"size:10" json:"recall_class"` // Class I, II, III
	Reason           string         `gorm:"type:text" json:"reason"`
	IssuedBy         string         `gorm:"size:200" json:"issued_by"` // CDSCO / State FDA
	RecallDate       time.Time      `json:"recall_date"`
	EffectiveUntil   *time.Time     `json:"effective_until,omitempty"`
	Status           string         `gorm:"size:20;default:'active'" json:"status"` // active, resolved
	AffectedQuantity int64          `json:"affected_quantity"`
	QuarantinedQty   int64          `json:"quarantined_qty"`
	ReturnedQty      int64          `json:"returned_qty"`
	Notes            string         `gorm:"type:text" json:"notes,omitempty"`
}

// DrugRecallAction records what was done for each inventory item under a recall.
type DrugRecallAction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	RecallID    uint      `gorm:"index;not null" json:"recall_id"`
	InventoryID uint      `gorm:"index" json:"inventory_id"`
	Action      string    `gorm:"size:50" json:"action"` // quarantined, returned_to_supplier, destroyed
	ActionBy    uint      `json:"action_by"`
	Notes       string    `gorm:"type:text" json:"notes,omitempty"`
}

// ── Schedule H / Narcotics Compliance ────────────────────────────────────────

// ScheduleHDrug flags a medicine as requiring a valid prescription (Schedule H/H1/X).
type ScheduleHDrug struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	DrugName   string         `gorm:"size:300;not null" json:"drug_name"`
	GenericName string        `gorm:"size:300" json:"generic_name"`
	Schedule   string         `gorm:"size:10;not null" json:"schedule"` // H, H1, X (narcotic)
	IsNarcotic bool           `gorm:"default:false" json:"is_narcotic"`
	Notes      string         `gorm:"type:text" json:"notes,omitempty"`
}

// NarcoticsRegisterEntry is the dispensing record required by the Narcotic Drugs
// and Psychotropic Substances Act, 1985.
type NarcoticsRegisterEntry struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	DrugName       string    `gorm:"size:300;not null" json:"drug_name"`
	BatchNumber    string    `gorm:"size:100" json:"batch_number"`
	QuantityGrams  float64   `json:"quantity_grams"`
	PatientName    string    `gorm:"size:200" json:"patient_name"`
	PatientID      uint      `json:"patient_id"`
	PrescriptionID uint      `json:"prescription_id"`
	DoctorName     string    `gorm:"size:200" json:"doctor_name"`
	DoctorRegNo    string    `gorm:"size:100" json:"doctor_reg_no"`
	DispenserID    uint      `json:"dispenser_id"` // pharmacy staff user ID
	DispensedAt    time.Time `json:"dispensed_at"`
}

func (NarcoticsRegisterEntry) TableName() string { return "narcotics_register" }
