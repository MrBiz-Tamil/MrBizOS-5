package analytics

import (
	"encoding/json"
	"fmt"
	"time"

	"drayya-core/internal/database"
)

// Track records an analytics event asynchronously-safe (fire-and-forget).
func Track(eventName, domain, platform string, userID *uint, sessionID string, entityID *uint, props map[string]interface{}, ip, ua string) {
	var propsJSON string
	if props != nil {
		b, _ := json.Marshal(props)
		propsJSON = string(b)
	}
	ev := &Event{
		EventName:  eventName,
		Domain:     domain,
		Platform:   platform,
		UserID:     userID,
		SessionID:  sessionID,
		EntityID:   entityID,
		Properties: propsJSON,
		IPAddress:  ip,
		UserAgent:  ua,
	}
	// Best-effort insert; silently ignore errors
	_ = database.DB.Create(ev).Error
}

// GetRecentEvents returns the latest events for admin inspection.
func GetRecentEvents(limit int, domain string) ([]Event, error) {
	var events []Event
	q := database.DB.Order("created_at DESC").Limit(limit)
	if domain != "" {
		q = q.Where("domain = ?", domain)
	}
	return events, q.Find(&events).Error
}

// GetEventCountByName aggregates event frequencies for a period.
func GetEventCountByName(from, to time.Time) ([]map[string]interface{}, error) {
	type row struct {
		EventName string
		Count     int64
	}
	var rows []row
	err := database.DB.Model(&Event{}).
		Select("event_name, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", from, to).
		Group("event_name").
		Order("count DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, len(rows))
	for i, r := range rows {
		result[i] = map[string]interface{}{
			"event_name": r.EventName,
			"count":      r.Count,
		}
	}
	return result, nil
}

// RefreshDoctorAnalytics recomputes the KPI summary for a doctor.
func RefreshDoctorAnalytics(doctorUserID uint, period string) (*DoctorAnalyticsSummary, error) {
	var from time.Time
	now := time.Now()
	switch period {
	case "30d":
		from = now.AddDate(0, 0, -30)
	case "90d":
		from = now.AddDate(0, 0, -90)
	default:
		from = time.Time{} // all time
	}

	// Count consultations from appointments table
	type stats struct {
		Total     int64
		Completed int64
		Cancelled int64
	}
	var s stats
	q := database.DB.Table("appointments").Where("doctor_id = ?", doctorUserID)
	if !from.IsZero() {
		q = q.Where("created_at >= ?", from)
	}
	q.Select("COUNT(*) as total, SUM(CASE WHEN status='completed' THEN 1 ELSE 0 END) as completed, SUM(CASE WHEN status='cancelled' THEN 1 ELSE 0 END) as cancelled").Scan(&s)

	// Average rating
	var avgRating float64
	database.DB.Table("ratings").Where("doctor_id = ?", doctorUserID).Select("COALESCE(AVG(score), 0)").Scan(&avgRating)

	// Earnings: sum consultation fees from completed appointments assigned to this doctor
	var earningsPaise int64
	database.DB.Table("appointments").
		Where("doctor_id = ? AND status = 'completed'", doctorUserID).
		Select("COALESCE(SUM(consultation_fee), 0)").
		Scan(&earningsPaise)

	// Repeat patient percentage
	type repeatData struct {
		Total   int64
		Repeats int64
	}
	var rd repeatData
	database.DB.Table("appointments").
		Select("COUNT(DISTINCT patient_id) as total").
		Where("doctor_id = ?", doctorUserID).Scan(&rd.Total)
	database.DB.Table("appointments").
		Select("COUNT(DISTINCT patient_id) as repeats").
		Where("doctor_id = ? AND patient_id IN (SELECT patient_id FROM appointments WHERE doctor_id = ? GROUP BY patient_id HAVING COUNT(*) > 1)", doctorUserID, doctorUserID).
		Scan(&rd.Repeats)

	repeatPct := float64(0)
	if rd.Total > 0 {
		repeatPct = float64(rd.Repeats) / float64(rd.Total) * 100
	}
	avgFee := int64(0)
	if s.Completed > 0 {
		avgFee = earningsPaise / s.Completed
	}

	summary := &DoctorAnalyticsSummary{
		DoctorUserID:       doctorUserID,
		Period:             period,
		TotalConsultations: s.Total,
		CompletedConsults:  s.Completed,
		CancelledConsults:  s.Cancelled,
		AvgRating:          avgRating,
		RepeatPatientPct:   repeatPct,
		TotalEarningsPaise: earningsPaise,
		AvgConsultFeePaise: avgFee,
	}

	// Upsert
	var existing DoctorAnalyticsSummary
	err := database.DB.Where("doctor_user_id = ? AND period = ?", doctorUserID, period).First(&existing).Error
	if err != nil {
		if dbErr := database.DB.Create(summary).Error; dbErr != nil {
			return nil, fmt.Errorf("analytics: create summary: %w", dbErr)
		}
	} else {
		summary.ID = existing.ID
		if dbErr := database.DB.Save(summary).Error; dbErr != nil {
			return nil, fmt.Errorf("analytics: save summary: %w", dbErr)
		}
	}
	return summary, nil
}
