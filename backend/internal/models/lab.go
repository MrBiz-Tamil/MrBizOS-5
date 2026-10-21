//go:build ignore
// +build ignore

package models

import (
	"drayya-core/internal/domain/identity"
	"time"
)

type LabTest struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"not null" json:"name"`
	Category       string    `json:"category"` // Biochemistry, Hematology, etc.
	Description    string    `json:"description"`
	ReferenceRange string    `json:"reference_range"` // e.g. "70-110 mg/dL"
	Unit           string    `json:"unit"`            // e.g. "mg/dL"
	BasePrice      int64     `json:"base_price"`      // in cents
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (LabTest) TableName() string {
	return "lab_tests"
}

type LabRequest struct {
	ID            uint          `gorm:"primaryKey" json:"id"`
	HospitalID    uint          `gorm:"index" json:"hospital_id"`
	PatientID     uint          `gorm:"index" json:"patient_id"`
	Patient       identity.User `gorm:"foreignKey:PatientID" json:"patient"`
	DoctorID      uint          `gorm:"index" json:"doctor_id"`
	Doctor        identity.User `gorm:"foreignKey:DoctorID" json:"doctor"`
	AppointmentID *uint         `gorm:"index" json:"appointment_id,omitempty"`
	AdmissionID   *uint         `gorm:"index" json:"admission_id,omitempty"`
	Status        string        `gorm:"default:'pending'" json:"status"`  // pending, collected, processing, completed, cancelled
	Priority      string        `gorm:"default:'normal'" json:"priority"` // normal, urgent, stat
	OrderedAt     time.Time     `json:"ordered_at"`
	CollectedAt   *time.Time    `json:"collected_at"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`

	Tests []LabTest `gorm:"many2many:lab_request_items;" json:"tests"`
}

func (LabRequest) TableName() string {
	return "lab_requests"
}

type LabResult struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	RequestID    uint      `gorm:"index" json:"request_id"`
	TestID       uint      `gorm:"index" json:"test_id"`
	Test         LabTest   `gorm:"foreignKey:TestID" json:"test"`
	Value        string    `json:"value"` // Stored as string to handle qualitative results
	Unit         string    `json:"unit"`
	ReferenceMin *float64  `json:"reference_min"`
	ReferenceMax *float64  `json:"reference_max"`
	Flag         string    `json:"flag"` // Normal, High, Low, Critical
	Comments     string    `json:"comments"`
	TechnicianID uint      `json:"technician_id"`
	VerifiedByID *uint     `json:"verified_by_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (LabResult) TableName() string {
	return "lab_results"
}

type Specimen struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	RequestID     uint      `gorm:"index" json:"request_id"`
	Type          string    `json:"type"` // Blood, Urine, etc.
	Barcode       string    `gorm:"uniqueIndex" json:"barcode"`
	Status        string    `gorm:"default:'pending'" json:"status"` // pending, collected, received, rejected, disposed
	CollectedAt   time.Time `json:"collected_at"`
	CollectedByID uint      `json:"collected_by_id"`
	Notes         string    `json:"notes"`
}

func (Specimen) TableName() string {
	return "lab_specimens"
}
