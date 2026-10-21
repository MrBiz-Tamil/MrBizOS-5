//go:build ignore
// +build ignore

package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
)

// Prescription mirrors the Python 'prescriptions' table
type Prescription struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	AppointmentID uint           `gorm:"index" json:"appointment_id"`
	DoctorID      uint           `gorm:"index" json:"doctor_id"`
	PatientID     uint           `gorm:"index" json:"patient_id"` // Sometimes derived via appointment, but useful to have
	Diagnosis     string         `json:"diagnosis"`
	Symptoms      string         `json:"symptoms"`
	Medicines     datatypes.JSON `json:"medicines"` // Stored as JSON [{"name": "...", "dosage": "..."}]
	LabTests      pq.StringArray `gorm:"type:text[]" json:"lab_tests"`
	Instructions  string         `json:"instructions"`
	PdfURL        string         `json:"pdf_url"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (Prescription) TableName() string {
	return "prescriptions"
}

// MedicalRecord mirrors the Python 'medical_records' table
type MedicalRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PatientID uint      `gorm:"index" json:"patient_id"`
	DoctorID  uint      `gorm:"index" json:"doctor_id"`
	Diagnosis string    `json:"diagnosis"`
	Notes     string    `json:"notes"`
	FileURL   string    `json:"file_url"`
	FileType  string    `json:"file_type"` // e.g., 'lab_result', 'xray'
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (MedicalRecord) TableName() string {
	return "medical_records"
}

// PrescriptionTemplate for quick reuse
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

// Diagnosis model for standalone or appointment-linked clinical data
type Diagnosis struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	PatientID     uint      `gorm:"index;not null" json:"patient_id"`
	DoctorID      uint      `gorm:"index;not null" json:"doctor_id"`
	AppointmentID *uint     `gorm:"index" json:"appointment_id"`
	Condition     string    `gorm:"not null" json:"condition"`
	Severity      string    `json:"severity"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Diagnosis) TableName() string {
	return "diagnosis"
}
