package anomaly

import (
	"encoding/json"
	"fmt"
	"time"

	"drayya-core/internal/audit"

	"gorm.io/gorm"
)

// Rule is one anomaly-detection check that runs over the recent window
// of audit_logs and returns 0..N detected Anomalies.
//
// Rules should be:
//   - PURE — never mutate the DB outside of returning Anomaly structs
//   - FAST — must complete on a 15-min window within ~100ms
//   - DEDUPED — see dedupeKey() and the scanner's seen-set
type Rule struct {
	Name        string
	Severity    string
	Description string // human-readable explanation of what this rule catches
	Detect      func(db *gorm.DB, since time.Time) ([]Anomaly, error)
}

// Registry — all rules evaluated on every scan tick.
// Add a new check by appending one entry. Order doesn't matter.
var Registry = []Rule{
	patientHardDeleteRule(),
	bulkDeleteRule("pharmacy_inventory", 20, SeverityWarning),
	bulkDeleteRule("medicine", 10, SeverityWarning),
	bulkDeleteRule("appointments", 15, SeverityCritical),
	feeJumpRule(5.0, SeverityWarning),  // 5x or more
	feeJumpRule(10.0, SeverityCritical), // 10x or more
	highVolumeActorRule(200, SeverityWarning),
	offHoursAdminRule(),
}

// ── Rule implementations ─────────────────────────────────────────────────────

// patientHardDeleteRule — ANY hard delete from the user table where role=patient
// should never happen in normal operation (soft-delete only).
func patientHardDeleteRule() Rule {
	return Rule{
		Name:        "patient_record_deleted",
		Severity:    SeverityCritical,
		Description: "A patient record was hard-deleted from the users table",
		Detect: func(db *gorm.DB, since time.Time) ([]Anomaly, error) {
			rows, err := scanLogs(db, since, `action = ? AND table_name IN ?`, "DELETE", []string{"user", "users", "patient_profiles"})
			if err != nil || len(rows) == 0 {
				return nil, err
			}
			var out []Anomaly
			for _, r := range rows {
				// Heuristic: only flag if the before-state mentions role=patient
				if hasField(r.Before, "role", "patient") {
					out = append(out, Anomaly{
						RuleName:    "patient_record_deleted",
						Severity:    SeverityCritical,
						Title:       fmt.Sprintf("Patient record DELETED — user #%s", r.RowID),
						Description: fmt.Sprintf("Patient row in %s deleted by actor #%d (%s) at %s",
							r.Table, r.ActorID, r.ActorRole, r.CreatedAt.Format(time.RFC3339)),
						ActorID:    r.ActorID,
						ActorRole:  r.ActorRole,
						Table:     r.Table,
						Hits:       1,
						Evidence:   mustJSON(map[string]interface{}{"audit_log_id": r.ID, "row_id": r.RowID}),
						DetectedAt: time.Now(),
					})
				}
			}
			return out, nil
		},
	}
}

// bulkDeleteRule — same actor + same table + ≥threshold DELETE rows in window
func bulkDeleteRule(table string, threshold int, severity string) Rule {
	return Rule{
		Name:        fmt.Sprintf("bulk_delete_%s", table),
		Severity:    severity,
		Description: fmt.Sprintf("Same actor deleted ≥%d rows from %s within the scan window", threshold, table),
		Detect: func(db *gorm.DB, since time.Time) ([]Anomaly, error) {
			type group struct {
				ActorID   uint
				ActorRole string
				N         int
				FirstID   uint64
				LastID    uint64
			}
			var groups []group
			err := db.Table("audit_logs").
				Select("actor_id, actor_role, COUNT(*) AS n, MIN(id) AS first_id, MAX(id) AS last_id").
				Where("action = ? AND table_name = ? AND created_at >= ?", "DELETE", table, since).
				Group("actor_id, actor_role").
				Having("COUNT(*) >= ?", threshold).
				Scan(&groups).Error
			if err != nil {
				return nil, err
			}
			out := make([]Anomaly, 0, len(groups))
			for _, g := range groups {
				out = append(out, Anomaly{
					RuleName:    fmt.Sprintf("bulk_delete_%s", table),
					Severity:    severity,
					Title:       fmt.Sprintf("Bulk delete: %d rows from %s by actor #%d", g.N, table, g.ActorID),
					Description: fmt.Sprintf("Actor #%d (%s) deleted %d %s rows in the scan window", g.ActorID, g.ActorRole, g.N, table),
					ActorID:     g.ActorID,
					ActorRole:   g.ActorRole,
					Table:       table,
					Hits:        g.N,
					Evidence:    mustJSON(map[string]interface{}{"first_audit_log_id": g.FirstID, "last_audit_log_id": g.LastID}),
					DetectedAt:  time.Now(),
				})
			}
			return out, nil
		},
	}
}

// feeJumpRule — doctor consultation_fee changed by factor ≥ multiplier in one UPDATE
func feeJumpRule(multiplier float64, severity string) Rule {
	return Rule{
		Name:        fmt.Sprintf("doctor_fee_jump_%dx", int(multiplier)),
		Severity:    severity,
		Description: fmt.Sprintf("Doctor consultation_fee changed by %.0fx or more in a single update", multiplier),
		Detect: func(db *gorm.DB, since time.Time) ([]Anomaly, error) {
			rows, err := scanLogs(db, since, `action = ? AND table_name = ? AND diff IS NOT NULL`, "UPDATE", "doctorprofile")
			if err != nil || len(rows) == 0 {
				return nil, err
			}
			var out []Anomaly
			for _, r := range rows {
				oldV, newV, ok := readDiffPair(r.Diff, "consultation_fee")
				if !ok || oldV <= 0 || newV <= 0 {
					continue
				}
				ratio := newV / oldV
				if ratio < 1 {
					ratio = oldV / newV
				}
				if ratio >= multiplier {
					out = append(out, Anomaly{
						RuleName:    fmt.Sprintf("doctor_fee_jump_%dx", int(multiplier)),
						Severity:    severity,
						Title:       fmt.Sprintf("Doctor fee changed %.1fx — ₹%.2f → ₹%.2f", ratio, oldV/100, newV/100),
						Description: fmt.Sprintf("doctorprofile #%s consultation_fee changed by actor #%d (%s)", r.RowID, r.ActorID, r.ActorRole),
						ActorID:     r.ActorID,
						ActorRole:   r.ActorRole,
						Table:      r.Table,
						Hits:        1,
						Evidence:    mustJSON(map[string]interface{}{"audit_log_id": r.ID, "row_id": r.RowID, "old": oldV, "new": newV, "ratio": ratio}),
						DetectedAt:  time.Now(),
					})
				}
			}
			return out, nil
		},
	}
}

// highVolumeActorRule — any actor producing > threshold audit rows in the
// scan window. Catches runaway scripts, looped UI, and abuse.
func highVolumeActorRule(threshold int, severity string) Rule {
	return Rule{
		Name:        "high_volume_actor",
		Severity:    severity,
		Description: fmt.Sprintf("One actor produced ≥%d audit-log rows within the scan window", threshold),
		Detect: func(db *gorm.DB, since time.Time) ([]Anomaly, error) {
			type group struct {
				ActorID   uint
				ActorRole string
				N         int
			}
			var groups []group
			err := db.Table("audit_logs").
				Select("actor_id, actor_role, COUNT(*) AS n").
				Where("actor_id > 0 AND created_at >= ?", since).
				Group("actor_id, actor_role").
				Having("COUNT(*) >= ?", threshold).
				Scan(&groups).Error
			if err != nil {
				return nil, err
			}
			out := make([]Anomaly, 0, len(groups))
			for _, g := range groups {
				out = append(out, Anomaly{
					RuleName:    "high_volume_actor",
					Severity:    severity,
					Title:       fmt.Sprintf("High activity: actor #%d produced %d audit rows", g.ActorID, g.N),
					Description: fmt.Sprintf("Actor #%d (%s) generated %d mutations in the scan window — investigate for runaway scripts or abuse", g.ActorID, g.ActorRole, g.N),
					ActorID:     g.ActorID,
					ActorRole:   g.ActorRole,
					Hits:        g.N,
					Evidence:    mustJSON(map[string]interface{}{"count": g.N}),
					DetectedAt:  time.Now(),
				})
			}
			return out, nil
		},
	}
}

// offHoursAdminRule — admin-role mutations between 2am-5am IST (3-hour quiet window)
// Wake-up flag: this is suspicious unless explicitly scheduled.
func offHoursAdminRule() Rule {
	return Rule{
		Name:        "off_hours_admin_change",
		Severity:    SeverityInfo,
		Description: "Admin-role mutation occurred between 2am and 5am IST",
		Detect: func(db *gorm.DB, since time.Time) ([]Anomaly, error) {
			rows, err := scanLogs(db, since, `actor_role IN ?`, []string{"admin", "super_admin"})
			if err != nil || len(rows) == 0 {
				return nil, err
			}
			ist, _ := time.LoadLocation("Asia/Kolkata")
			var out []Anomaly
			for _, r := range rows {
				h := r.CreatedAt.In(ist).Hour()
				if h >= 2 && h < 5 {
					out = append(out, Anomaly{
						RuleName:    "off_hours_admin_change",
						Severity:    SeverityInfo,
						Title:       fmt.Sprintf("Off-hours %s on %s by admin #%d at %02d:%02d IST", r.Action, r.Table, r.ActorID, h, r.CreatedAt.In(ist).Minute()),
						Description: fmt.Sprintf("Admin activity in the 2-5am IST quiet window is rare. Verify this was intentional."),
						ActorID:     r.ActorID,
						ActorRole:   r.ActorRole,
						Table:      r.Table,
						Hits:        1,
						Evidence:    mustJSON(map[string]interface{}{"audit_log_id": r.ID, "hour_ist": h, "action": r.Action}),
						DetectedAt:  time.Now(),
					})
				}
			}
			return out, nil
		},
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// scanLogs pulls recent audit_logs matching a WHERE clause. Limited to 1000
// rows per scan to keep memory bounded — rules with more than 1000 hits in a
// 15-min window are almost certainly a different kind of problem.
func scanLogs(db *gorm.DB, since time.Time, where string, args ...interface{}) ([]audit.AuditLog, error) {
	var rows []audit.AuditLog
	cond := append([]interface{}{since}, args...)
	err := db.Where("created_at >= ? AND "+where, cond...).
		Order("id desc").Limit(1000).Find(&rows).Error
	return rows, err
}

// readDiffPair extracts {field: [old, new]} where both are numeric (paise).
// Returns ok=false when the field is missing or not numeric.
func readDiffPair(raw []byte, field string) (oldV, newV float64, ok bool) {
	if len(raw) == 0 {
		return 0, 0, false
	}
	var diff map[string]interface{}
	if err := json.Unmarshal(raw, &diff); err != nil {
		return 0, 0, false
	}
	pair, exists := diff[field].([]interface{})
	if !exists || len(pair) != 2 {
		return 0, 0, false
	}
	oldF, oOk := toFloat(pair[0])
	newF, nOk := toFloat(pair[1])
	if !oOk || !nOk {
		return 0, 0, false
	}
	return oldF, newF, true
}

func toFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	}
	return 0, false
}

// hasField returns true when the JSON blob (audit before/after) has the given
// field with the given string value.
func hasField(raw []byte, field, value string) bool {
	if len(raw) == 0 {
		return false
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	s, ok := m[field].(string)
	return ok && s == value
}

// dedupeKey returns a stable key for an anomaly so the scanner won't write
// the same Anomaly twice in consecutive scans.
func dedupeKey(a Anomaly) string {
	return fmt.Sprintf("%s|%d|%s|%s",
		a.RuleName, a.ActorID, a.Table, string(a.Evidence))
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
