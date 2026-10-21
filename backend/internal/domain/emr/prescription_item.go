package emr

import (
	"time"

	"gorm.io/gorm"
)

// PrescriptionItem is one structured line of a prescription. It replaces the
// opaque `Prescription.medicines` JSON column with rows that:
//
//   - reference a Medicine (the AYUSH product master) by ID, so pharmacy
//     auto-cart generation can map directly to inventory SKUs;
//   - capture the doctor's dosage in machine-readable form (qty/freq/duration)
//     so total quantity to dispense is computable, refill reminders work, and
//     pharmacovigilance/adherence reports stay accurate;
//   - track refills so chronic-supplement prescriptions self-limit;
//   - record substitutions a pharmacist makes (with reason + auditor).
//
// The legacy `Prescription.medicines` JSON stays in place for old records and
// is migrated lazily — new prescriptions created via the structured endpoint
// write here; old ones continue to display from the JSON.
type PrescriptionItem struct {
	ID             uint           `gorm:"primaryKey"        json:"id"`
	PrescriptionID uint           `gorm:"index;not null"    json:"prescription_id"`
	MedicineID     uint           `gorm:"index;not null"    json:"medicine_id"` // FK to pharmacy.Medicine

	// Doctor's structured dose plan.
	DosageText       string  `gorm:"type:text" json:"dosage_text"` // free-form ("2 tabs after meals")
	QuantityPerDose  float64 `json:"quantity_per_dose"`            // e.g., 2
	FrequencyPerDay  float64 `json:"frequency_per_day"`            // e.g., 2 (BID)
	DurationDays     int     `json:"duration_days"`
	TotalQuantity    float64 `json:"total_quantity"` // computed: qty × freq × duration
	Instructions     string  `gorm:"type:text" json:"instructions,omitempty"`

	// Refill tracking — for chronic wellness supplements where the doctor
	// authorises N repeat dispensings without a new consultation.
	RefillsAllowed int `gorm:"default:0" json:"refills_allowed"`
	RefillsUsed    int `gorm:"default:0" json:"refills_used"`
	// LastRefilledAt and NextRefillDueAt drive the WhatsApp/email nudge sweep:
	// when refills_used < refills_allowed AND next_refill_due_at < now, the
	// patient gets a reminder that their next pack should be ordered.
	LastRefilledAt   *time.Time `gorm:"index" json:"last_refilled_at,omitempty"`
	NextRefillDueAt  *time.Time `gorm:"index" json:"next_refill_due_at,omitempty"`
	LastReminderSentAt *time.Time `json:"last_reminder_sent_at,omitempty"`

	// Pharmacist substitution audit trail.
	SubstitutedMedicineID *uint      `gorm:"index" json:"substituted_medicine_id,omitempty"`
	SubstitutionReason    string     `json:"substitution_reason,omitempty"`
	SubstitutedByUserID   *uint      `json:"substituted_by_user_id,omitempty"`
	SubstitutedAt         *time.Time `json:"substituted_at,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (PrescriptionItem) TableName() string { return "prescription_items" }

// EffectiveMedicineID returns the medicine that should actually be dispensed:
// the pharmacist's substitution if one was made, otherwise the doctor's pick.
func (it PrescriptionItem) EffectiveMedicineID() uint {
	if it.SubstitutedMedicineID != nil && *it.SubstitutedMedicineID != 0 {
		return *it.SubstitutedMedicineID
	}
	return it.MedicineID
}

// HasRefillsRemaining reports whether this line can be re-dispensed without a
// new prescription.
func (it PrescriptionItem) HasRefillsRemaining() bool {
	return it.RefillsUsed < it.RefillsAllowed
}
