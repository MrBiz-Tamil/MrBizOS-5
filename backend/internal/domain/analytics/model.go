package analytics

import (
	"time"

	"gorm.io/gorm"
)

// Event is a generic, append-only telemetry row.
type Event struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
	UserID     *uint     `gorm:"index" json:"user_id,omitempty"` // nil for anonymous
	SessionID  string    `gorm:"size:100;index" json:"session_id,omitempty"`
	EventName  string    `gorm:"size:100;index;not null" json:"event_name"`
	Domain     string    `gorm:"size:50" json:"domain"`    // consultation, pharmacy, lab, subscription
	EntityID   *uint     `json:"entity_id,omitempty"`      // contextual ID (booking_id, order_id …)
	Properties string    `gorm:"type:text" json:"properties,omitempty"` // JSON blob
	IPAddress  string    `gorm:"size:45" json:"-"`
	UserAgent  string    `gorm:"size:500" json:"-"`
	Platform   string    `gorm:"size:20" json:"platform"` // web, android, ios, api
}

func (Event) TableName() string { return "analytics_events" }

// DoctorAnalyticsSummary is a pre-aggregated cache for doctor dashboard KPIs.
type DoctorAnalyticsSummary struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	UpdatedAt           time.Time `json:"updated_at"`
	DoctorUserID        uint      `gorm:"uniqueIndex;not null" json:"doctor_user_id"`
	Period              string    `gorm:"size:20" json:"period"` // "30d", "90d", "all"
	TotalConsultations  int64     `json:"total_consultations"`
	CompletedConsults   int64     `json:"completed_consults"`
	CancelledConsults   int64     `json:"cancelled_consults"`
	AvgRating           float64   `json:"avg_rating"`
	RepeatPatientPct    float64   `json:"repeat_patient_pct"`
	TotalEarningsPaise  int64     `json:"total_earnings_paise"`
	AvgConsultFeePaise  int64     `json:"avg_consult_fee_paise"`
}

type CFOMetrics struct {
	Revenue struct {
		TotalMonthlyRevenue Double `json:"total_monthly_revenue"`
		ProjectedRevenue    Double `json:"projected_revenue"`
		GrowthPercentage    Double `json:"growth_percentage"`
		BreakdownByService  map[string]Double `json:"breakdown_by_service"`
	} `json:"revenue"`
	Retention struct {
		RetentionRate     Double `json:"retention_rate"`
		ChurnRate         Double `json:"churn_rate"`
		NewPatients       int    `json:"new_patients"`
		ReturningPatients int    `json:"returning_patients"`
	} `json:"retention"`
	WaitTime struct {
		AverageWaitMinutes           Double `json:"average_wait_minutes"`
		AppointmentsOnTimePercentage Double `json:"appointments_on_time_percentage"`
	} `json:"wait_time"`
}

type Double = float64


func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Event{}, &DoctorAnalyticsSummary{})
}
