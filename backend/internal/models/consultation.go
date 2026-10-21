package models

import (
	"time"
)

type VideoConsultation struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	SessionID       string     `gorm:"uniqueIndex" json:"session_id"`
	AppointmentID   uint       `json:"appointment_id"`
	DoctorID        uint       `json:"doctor_id"`
	PatientID       uint       `json:"patient_id"`
	Status          string     `json:"status"` // scheduled, in_progress, completed
	StartTime       *time.Time `json:"start_time"`
	EndTime         *time.Time `json:"end_time"`
	DurationMinutes int        `json:"duration_minutes"`
	CallQuality     string     `json:"call_quality"` // excellent, good, fair, poor
	RecordingURL    string     `json:"recording_url"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (VideoConsultation) TableName() string {
	return "videoconsultation"
}
