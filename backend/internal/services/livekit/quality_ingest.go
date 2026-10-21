package livekit

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

// CallQualityReport stores the quality metrics for a video consultation
type CallQualityReport struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	AppointmentID   uint      `gorm:"index;not null" json:"appointment_id"`
	RoomName        string    `gorm:"index" json:"room_name"`
	ParticipantID   string    `json:"participant_id"`
	ParticipantRole string    `json:"participant_role"` // doctor, patient
	// Network metrics
	AvgBitrateKbps  float64   `json:"avg_bitrate_kbps"`
	AvgPacketLoss   float64   `json:"avg_packet_loss_pct"` // 0–100
	AvgJitterMs     float64   `json:"avg_jitter_ms"`
	AvgRTTMs        float64   `json:"avg_rtt_ms"`
	// Video metrics
	AvgFramerate    float64   `json:"avg_framerate"`
	AvgResolution   string    `json:"avg_resolution"` // e.g. "640x480"
	VideoCodec      string    `json:"video_codec"`    // VP8, VP9, H264, AV1
	// Audio metrics
	AudioCodec      string    `json:"audio_codec"` // OPUS
	AudioLevel      float64   `json:"avg_audio_level_db"`
	// Call events
	Reconnects      int       `json:"reconnect_count"`
	Disconnects     int       `json:"disconnect_count"`
	CallDurationSec int       `json:"call_duration_sec"`
	// Quality score (computed)
	QualityScore    float64   `json:"quality_score"` // 0–5 (like MOS)
	QualityGrade    string    `json:"quality_grade"` // excellent, good, fair, poor, bad
	// Raw payload
	RawMetrics      string    `gorm:"type:text" json:"raw_metrics,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

func (CallQualityReport) TableName() string {
	return "call_quality_reports"
}

// RoomStats is the payload received from LiveKit webhooks or polled from the SDK
type RoomStats struct {
	RoomName      string            `json:"room_name"`
	AppointmentID uint              `json:"appointment_id"`
	Participants  []ParticipantStat `json:"participants"`
	StartedAt     time.Time         `json:"started_at"`
	EndedAt       time.Time         `json:"ended_at"`
}

type ParticipantStat struct {
	ParticipantID string  `json:"participant_id"`
	Role          string  `json:"role"`
	BitrateKbps   float64 `json:"bitrate_kbps"`
	PacketLoss    float64 `json:"packet_loss_pct"`
	JitterMs      float64 `json:"jitter_ms"`
	RTTMs         float64 `json:"rtt_ms"`
	Framerate     float64 `json:"framerate"`
	Resolution    string  `json:"resolution"`
	VideoCodec    string  `json:"video_codec"`
	AudioCodec    string  `json:"audio_codec"`
	AudioLevel    float64 `json:"audio_level_db"`
	Reconnects    int     `json:"reconnects"`
	Disconnects   int     `json:"disconnects"`
}

// QualityIngestService handles incoming call quality data from LiveKit
type QualityIngestService struct {
	db *gorm.DB
}

// NewQualityIngestService creates a new quality ingestion service
func NewQualityIngestService(db *gorm.DB) *QualityIngestService {
	return &QualityIngestService{db: db}
}

// IngestRoomStats processes quality stats for a completed call
func (s *QualityIngestService) IngestRoomStats(stats *RoomStats) error {
	if stats == nil {
		return fmt.Errorf("nil stats received")
	}

	callDuration := int(stats.EndedAt.Sub(stats.StartedAt).Seconds())
	if callDuration < 0 {
		callDuration = 0
	}

	raw, _ := json.Marshal(stats)

	for _, p := range stats.Participants {
		score, grade := computeQualityScore(p)

		report := &CallQualityReport{
			AppointmentID:   stats.AppointmentID,
			RoomName:        stats.RoomName,
			ParticipantID:   p.ParticipantID,
			ParticipantRole: p.Role,
			AvgBitrateKbps:  p.BitrateKbps,
			AvgPacketLoss:   p.PacketLoss,
			AvgJitterMs:     p.JitterMs,
			AvgRTTMs:        p.RTTMs,
			AvgFramerate:    p.Framerate,
			AvgResolution:   p.Resolution,
			VideoCodec:      p.VideoCodec,
			AudioCodec:      p.AudioCodec,
			AudioLevel:      p.AudioLevel,
			Reconnects:      p.Reconnects,
			Disconnects:     p.Disconnects,
			CallDurationSec: callDuration,
			QualityScore:    score,
			QualityGrade:    grade,
			RawMetrics:      string(raw),
		}

		if err := s.db.Create(report).Error; err != nil {
			log.Printf("[LiveKit] Failed to save quality report for participant %s: %v", p.ParticipantID, err)
			return err
		}
	}

	// Update appointment with call quality grade (worst of doctor/patient)
	var worstGrade string
	s.db.Raw(`
		SELECT quality_grade FROM call_quality_reports
		WHERE appointment_id = ?
		ORDER BY quality_score ASC LIMIT 1
	`, stats.AppointmentID).Scan(&worstGrade)

	if worstGrade != "" {
		s.db.Exec("UPDATE appointments SET call_quality = ? WHERE id = ?", worstGrade, stats.AppointmentID)
	}

	return nil
}

// GetRoomQuality returns quality reports for a specific appointment
func (s *QualityIngestService) GetRoomQuality(appointmentID uint) ([]CallQualityReport, error) {
	var reports []CallQualityReport
	err := s.db.Where("appointment_id = ?", appointmentID).Find(&reports).Error
	return reports, err
}

// GetAggregateQuality returns platform-wide quality statistics for a period
func (s *QualityIngestService) GetAggregateQuality(startDate, endDate time.Time) (map[string]interface{}, error) {
	var stats struct {
		AvgScore      float64
		TotalCalls    int
		ExcellentPct  float64
		GoodPct       float64
		FairPct       float64
		PoorPct       float64
		BadPct        float64
		AvgDurationMin float64
	}

	s.db.Raw(`
		SELECT
			AVG(quality_score) AS avg_score,
			COUNT(DISTINCT appointment_id) AS total_calls,
			100.0 * SUM(CASE WHEN quality_grade = 'excellent' THEN 1 ELSE 0 END) / COUNT(*) AS excellent_pct,
			100.0 * SUM(CASE WHEN quality_grade = 'good' THEN 1 ELSE 0 END) / COUNT(*) AS good_pct,
			100.0 * SUM(CASE WHEN quality_grade = 'fair' THEN 1 ELSE 0 END) / COUNT(*) AS fair_pct,
			100.0 * SUM(CASE WHEN quality_grade = 'poor' THEN 1 ELSE 0 END) / COUNT(*) AS poor_pct,
			100.0 * SUM(CASE WHEN quality_grade = 'bad' THEN 1 ELSE 0 END) / COUNT(*) AS bad_pct,
			AVG(call_duration_sec) / 60.0 AS avg_duration_min
		FROM call_quality_reports
		WHERE created_at BETWEEN ? AND ?
	`, startDate, endDate).Scan(&stats)

	return map[string]interface{}{
		"period":           fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		"avg_quality_score": stats.AvgScore,
		"total_calls":      stats.TotalCalls,
		"grade_distribution": map[string]float64{
			"excellent": stats.ExcellentPct,
			"good":      stats.GoodPct,
			"fair":      stats.FairPct,
			"poor":      stats.PoorPct,
			"bad":       stats.BadPct,
		},
		"avg_call_duration_min": stats.AvgDurationMin,
	}, nil
}

// AutoMigrate creates the call_quality_reports table
func (s *QualityIngestService) AutoMigrate() error {
	return s.db.AutoMigrate(&CallQualityReport{})
}

// computeQualityScore computes a MOS-like score (1–5) from network metrics
// Based on ITU-T E-model inspired heuristics
func computeQualityScore(p ParticipantStat) (float64, string) {
	score := 5.0

	// Penalize for packet loss (most impactful)
	if p.PacketLoss > 0 {
		switch {
		case p.PacketLoss > 10:
			score -= 2.5
		case p.PacketLoss > 5:
			score -= 1.5
		case p.PacketLoss > 2:
			score -= 0.8
		case p.PacketLoss > 0.5:
			score -= 0.3
		}
	}

	// Penalize for high latency (RTT)
	switch {
	case p.RTTMs > 400:
		score -= 1.5
	case p.RTTMs > 200:
		score -= 0.8
	case p.RTTMs > 100:
		score -= 0.3
	}

	// Penalize for jitter
	switch {
	case p.JitterMs > 50:
		score -= 0.8
	case p.JitterMs > 30:
		score -= 0.4
	case p.JitterMs > 15:
		score -= 0.1
	}

	// Penalize for low bitrate
	if p.BitrateKbps > 0 && p.BitrateKbps < 100 {
		score -= 0.5
	}

	// Penalize for reconnects
	score -= float64(p.Reconnects) * 0.3

	// Clamp to [1, 5]
	if score < 1.0 {
		score = 1.0
	}
	if score > 5.0 {
		score = 5.0
	}

	var grade string
	switch {
	case score >= 4.5:
		grade = "excellent"
	case score >= 3.5:
		grade = "good"
	case score >= 2.5:
		grade = "fair"
	case score >= 1.5:
		grade = "poor"
	default:
		grade = "bad"
	}

	return score, grade
}
