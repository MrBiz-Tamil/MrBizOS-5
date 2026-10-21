//go:build ignore
// +build ignore

package models

import (
	"time"
)

// DoctorProfile mirrors the Python 'doctorprofile' table
type DoctorProfile struct {
	ID                       uint       `gorm:"primaryKey" json:"id"`
	UserID                   uint       `gorm:"uniqueIndex;not null" json:"user_id"`
	Specialization           string     `json:"specialization"`
	LicenseNumber            string     `gorm:"uniqueIndex" json:"license_number"`
	YearsOfExperience        int        `json:"years_of_experience"`
	Qualification            string     `json:"qualification"`
	ConsultationFee          int64      `json:"consultation_fee"` // In cents
	IsVerified               bool       `gorm:"default:false" json:"is_verified"`
	IsOnline                 bool       `gorm:"default:false" json:"is_online"`
	CurrentStatus            string     `gorm:"default:'offline'" json:"current_status"` // available, in_consultation, offline
	Gender                   string     `json:"gender"`                                  // explicit gender preference
	Latitude                 float64    `json:"latitude"`
	Longitude                float64    `json:"longitude"`
	Bio                      string     `json:"bio"`
	AverageRating            float64    `json:"average_rating"`
	TotalConsultations       int        `json:"total_consultations"`
	TotalConsultationsToday  int        `gorm:"default:0" json:"total_consultations_today"` // for round-robin balancing
	ProfileCompletionPercent int        `json:"profile_completion_percent"`
	LanguagesSpoken          string     `json:"languages_spoken"` // JSON string ["english", "hindi"]
	AdminBoostScore          float64    `gorm:"default:0" json:"admin_boost_score"`
	OffersFreeConsultation   bool       `gorm:"default:false" json:"offers_free_consultation"`
	DailyFreeConsultLimit    int        `gorm:"default:0" json:"daily_free_consult_limit"`  // Anti-abuse daily cap
	FreeConsultsGivenToday   int        `gorm:"default:0" json:"free_consults_given_today"` // Tracker for today
	FreeType                 string     `gorm:"default:'first_time_only'" json:"free_type"` // Campaign Type: 'first_time_only' or 'weekly_free_day'
	FreeDayOfWeek            int        `gorm:"default:0" json:"free_day_of_week"`          // 0=Sunday, 1=Monday, etc (for weekly_free_day)
	LastPing                 *time.Time `json:"last_ping"`                                  // to detect abandoned sessions
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

func (DoctorProfile) TableName() string {
	return "doctorprofile"
}

// DoctorAvailability represents doctor's weekly availability schedule
type DoctorAvailability struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	DoctorID    uint      `gorm:"index;not null" json:"doctor_id"`
	DayOfWeek   int       `gorm:"not null" json:"day_of_week"` // 0=Sunday, 1=Monday, etc.
	StartTime   string    `gorm:"not null" json:"start_time"`  // Format: "09:00"
	EndTime     string    `gorm:"not null" json:"end_time"`    // Format: "17:00"
	IsAvailable bool      `gorm:"default:true" json:"is_available"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (DoctorAvailability) TableName() string {
	return "doctor_availability"
}
