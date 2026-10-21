package devi

import (
	"time"

	"gorm.io/gorm"
)

// AdminStats holds aggregate Devi chatbot statistics for the admin dashboard.
type AdminStats struct {
	TotalSessions    int64       `json:"total_sessions"`
	ActiveSessions   int64       `json:"active_sessions"`
	MessagesToday    int64       `json:"messages_today"`
	AvgResponseMS    int         `json:"avg_response_ms"`
	SatisfactionScore float64   `json:"satisfaction_score"`
	EscalationRate   float64    `json:"escalation_rate"`
	TopIntents       []IntentStat `json:"top_intents"`
	HourlyVolume     []HourStat   `json:"hourly_volume"`
}

type IntentStat struct {
	Intent string `json:"intent"`
	Count  int64  `json:"count"`
}

type HourStat struct {
	Hour  string `json:"hour"`
	Count int64  `json:"count"`
}

// GetAdminStats returns aggregate Devi chatbot stats. All metrics come from
// real measurements — none are constants. Queries use parameterised time
// windows (no NOW()/INTERVAL/ILIKE/TO_CHAR) so the same code runs on both
// PostgreSQL (prod) and SQLite (dev).
func GetAdminStats(db *gorm.DB) (AdminStats, error) {
	var stats AdminStats
	now := time.Now()
	last24h := now.Add(-24 * time.Hour)
	last15m := now.Add(-15 * time.Minute)

	// Totals
	db.Model(&SymptomLog{}).Count(&stats.TotalSessions)
	db.Model(&SymptomLog{}).Where("created_at >= ?", last24h).Count(&stats.MessagesToday)

	// Active sessions — distinct session_id in last 15 minutes
	db.Raw(`SELECT COUNT(DISTINCT session_id) FROM devi_symptom_logs WHERE created_at >= ?`, last15m).
		Scan(&stats.ActiveSessions)

	// ─── Real Avg Response (Phase 1.3) ──────────────────────────────────────
	// Only count rows where the handler successfully measured a response time.
	// 0 means "not measured" (legacy rows, async paths) — exclude so the
	// average reflects actual AI generation time.
	var avgMs float64
	db.Model(&SymptomLog{}).
		Where("created_at >= ? AND response_ms > 0", last24h).
		Select("COALESCE(AVG(response_ms), 0)").Scan(&avgMs)
	stats.AvgResponseMS = int(avgMs)

	// ─── Real Escalation Rate (Phase 3) ─────────────────────────────────────
	// "Escalated" = symptom log with urgency >= 4 (high/critical).
	var total24h, escalated24h int64
	db.Model(&SymptomLog{}).Where("created_at >= ?", last24h).Count(&total24h)
	db.Model(&SymptomLog{}).Where("created_at >= ? AND urgency >= ?", last24h, 4).Count(&escalated24h)
	if total24h > 0 {
		stats.EscalationRate = float64(escalated24h) / float64(total24h) * 100
	}

	// ─── Satisfaction proxy via conversion (Phase 4, option c) ──────────────
	// We don't have explicit feedback yet, so use the booking-conversion rate
	// as a proxy: sessions that successfully booked an appointment. Mapped to
	// a 0–5 scale (% / 20) — a 100% booking rate ≈ 5.0. Until we add a real
	// feedback table this is the most honest signal available.
	var booked int64
	db.Model(&SymptomLog{}).Where("created_at >= ? AND booked_appt = ?", last24h, true).Count(&booked)
	if total24h > 0 {
		stats.SatisfactionScore = (float64(booked) / float64(total24h)) * 5.0
	}

	// ─── Top Intents (Phase 6) ──────────────────────────────────────────────
	// Use the real intent column written by the chat flows. No keyword guessing.
	rows, err := db.Raw(`
		SELECT intent, COUNT(*) AS count
		FROM devi_symptom_logs
		WHERE created_at >= ?
		GROUP BY intent
		ORDER BY count DESC
		LIMIT 5
	`, last24h).Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s IntentStat
			_ = rows.Scan(&s.Intent, &s.Count)
			stats.TopIntents = append(stats.TopIntents, s)
		}
	}
	if stats.TopIntents == nil {
		stats.TopIntents = []IntentStat{}
	}

	// ─── Hourly Volume (Phase 2 — cross-DB) ─────────────────────────────────
	// Fetch raw rows from the last 8 hours and bucket by hour in Go. This
	// avoids Postgres-only TO_CHAR/EXTRACT and works on SQLite too.
	last8h := now.Add(-8 * time.Hour)
	buckets := make([]int64, 8) // index 0 = 8h ago, index 7 = current hour
	startHour := time.Date(last8h.Year(), last8h.Month(), last8h.Day(), last8h.Hour(), 0, 0, 0, last8h.Location())
	var times []time.Time
	db.Model(&SymptomLog{}).
		Where("created_at >= ?", last8h).
		Pluck("created_at", &times)
	for _, t := range times {
		idx := int(t.Sub(startHour) / time.Hour)
		if idx >= 0 && idx < len(buckets) {
			buckets[idx]++
		}
	}
	stats.HourlyVolume = make([]HourStat, 0, len(buckets))
	for i, n := range buckets {
		bucketStart := startHour.Add(time.Duration(i) * time.Hour)
		stats.HourlyVolume = append(stats.HourlyVolume, HourStat{
			Hour:  bucketStart.Format("15:04"),
			Count: n,
		})
	}

	return stats, nil
}

// GetRecentSymptomLogs returns the most recent symptom logs across all users.
func GetRecentSymptomLogs(db *gorm.DB, limit int) ([]SymptomLog, error) {
	var logs []SymptomLog
	err := db.Order("created_at desc").Limit(limit).Find(&logs).Error
	return logs, err
}

// DeviConfig persists admin-controlled feature toggles for the Devi engine.
// One row per setting; key is unique. Read at engine startup; invalidated
// by PATCH /admin/devi/config and re-fetched on next request.
type DeviConfig struct {
	Key       string    `gorm:"primaryKey"    json:"key"`
	Value     string    `gorm:"type:text"     json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DeviConfig) TableName() string { return "devi_configs" }

// LearnedIntent represents a phrase the LLM successfully resolved to an intent.
// Admins review pending phrases and approve them to automatically inject them into the Vector Router.
type LearnedIntent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Phrase    string    `gorm:"type:text;uniqueIndex" json:"phrase"`
	Intent    string    `gorm:"index"      json:"intent"`
	Status    string    `gorm:"index;default:'pending'" json:"status"` // pending, approved, rejected
	Embedding string    `gorm:"type:text"  json:"-"` // Stored for fast injection on startup
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (LearnedIntent) TableName() string { return "devi_learned_intents" }

// AutoMigrate runs table migrations.
// On PostgreSQL, SetupPgVector should be called afterwards to enable the vector extension.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&SymptomLog{},
		&GuestSession{},
		&WhatsAppUserLink{},
		&KnowledgeItem{},
		&DeviConfig{},
		&LearnedIntent{},
	)
}

// KnowledgeItem stores medical protocols, service FAQs, and health information
// for semantic retrieval (RAG) by the AI Agent.
type KnowledgeItem struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Category  string    `gorm:"index"      json:"category"` // e.g. "protocol", "faq", "service"
	Title     string    `gorm:"type:text"  json:"title"`
	Content   string    `gorm:"type:text"  json:"content"`
	Source    string    `json:"source,omitempty"`
	// Embedding stored as JSON text for SQLite compat.
	Embedding string    `gorm:"type:text"  json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (KnowledgeItem) TableName() string { return "knowledge_items" }


// ParticipantType identifies who triggered a Devi session.
type ParticipantType string

const (
	ParticipantPrimary      ParticipantType = "primary_user"
	ParticipantFamilyMember ParticipantType = "family_member"
	ParticipantGuest        ParticipantType = "guest"
)

// SymptomLog persists every symptom-check interaction Devi handles.
// The Embedding column stores a JSON-serialised []float32 (768-dim Gemini vector).
// On PostgreSQL with pgvector, SetupPgVector() alters the column to a native
// vector(768) type so the index and <=> operator are available.
type SymptomLog struct {
	ID              uint            `gorm:"primaryKey"                     json:"id"`
	UserID          uint            `gorm:"index"                          json:"user_id"`     // 0 for pure guests
	FamilyMemberID  *uint           `gorm:"index"                          json:"family_member_id,omitempty"`
	GuestBrowserID  *string         `gorm:"index"                          json:"guest_browser_id,omitempty"`
	ParticipantType ParticipantType `gorm:"default:'primary_user'"         json:"participant_type"`
	SessionID       string          `gorm:"index"                          json:"session_id"`
	Intent          string          `gorm:"index;default:'symptom_check'"  json:"intent"`
	SymptomsText    string          `gorm:"type:text"                      json:"symptoms_text"`
	DeptMatched     string          `json:"dept_matched"`
	Urgency         int             `json:"urgency"`
	// Embedding stored as JSON text for SQLite compat.
	// On Postgres with pgvector this column is cast to vector(768) by SetupPgVector.
	Embedding  string    `gorm:"type:vector(768)"  json:"-"`
	BookedAppt bool      `gorm:"default:false" json:"booked_appt"`
	// ResponseMS is the wall-clock milliseconds it took Devi to generate a reply
	// for this turn. Populated by the chat handler around the AI call. 0 means
	// "not measured" (legacy rows, async paths). Surfaced in the admin Avg Response stat.
	ResponseMS int       `gorm:"default:0" json:"response_ms"`
	CreatedAt  time.Time `json:"created_at"`
}

func (SymptomLog) TableName() string { return "devi_symptom_logs" }

// GuestSession tracks unauthenticated Devi users server-side.
// The browser_id is a UUID generated by the frontend and stored in localStorage,
// but is also registered here so the session survives a localStorage clear.
type GuestSession struct {
	ID                  uint           `gorm:"primaryKey"         json:"id"`
	BrowserID           string         `gorm:"uniqueIndex;not null" json:"browser_id"`
	Nickname            *string        `json:"nickname,omitempty"` // provided voluntarily
	Phone               *string        `gorm:"index"              json:"phone,omitempty"`
	ConvertedToUserID   *uint          `gorm:"index"              json:"converted_to_user_id,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	LastSeenAt          time.Time      `json:"last_seen_at"`
	ExpiresAt           time.Time      `json:"expires_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index"              json:"-"`
}

func (GuestSession) TableName() string { return "devi_guest_sessions" }

// WhatsAppUserLink ties a WhatsApp phone number to a platform user account.
// Created during the WA phone-linking OTP flow.
type WhatsAppUserLink struct {
	ID             uint       `gorm:"primaryKey"         json:"id"`
	Phone          string     `gorm:"uniqueIndex;not null" json:"phone"` // E.164 format
	UserID         *uint      `gorm:"index"              json:"user_id,omitempty"`
	FamilyMemberID *uint      `json:"family_member_id,omitempty"`
	Verified       bool       `gorm:"default:false"      json:"verified"`
	OTPCode        *string    `json:"-"`
	OTPExpiresAt   *time.Time `json:"-"`
	LinkedAt       *time.Time `json:"linked_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (WhatsAppUserLink) TableName() string { return "whatsapp_user_links" }
