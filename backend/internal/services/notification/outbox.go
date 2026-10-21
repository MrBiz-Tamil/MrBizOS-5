package notification

// outbox.go — NotificationJob model, enqueue, claim, and state-transition helpers.
//
// The outbox pattern guarantees at-least-once delivery: jobs are written to
// Postgres before any network call is attempted.  The worker retries failed
// jobs with exponential back-off until MaxAttempts is reached, at which point
// the job is marked "dead" for manual inspection.
//
// Multi-instance safety: claimNextJob wraps the SELECT … FOR UPDATE SKIP LOCKED
// + status-flip inside a single transaction so two replicas never race on the
// same row.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormlogger "gorm.io/gorm/logger"
)

// silentGormLogger suppresses noisy "record not found" logs that the outbox
// poller produces on every empty-queue poll (every 10s). Real errors during
// claim/update are still surfaced because we return them to the caller, which
// has its own structured logging.
var silentGormLogger = gormlogger.Default.LogMode(gormlogger.Silent)

// NotificationJob is a single outbound notification that must be delivered
// reliably.  Table name: notification_jobs.
type NotificationJob struct {
	ID uint `gorm:"primaryKey"`

	// Channel is the delivery transport (email / whatsapp).
	Channel Channel `gorm:"not null;size:32;index"`

	// JobType is a human-readable event name used for logging and filtering
	// (e.g. "mfa_otp", "welcome", "appointment_confirmed").
	JobType string `gorm:"not null;size:64;index"`

	// Recipient is the destination address: email for ChannelEmail,
	// E.164 phone number (without leading +) for ChannelWhatsApp.
	Recipient string `gorm:"not null;size:320"`

	// UserID optionally links the job to a user row for audit purposes.
	UserID *uint `gorm:"index"`

	// Email-only fields
	Subject string `gorm:"size:255"`
	Body    string `gorm:"type:text"` // pre-rendered HTML

	// WhatsApp template fields
	TemplateID string `gorm:"size:128"` // approved Meta template name
	Params     string `gorm:"type:text"` // JSON-encoded []string of template body params

	// Processing state machine: pending → processing → sent | dead
	Status      string `gorm:"not null;default:'pending';size:16;index"`
	Attempts    int    `gorm:"not null;default:0"`
	MaxAttempts int    `gorm:"not null;default:3"`
	LastError   string `gorm:"type:text"`

	// Scheduling
	NextAttemptAt time.Time  `gorm:"not null;index"`
	SentAt        *time.Time

	CreatedAt time.Time
}

// AutoMigrateOutbox creates or updates the notification_jobs table.
func AutoMigrateOutbox(db *gorm.DB) error {
	return db.AutoMigrate(&NotificationJob{})
}

// enqueue persists a new job that is immediately due for processing.
func enqueue(ctx context.Context, db *gorm.DB, job *NotificationJob) error {
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3
	}
	if job.NextAttemptAt.IsZero() {
		job.NextAttemptAt = time.Now()
	}
	job.Status = "pending"
	return db.WithContext(ctx).Create(job).Error
}

// claimNextJob atomically selects the oldest pending due job and flips its
// status to "processing" inside a transaction.  Returns gorm.ErrRecordNotFound
// when the queue is empty.
//
// Note: an empty queue is the common case (worker polls every 10s). We silence
// GORM's default "record not found" log noise for this query via
// Session{Logger: silentLogger} so production logs stay clean.
func claimNextJob(ctx context.Context, db *gorm.DB) (*NotificationJob, error) {
	var job NotificationJob
	// Use a session with a silent logger so empty-queue polls don't spam logs.
	silent := db.Session(&gorm.Session{Logger: silentGormLogger})

	err := silent.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = 'pending' AND next_attempt_at <= ?", time.Now()).
			Order("next_attempt_at ASC").
			First(&job).Error; err != nil {
			return err
		}
		return tx.Model(&job).Update("status", "processing").Error
	})
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// markSent records a successful delivery.
func markSent(ctx context.Context, db *gorm.DB, id uint) {
	now := time.Now()
	db.WithContext(ctx).
		Model(&NotificationJob{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     "sent",
			"sent_at":    &now,
			"last_error": "",
		})
}

// markFailed increments the attempt counter and either re-queues with
// exponential back-off or marks the job "dead" when MaxAttempts is reached.
func markFailed(ctx context.Context, db *gorm.DB, job *NotificationJob, sendErr error) {
	job.Attempts++
	errMsg := sendErr.Error()
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	updates := map[string]interface{}{
		"attempts":   job.Attempts,
		"last_error": errMsg,
	}
	if job.Attempts >= job.MaxAttempts {
		updates["status"] = "dead"
	} else {
		updates["status"] = "pending"
		updates["next_attempt_at"] = time.Now().Add(backoffDuration(job.Attempts))
	}
	db.WithContext(ctx).
		Model(&NotificationJob{}).
		Where("id = ?", job.ID).
		Updates(updates)
}

// backoffDuration returns how long to wait before the next attempt.
// attempt=1 → 1 min, attempt=2 → 5 min, attempt≥3 → 30 min.
func backoffDuration(attempt int) time.Duration {
	switch attempt {
	case 1:
		return 1 * time.Minute
	case 2:
		return 5 * time.Minute
	default:
		return 30 * time.Minute
	}
}

// jobSummary returns a compact string for log lines.
func jobSummary(job *NotificationJob) string {
	return fmt.Sprintf("job#%d type=%s channel=%s to=%s attempt=%d/%d",
		job.ID, job.JobType, job.Channel, job.Recipient, job.Attempts+1, job.MaxAttempts)
}

// ── Params helpers ────────────────────────────────────────────────────────────

// SliceToParams serialises a []string into the JSON params field.
func SliceToParams(params []string) string {
	if len(params) == 0 {
		return ""
	}
	b, _ := json.Marshal(params)
	return string(b)
}

// ParamsToSlice deserialises the JSON params field back to []string.
func ParamsToSlice(params string) []string {
	if params == "" {
		return nil
	}
	var s []string
	_ = json.Unmarshal([]byte(params), &s)
	return s
}
