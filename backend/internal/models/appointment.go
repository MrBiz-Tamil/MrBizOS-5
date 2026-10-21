//go:build ignore
// +build ignore

// Deprecated: Migrated to internal/domain/appointment
package models

import (
	"time"
)

type AppointmentStatus string

const (
	StatusPending       AppointmentStatus = "pending"
	StatusScheduled     AppointmentStatus = "scheduled"
	StatusConfirmed     AppointmentStatus = "confirmed"
	StatusCompleted     AppointmentStatus = "completed"
	StatusCancelled     AppointmentStatus = "cancelled"
	StatusNoShow        AppointmentStatus = "no_show"
	StatusDoctorEnRoute AppointmentStatus = "doctor_en_route"
	StatusDoctorArrived AppointmentStatus = "doctor_arrived"
)

type Appointment struct {
	ID               uint              `gorm:"primaryKey" json:"id"`
	PatientID        uint              `gorm:"not null" json:"patient_id"`
	Patient          User              `gorm:"foreignKey:PatientID" json:"patient"`
	DoctorID         *uint             `json:"doctor_id"`
	Doctor           *User             `gorm:"foreignKey:DoctorID" json:"doctor"`
	StartTime        time.Time         `json:"start_time"`
	EndTime          time.Time         `json:"end_time"`
	Status           AppointmentStatus `gorm:"default:'pending'" json:"status"`
	AppointmentType  string            `json:"appointment_type"`
	ConsultationMode string            `json:"consultation_mode"`
	ConsultationFee  int64             `json:"consultation_fee"` // In cents
	Notes            string            `json:"notes"`
	QueueNumber      *int              `json:"queue_number"`
	IsPriority       bool              `gorm:"default:false" json:"is_priority"`
	FamilyMemberID   *uint             `json:"family_member_id"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

func (Appointment) TableName() string {
	return "appointment"
}
