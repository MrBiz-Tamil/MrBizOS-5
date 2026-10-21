// Package clinical provides chief-doctor clinical oversight — the lean Stage-1
// quality layer for online consultations. A chief_doctor samples completed
// consultations and records a quality review (score, notes, follow-up flag).
// No zone hierarchy yet; this scopes across all online doctors.
package clinical

import (
	"time"

	"gorm.io/gorm"
)

// ConsultationReview is a chief doctor's quality assessment of one completed
// consultation. One review per appointment.
type ConsultationReview struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	AppointmentID uint      `gorm:"uniqueIndex;not null" json:"appointment_id"`
	DoctorID      uint      `gorm:"index;not null" json:"doctor_id"`   // the reviewed doctor (user id)
	ReviewerID    uint      `gorm:"index;not null" json:"reviewer_id"` // the chief doctor (user id)
	QualityScore  int       `gorm:"not null" json:"quality_score"`     // 1–5
	Notes         string    `gorm:"type:text" json:"notes"`
	Flagged       bool      `gorm:"default:false;index" json:"flagged"` // needs follow-up
	// Outcome is the action taken: ok | coaching | warning.
	Outcome   string    `gorm:"default:'ok'" json:"outcome"`
	CreatedAt time.Time `json:"created_at"`
}

func (ConsultationReview) TableName() string { return "consultation_reviews" }

// AutoMigrate creates the clinical-oversight tables.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&ConsultationReview{}, &DoctorOnboarding{}, &Pod{})
}
