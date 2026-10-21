package cron

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// DoctorPerformanceSummary holds computed KPIs for a doctor in a period
type DoctorPerformanceSummary struct {
	DoctorID          uint    `json:"doctor_id"`
	Period            string  `json:"period"` // YYYY-MM
	TotalConsultations int    `json:"total_consultations"`
	CompletedConsultations int `json:"completed_consultations"`
	CancellationRate  float64 `json:"cancellation_rate"`
	AvgRating         float64 `json:"avg_rating"`
	TotalRevenue      int64   `json:"total_revenue"` // paise
	PatientRetention  float64 `json:"patient_retention"` // % returning patients
	ResponseTimeMins  float64 `json:"avg_response_time_mins"`
	BonusAmount       int64   `json:"bonus_amount"` // paise
	BonusTier         string  `json:"bonus_tier"` // none, bronze, silver, gold, platinum
	CreatedAt         time.Time `json:"created_at"`
}

func (DoctorPerformanceSummary) TableName() string {
	return "doctor_performance_summaries"
}

// BonusTier defines the thresholds and multipliers for performance bonuses
type BonusTier struct {
	Name           string
	MinRating      float64
	MinConsultations int
	MaxCancellation float64 // upper bound %
	BonusMultiplier float64 // multiplier on base bonus (10% of revenue)
}

var bonusTiers = []BonusTier{
	{Name: "platinum", MinRating: 4.8, MinConsultations: 50, MaxCancellation: 0.03, BonusMultiplier: 0.15},
	{Name: "gold",     MinRating: 4.5, MinConsultations: 30, MaxCancellation: 0.05, BonusMultiplier: 0.12},
	{Name: "silver",   MinRating: 4.0, MinConsultations: 20, MaxCancellation: 0.08, BonusMultiplier: 0.08},
	{Name: "bronze",   MinRating: 3.5, MinConsultations: 10, MaxCancellation: 0.12, BonusMultiplier: 0.05},
	{Name: "none",     MinRating: 0,   MinConsultations: 0,  MaxCancellation: 1.0,  BonusMultiplier: 0.0},
}

// DoctorPerformanceBonusCron computes and credits monthly performance bonuses
type DoctorPerformanceBonusCron struct {
	db *gorm.DB
}

// NewDoctorPerformanceBonusCron creates the cron job
func NewDoctorPerformanceBonusCron(db *gorm.DB) *DoctorPerformanceBonusCron {
	return &DoctorPerformanceBonusCron{db: db}
}

// Run executes the monthly performance bonus computation for the given period
// Typically called on the 1st of each month for the previous month
func (c *DoctorPerformanceBonusCron) Run(period string) ([]DoctorPerformanceSummary, error) {
	log.Printf("[BonusCron] Starting performance bonus computation for period: %s", period)

	// Parse period
	periodTime, err := time.Parse("2006-01", period)
	if err != nil {
		return nil, fmt.Errorf("invalid period format, expected YYYY-MM: %w", err)
	}
	startDate := periodTime
	endDate := periodTime.AddDate(0, 1, 0)

	// Fetch all active doctors
	var doctorIDs []uint
	c.db.Raw("SELECT id FROM users WHERE role = 'doctor' AND is_active = true").Scan(&doctorIDs)

	results := []DoctorPerformanceSummary{}

	for _, doctorID := range doctorIDs {
		summary, err := c.computeForDoctor(doctorID, period, startDate, endDate)
		if err != nil {
			log.Printf("[BonusCron] Error computing bonus for doctor %d: %v", doctorID, err)
			continue
		}

		// Save summary
		c.db.Where(DoctorPerformanceSummary{DoctorID: doctorID, Period: period}).
			Assign(*summary).
			FirstOrCreate(summary)

		// Credit bonus if non-zero
		if summary.BonusAmount > 0 {
			if err := c.creditBonus(doctorID, summary.BonusAmount, period, summary.BonusTier); err != nil {
				log.Printf("[BonusCron] Failed to credit bonus for doctor %d: %v", doctorID, err)
			}
		}

		results = append(results, *summary)
	}

	log.Printf("[BonusCron] Completed. Processed %d doctors for %s", len(results), period)
	return results, nil
}

func (c *DoctorPerformanceBonusCron) computeForDoctor(doctorID uint, period string, start, end time.Time) (*DoctorPerformanceSummary, error) {
	summary := &DoctorPerformanceSummary{
		DoctorID:  doctorID,
		Period:    period,
		CreatedAt: time.Now(),
	}

	// Total and completed consultations
	c.db.Raw(`
		SELECT COUNT(*) FROM appointments
		WHERE doctor_id = ? AND scheduled_at BETWEEN ? AND ?
	`, doctorID, start, end).Scan(&summary.TotalConsultations)

	c.db.Raw(`
		SELECT COUNT(*) FROM appointments
		WHERE doctor_id = ? AND scheduled_at BETWEEN ? AND ? AND status = 'completed'
	`, doctorID, start, end).Scan(&summary.CompletedConsultations)

	// Cancellation rate
	if summary.TotalConsultations > 0 {
		var cancelled int
		c.db.Raw(`
			SELECT COUNT(*) FROM appointments
			WHERE doctor_id = ? AND scheduled_at BETWEEN ? AND ? AND status = 'cancelled'
		`, doctorID, start, end).Scan(&cancelled)
		summary.CancellationRate = float64(cancelled) / float64(summary.TotalConsultations)
	}

	// Average rating from consultations in period
	c.db.Raw(`
		SELECT COALESCE(AVG(r.rating), 0) FROM reviews r
		JOIN appointments a ON a.id = r.appointment_id
		WHERE a.doctor_id = ? AND a.scheduled_at BETWEEN ? AND ?
	`, doctorID, start, end).Scan(&summary.AvgRating)

	// Total revenue from completed consultations
	c.db.Raw(`
		SELECT COALESCE(SUM(b.amount), 0) FROM billings b
		JOIN appointments a ON a.id = b.appointment_id
		WHERE a.doctor_id = ? AND a.scheduled_at BETWEEN ? AND ? AND a.status = 'completed'
	`, doctorID, start, end).Scan(&summary.TotalRevenue)

	// Patient retention (returning patients)
	var uniquePatients, returningPatients int64
	c.db.Raw(`SELECT COUNT(DISTINCT patient_id) FROM appointments WHERE doctor_id = ? AND scheduled_at BETWEEN ? AND ?`, doctorID, start, end).Scan(&uniquePatients)
	c.db.Raw(`
		SELECT COUNT(DISTINCT patient_id) FROM appointments
		WHERE doctor_id = ? AND scheduled_at BETWEEN ? AND ?
		AND patient_id IN (
			SELECT DISTINCT patient_id FROM appointments
			WHERE doctor_id = ? AND scheduled_at < ?
		)
	`, doctorID, start, end, doctorID, start).Scan(&returningPatients)
	if uniquePatients > 0 {
		summary.PatientRetention = float64(returningPatients) / float64(uniquePatients)
	}

	// Determine bonus tier
	tier := determineTier(summary)
	summary.BonusTier = tier.Name
	summary.BonusAmount = int64(float64(summary.TotalRevenue) * tier.BonusMultiplier)

	return summary, nil
}

func determineTier(s *DoctorPerformanceSummary) BonusTier {
	for _, tier := range bonusTiers {
		if s.AvgRating >= tier.MinRating &&
			s.CompletedConsultations >= tier.MinConsultations &&
			s.CancellationRate <= tier.MaxCancellation {
			return tier
		}
	}
	return bonusTiers[len(bonusTiers)-1] // "none"
}

func (c *DoctorPerformanceBonusCron) creditBonus(doctorID uint, amount int64, period, tier string) error {
	// Create a ledger credit entry for the doctor
	return c.db.Exec(`
		INSERT INTO ledger_entries (user_id, amount, entry_type, description, reference, created_at)
		VALUES (?, ?, 'credit', ?, ?, NOW())
	`, doctorID, amount, fmt.Sprintf("Performance bonus (%s tier) for %s", tier, period), "bonus_"+period).Error
}

// AutoMigrate creates the performance summary table
func (c *DoctorPerformanceBonusCron) AutoMigrate() error {
	return c.db.AutoMigrate(&DoctorPerformanceSummary{})
}

// RunForPreviousMonth is a convenience function to run the cron for last month
func (c *DoctorPerformanceBonusCron) RunForPreviousMonth() ([]DoctorPerformanceSummary, error) {
	lastMonth := time.Now().AddDate(0, -1, 0)
	period := lastMonth.Format("2006-01")
	return c.Run(period)
}
