package emr

import (
	"drayya-core/internal/domain/identity"
	"drayya-core/internal/utils/privacy"
	"time"

	"gorm.io/datatypes"
)

type Prescription struct {
	ID            uint                    `gorm:"primaryKey" json:"id"`
	AppointmentID uint                    `gorm:"index" json:"appointment_id"`
	DoctorID      uint                    `gorm:"index;index:idx_prescription_doctor_created,priority:1" json:"doctor_id"`
	PatientID     uint                    `gorm:"column:user_id;index:idx_prescription_patient_created,priority:1" json:"patient_id"`
	Diagnosis     privacy.EncryptedString `json:"diagnosis"`
	Symptoms      privacy.EncryptedString `json:"symptoms"`
	Medicines     datatypes.JSON          `json:"medicines"`
	LabTests      datatypes.JSON          `json:"lab_tests"`
	Instructions  privacy.EncryptedString `json:"instructions"`
	PdfURL        string                  `gorm:"column:file_url" json:"pdf_url"`
	Status        string                  `gorm:"default:'issued'" json:"status"` // issued, verified, rejected
	CreatedAt     time.Time               `gorm:"index:idx_prescription_patient_created,priority:2,sort:desc;index:idx_prescription_doctor_created,priority:2,sort:desc" json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`

	// Relations
	Doctor  identity.User `gorm:"foreignKey:DoctorID" json:"doctor,omitempty"`
	Patient identity.User `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}

func (Prescription) TableName() string {
	return "prescriptions"
}

type MedicalRecord struct {
	ID        uint                    `gorm:"primaryKey" json:"id"`
	PatientID uint                    `gorm:"index;index:idx_med_record_patient_created,priority:1" json:"patient_id"`
	DoctorID  uint                    `gorm:"index;index:idx_med_record_doctor_created,priority:1" json:"doctor_id"`
	Diagnosis privacy.EncryptedString `json:"diagnosis"`
	Notes     privacy.EncryptedString `json:"notes"`
	// DocumentID links to a private, consent-gated file in the document vault.
	// Preferred over FileURL, which is a legacy plain-URL/empty field. When set,
	// the attached file is fetched via a short-lived signed URL after an access
	// check, not a public link.
	DocumentID *uint  `gorm:"index" json:"document_id"`
	FileURL    string `json:"file_url"`
	FileType   string `json:"file_type"`
	CreatedAt time.Time               `gorm:"index:idx_med_record_patient_created,priority:2,sort:desc;index:idx_med_record_doctor_created,priority:2,sort:desc" json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`

	// Relations
	Doctor  identity.User `gorm:"foreignKey:DoctorID" json:"doctor,omitempty"`
	Patient identity.User `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}

func (MedicalRecord) TableName() string {
	return "medical_records"
}

type PrescriptionTemplate struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	DoctorID  uint           `gorm:"index" json:"doctor_id"`
	Name      string         `json:"name"`
	Diagnosis string         `json:"diagnosis"`
	Medicines datatypes.JSON `json:"medicines"`
	CreatedAt time.Time      `json:"created_at"`
}

func (PrescriptionTemplate) TableName() string {
	return "prescription_templates"
}

type Diagnosis struct {
	ID            uint                    `gorm:"primaryKey" json:"id"`
	PatientID     uint                    `gorm:"index;index:idx_diagnosis_patient_created,priority:1;not null" json:"patient_id"`
	DoctorID      uint                    `gorm:"index;index:idx_diagnosis_doctor_created,priority:1;not null" json:"doctor_id"`
	AppointmentID *uint                   `gorm:"index" json:"appointment_id"`
	Condition     privacy.EncryptedString `gorm:"not null" json:"condition"`
	Severity      string                  `json:"severity"`
	Notes         privacy.EncryptedString `json:"notes"`
	CreatedAt     time.Time               `gorm:"index:idx_diagnosis_patient_created,priority:2,sort:desc;index:idx_diagnosis_doctor_created,priority:2,sort:desc" json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`

	// Relations
	Doctor  identity.User `gorm:"foreignKey:DoctorID" json:"doctor,omitempty"`
	Patient identity.User `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}

func (Diagnosis) TableName() string {
	return "diagnosis"
}

type PatientVitals struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PatientID   uint      `gorm:"index:idx_vitals_patient_recorded,priority:1" json:"patient_id"`
	SystolicBP  float64   `json:"systolic_bp"`
	DiastolicBP float64   `json:"diastolic_bp"`
	HeartRate   float64   `json:"heart_rate"`
	RecordedAt  time.Time `gorm:"index:idx_vitals_patient_recorded,priority:2,sort:desc" json:"recorded_at"`
}

func (PatientVitals) TableName() string {
	return "patient_vitals"
}

type HealthTimelineEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PatientID   uint      `gorm:"index:idx_patient_timeline_date,priority:1" json:"patient_id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Date        time.Time `gorm:"index:idx_patient_timeline_date,priority:2,sort:desc" json:"date"`
	ReferenceID uint      `json:"reference_id"`
}

func (HealthTimelineEvent) TableName() string {
	return "health_timeline_events"
}

type EMRAccessGrant struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	DoctorID   uint      `gorm:"index:idx_grant_lookup,priority:1;not null" json:"doctor_id"`
	PatientID  uint      `gorm:"index:idx_grant_lookup,priority:2;not null" json:"patient_id"`
	SourceType string    `gorm:"type:varchar(50);not null" json:"source_type"` // e.g. "appointment", "manual_consent", "pharmacy"
	SourceID   uint      `gorm:"index;not null" json:"source_id"`             // ID of the booking / order
	GrantedAt  time.Time `json:"granted_at"`
	ExpiresAt  time.Time `gorm:"index:idx_grant_lookup,priority:3,sort:asc;not null" json:"expires_at"`
}

func (EMRAccessGrant) TableName() string {
	return "emr_access_grants"
}

