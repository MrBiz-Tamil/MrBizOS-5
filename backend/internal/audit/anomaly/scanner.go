package anomaly

import (
	"context"
	"log"
	"sync"
	"time"

	"drayya-core/internal/audit"
	"drayya-core/internal/database"

	"gorm.io/gorm"
)

// Notifier is the callback fired the moment a critical-severity anomaly is
// detected. Wired by Day 3 to send an email + Slack/Telegram alert.
// Defaults to a no-op; set with SetNotifier from the alerts package.
type Notifier func(a Anomaly)

var (
	notifier   Notifier = func(Anomaly) {}
	notifierMu sync.RWMutex
)

func SetNotifier(n Notifier) {
	notifierMu.Lock()
	defer notifierMu.Unlock()
	notifier = n
}

// AutoMigrate ensures the audit_anomalies table exists.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Anomaly{})
}

// StartScanner launches a background goroutine that runs every interval,
// evaluates all rules in Registry, and persists newly-detected Anomalies.
//
// Call once from server startup:
//
//	go anomaly.StartScanner(15 * time.Minute)
//
// The scan window is `2 * interval` so brief overlaps don't miss events,
// while the in-table deduplication prevents the same anomaly being written
// twice across consecutive scans.
func StartScanner(interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	log.Printf("[anomaly] scanner started — interval %s, %d rules registered", interval, len(Registry))

	// Run one immediately on startup so admins see anything that piled up
	// while the scanner was offline.
	scan(interval * 2)

	tick := time.NewTicker(interval)
	defer tick.Stop()
	for range tick.C {
		scan(interval * 2)
	}
}

func scan(window time.Duration) {
	ctx := audit.ForSystem(context.Background(), "anomaly:scan")
	db := database.DB.WithContext(ctx)

	since := time.Now().Add(-window)
	total := 0
	for _, rule := range Registry {
		found, err := rule.Detect(db, since)
		if err != nil {
			log.Printf("[anomaly] rule %s failed: %v", rule.Name, err)
			continue
		}
		for _, a := range found {
			if writeIfNew(db, a) {
				total++
				if a.Severity == SeverityCritical {
					notifierMu.RLock()
					n := notifier
					notifierMu.RUnlock()
					n(a)
				}
			}
		}
	}
	if total > 0 {
		log.Printf("[anomaly] scan complete — %d new anomalies recorded", total)
	}
}

// Record persists an externally-produced anomaly (e.g. from the finance
// reconciliation job) through the same dedupe + critical-notify path the rule
// scanner uses. Returns true if a new row was written. Lets other subsystems
// surface findings in the same Anomalies dashboard without re-implementing
// dedup or alerting.
func Record(a Anomaly) bool {
	db := database.DB
	if !writeIfNew(db, a) {
		return false
	}
	if a.Severity == SeverityCritical {
		notifierMu.RLock()
		n := notifier
		notifierMu.RUnlock()
		n(a)
	}
	return true
}

// writeIfNew persists the anomaly UNLESS an unresolved row with the same
// dedupe key was created in the last hour. Prevents the same flag firing
// every scan tick.
func writeIfNew(db *gorm.DB, a Anomaly) bool {
	key := dedupeKey(a)
	cutoff := time.Now().Add(-time.Hour)

	var existing int64
	db.Model(&Anomaly{}).
		Where("rule_name = ? AND actor_id = ? AND table_name = ? AND status = ? AND detected_at >= ?",
			a.RuleName, a.ActorID, a.TableName, StatusNew, cutoff).
		Count(&existing)
	if existing > 0 {
		_ = key // dedupe is via the WHERE clause; key is kept for future debugging
		return false
	}

	a.Status = StatusNew
	if err := db.Create(&a).Error; err != nil {
		log.Printf("[anomaly] write failed: %v", err)
		return false
	}
	return true
}
