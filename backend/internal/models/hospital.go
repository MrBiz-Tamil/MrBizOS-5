//go:build ignore
// +build ignore

package models

import (
	"time"

	"gorm.io/datatypes"
)

type Hospital struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Name         string     `gorm:"not null" json:"name"`
	Address      string     `json:"address"`
	ContactInfo  string     `json:"contact_info"`
	Type         string     `gorm:"default:'General'" json:"type"`        // e.g., General, Specialized
	HospitalType string     `gorm:"default:'owned'" json:"hospital_type"` // owned or franchise
	FranchiseID  *uint      `json:"franchise_id"`
	Franchise    *Franchise `gorm:"foreignKey:FranchiseID" json:"franchise,omitempty"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (Hospital) TableName() string {
	return "hospitals"
}

type HospitalDepartment struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	HospitalID  uint      `gorm:"index" json:"hospital_id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (HospitalDepartment) TableName() string {
	return "hospital_departments"
}

type Bed struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	HospitalID    uint      `gorm:"index" json:"hospital_id"`
	DepartmentID  uint      `gorm:"index" json:"department_id"`
	RoomNumber    string    `json:"room_number"`
	BedNumber     string    `json:"bed_number"`
	Status        string    `gorm:"default:'available'" json:"status"` // available, occupied, maintenance
	PricePerNight float64   `json:"price_per_night"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Bed) TableName() string {
	return "hospital_beds"
}

// WardServiceRequest represents an in-hospital order for medicines or supplies from a Nurse to the Pharmacy
type WardServiceRequest struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	HospitalID    uint           `gorm:"index" json:"hospital_id"`
	PatientID     *uint          `json:"patient_id"`
	RequestedByID uint           `json:"requested_by_id"` // UserID of the Nurse
	BedID         *uint          `json:"bed_id"`
	Items         datatypes.JSON `json:"items"`                            // JSON [{"medicine_id": 1, "qty": 2}]
	Status        string         `gorm:"default:'pending'" json:"status"`  // pending, processing, dispatched, delivered
	Priority      string         `gorm:"default:'normal'" json:"priority"` // normal, urgent, stat
	Notes         string         `json:"notes"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (WardServiceRequest) TableName() string {
	return "hospital_ward_service_requests"
}

type IPDAdmission struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	HospitalID       uint       `gorm:"index" json:"hospital_id"`
	PatientID        uint       `gorm:"index" json:"patient_id"`
	Patient          User       `gorm:"foreignKey:PatientID" json:"patient"`
	DoctorID         uint       `gorm:"index" json:"doctor_id"`
	Doctor           User       `gorm:"foreignKey:DoctorID" json:"doctor"`
	BedID            uint       `gorm:"index" json:"bed_id"`
	Bed              Bed        `gorm:"foreignKey:BedID" json:"bed"`
	AdmissionDate    time.Time  `json:"admission_date"`
	DischargeDate    *time.Time `json:"discharge_date"`
	Status           string     `gorm:"default:'admitted'" json:"status"` // admitted, stable, critical, discharged
	DischargeSummary string     `json:"discharge_summary"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (IPDAdmission) TableName() string {
	return "hospital_ipd_admissions"
}
