package models

import (
	"time"
)

// PatientVitals mirrors the frontend PatientVitals type
type PatientVitals struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	PatientID         uint      `gorm:"index;not null" json:"patient_id"`
	SystolicBP        float64   `json:"systolic_bp"`
	DiastolicBP       float64   `json:"diastolic_bp"`
	HeartRate         float64   `json:"heart_rate"`
	Temperature       float64   `json:"temperature"`
	Weight            float64   `json:"weight"`
	BloodSugarFasting float64   `json:"blood_sugar_fasting"`
	Spo2              float64   `json:"spo2"`
	RecordedAt        time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"recorded_at"`
	CreatedAt         time.Time `json:"created_at"`
}

func (PatientVitals) TableName() string {
	return "patient_vitals"
}

// HealthTimelineEvent represents a unified timeline event
type HealthTimelineEvent struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	PatientID   uint      `gorm:"index;not null" json:"patient_id"`
	Type        string    `gorm:"not null" json:"type"` // 'visit', 'prescription', 'vitals', 'lab', 'document'
	Title       string    `gorm:"not null" json:"title"`
	Description string    `json:"description"`
	Date        time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"date"`
	ReferenceID uint      `json:"reference_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func (HealthTimelineEvent) TableName() string {
	return "health_timeline_events"
}
