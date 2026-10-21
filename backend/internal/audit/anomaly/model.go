// Package anomaly is the watchdog layer on top of audit_logs.
// It periodically scans recent audit activity for suspicious patterns
// (mass deletes, off-hours admin edits, fee spikes, etc.) and persists
// them to audit_anomalies for review.
//
// See AUDIT_LOG_ARCHITECTURE.md for the full design.
package anomaly

import (
	"time"

	"gorm.io/datatypes"
)

// Severity levels — used for filtering + alert routing.
//   info     → ignore unless you're looking for it; daily-digest only
//   warning  → reviewer attention; daily-digest highlighted
//   critical → page someone NOW; sent via real-time email + Slack
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// Status of an anomaly entry — workflow state for the admin UI.
const (
	StatusNew      = "new"      // just detected
	StatusResolved = "resolved" // admin clicked "mark resolved"
)

// Anomaly is one detected suspicious pattern in audit_logs.
// Multiple audit_log rows can roll up into a single Anomaly (see Hits count).
type Anomaly struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	RuleName    string         `gorm:"index;size:64" json:"rule_name"`
	Severity    string         `gorm:"index;size:10" json:"severity"`
	Title       string         `gorm:"size:255" json:"title"`
	Description string         `gorm:"type:text" json:"description"`
	ActorID     uint           `gorm:"index" json:"actor_id"`
	ActorRole   string         `gorm:"size:32" json:"actor_role"`
	Table       string         `gorm:"index;size:64;column:table_name" json:"table_name"`
	Hits        int            `json:"hits"` // how many audit_log rows contributed
	Evidence    datatypes.JSON `json:"evidence"` // sample audit_log IDs + key facts
	Status      string         `gorm:"index;size:16;default:'new'" json:"status"`
	DetectedAt  time.Time      `gorm:"index" json:"detected_at"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
	ResolvedBy  uint           `json:"resolved_by,omitempty"`
	Notes       string         `gorm:"type:text" json:"notes"`
}

func (Anomaly) TableName() string { return "audit_anomalies" }

// AuditSkip excludes the anomaly table from audit logging (it's derived data
// — auditing the derived view would create noise).
func (Anomaly) AuditSkip() bool { return true }
