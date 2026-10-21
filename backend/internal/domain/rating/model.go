package rating

import (
	"time"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Rating{},
	)
}

type Rating struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	DoctorID      uint      `gorm:"index;not null" json:"doctor_id"`
	UserID        uint      `gorm:"index;not null" json:"user_id"`
	AppointmentID uint      `gorm:"uniqueIndex" json:"appointment_id"` // one rating per appointment
	Score         int       `gorm:"not null" json:"score"`             // 1-5
	Comment       string    `gorm:"type:text" json:"comment"`
	IsHidden      bool      `gorm:"default:false" json:"is_hidden"`
	HiddenBy      *uint     `json:"hidden_by,omitempty"`
	HiddenReason  string    `gorm:"type:text" json:"hidden_reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (Rating) TableName() string {
	return "ratings"
}
