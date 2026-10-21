package hr

import (
	"time"

	"gorm.io/gorm"
)

type PerformanceReview struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	DoctorID             uint           `gorm:"index;not null" json:"doctor_id"`
	DoctorName           string         `json:"doctor_name"`
	ReviewerID           uint           `json:"reviewer_id"`
	ReviewerName         string         `json:"reviewer_name"`
	Period               string         `json:"period"`
	ReviewCycle          string         `gorm:"index" json:"review_cycle"` // e.g. "H1-2026", "Q2-2026", "Annual-2026"
	OverallScore         float64        `json:"overall_performance_score"`
	Category             string         `json:"performance_category"`
	// Rating dimensions (0–10) — a basic competency breakdown behind the overall.
	ClinicalScore        float64        `json:"clinical_score"`
	CommunicationScore   float64        `json:"communication_score"`
	PunctualityScore     float64        `json:"punctuality_score"`
	Goals                string         `gorm:"type:text" json:"goals"` // free-text or JSON goals for the next cycle
	AdjustmentPercentage float64        `json:"recommended_rate_adjustment"`
	HRNotes              string         `gorm:"type:text" json:"hr_notes"`
	Status               string         `gorm:"default:'draft'" json:"status"` // draft, approved
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

func (PerformanceReview) TableName() string {
	return "performance_reviews"
}
