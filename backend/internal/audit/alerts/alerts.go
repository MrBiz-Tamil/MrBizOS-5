// Package alerts pushes audit signals out of the system — daily digest emails
// at 08:00 IST and real-time emails for critical anomalies.
//
// Wired by cmd/server/main.go after the anomaly scanner starts:
//
//	anomaly.SetNotifier(alerts.OnCriticalAnomaly)
//	go alerts.RunDailyDigest()
//
// Recipient: AUDIT_ALERT_EMAIL env var, falls back to a compile-time default
// so digest never silently disappears in misconfigured deployments.
package alerts

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"drayya-core/internal/audit"
	"drayya-core/internal/audit/anomaly"
	"drayya-core/internal/database"
	"drayya-core/internal/services/email"
)

// DefaultRecipient is the fallback when AUDIT_ALERT_EMAIL env var is unset.
// Per project decision (May 2026), audit alerts go to sithcare@gmail.com.
const DefaultRecipient = "sithcare@gmail.com"

func recipient() string {
	if v := strings.TrimSpace(os.Getenv("AUDIT_ALERT_EMAIL")); v != "" {
		return v
	}
	return DefaultRecipient
}

var ist *time.Location

func init() {
	var err error
	ist, err = time.LoadLocation("Asia/Kolkata")
	if err != nil {
		ist = time.UTC
	}
}

// ── Real-time critical alerts ────────────────────────────────────────────────

// OnCriticalAnomaly is the callback to plug into anomaly.SetNotifier.
// Fires immediately when a critical anomaly is detected.
func OnCriticalAnomaly(a anomaly.Anomaly) {
	if a.Severity != anomaly.SeverityCritical {
		return
	}
	html := buildCriticalEmailHTML(a)
	subject := fmt.Sprintf("🔴 CRITICAL audit anomaly — %s", a.RuleName)
	if err := email.SendSimple(recipient(), subject, html); err != nil {
		log.Printf("[alerts] critical email send failed: %v (anomaly_id=%d rule=%s)", err, a.ID, a.RuleName)
	} else {
		log.Printf("[alerts] critical email sent → %s (rule=%s, id=%d)", recipient(), a.RuleName, a.ID)
	}
}

func buildCriticalEmailHTML(a anomaly.Anomaly) string {
	return email.AuditCriticalAnomalyEmail(
		a.Title,
		a.Description,
		a.RuleName,
		a.ActorRole,
		a.Table,
		a.DetectedAt.In(ist).Format("02 Jan 2006, 15:04:05"),
		a.ActorID,
	)
}

// ── Daily digest ─────────────────────────────────────────────────────────────

// RunDailyDigest blocks forever, firing the digest email every day at 08:00 IST.
// Call once from main: go alerts.RunDailyDigest().
func RunDailyDigest() {
	for {
		next := nextDailyAt(8, 0)
		log.Printf("[alerts] next daily digest scheduled at %s", next.In(ist).Format(time.RFC1123))
		time.Sleep(time.Until(next))
		sendDigest()
	}
}

func nextDailyAt(hour, minute int) time.Time {
	now := time.Now().In(ist)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, ist)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// Package-level types so they can be passed between sendDigest and the
// builder helper.
type volRow struct {
	Table string `gorm:"column:table_name"`
	N     int64
}
type sevRow struct {
	Severity string
	N        int64
}
type srcRow struct {
	Source string
	N      int64
}

// autoActorSQL matches rows written by background schedulers under a system
// context (audit.ForSystem sets actor_id=0 and actor_role="system:<job>").
// Used to separate automated churn from real staff/user actions in the digest.
const autoActorSQL = "(actor_id = 0 OR actor_role LIKE 'system:%')"

// claimDailyDigest returns true if this replica should send today's digest.
// A per-day Redis lock makes the digest a true singleton across horizontally
// scaled replicas (which may each run the scheduler under ROLE=all), preventing
// the duplicate emails that arise when more than one instance fires it. Falls
// back to true when Redis is unavailable so the digest is never silently lost in
// a single-instance / dev setup.
func claimDailyDigest() bool {
	if database.Rdb == nil {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	day := time.Now().In(ist).Format("2006-01-02")
	key := "lock:audit:daily_digest:" + day
	// 23h TTL: long enough that no second replica can claim the same day, short
	// enough that the key clears before the next 08:00 IST run.
	ok, err := database.Rdb.SetNX(ctx, key, "sent", 23*time.Hour).Result()
	if err != nil {
		log.Printf("[alerts] digest lock check failed (%v) — proceeding to avoid missing the digest", err)
		return true
	}
	if !ok {
		log.Printf("[alerts] daily digest for %s already claimed by another replica — skipping", day)
	}
	return ok
}

func sendDigest() {
	// Singleton guard — see claimDailyDigest. Prevents duplicate digest emails
	// when multiple replicas run the scheduler.
	if !claimDailyDigest() {
		return
	}

	ctx := audit.ForSystem(context.Background(), "alerts:daily_digest")
	db := database.DB.WithContext(ctx)

	// Yesterday's window in IST
	now := time.Now().In(ist)
	dayStart := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, ist)
	dayEnd := dayStart.Add(24 * time.Hour)

	// Total audit events, split by source. Automated rows are written by the
	// background schedulers (anomaly scanner, SLA monitor, batch optimiser,
	// dispatch worker, cron) under a system context. Lumping them in with staff
	// actions makes the headline look wildly inflated — the reported "wrong
	// data" — so we separate manual (human) from automated (system) activity.
	var srcCounts []srcRow
	db.Table("audit_logs").
		Select("CASE WHEN "+autoActorSQL+" THEN 'automated' ELSE 'manual' END AS source, COUNT(*) AS n").
		Where("created_at >= ? AND created_at < ?", dayStart.UTC(), dayEnd.UTC()).
		Group("source").
		Scan(&srcCounts)
	var manualEvents, autoEvents int64
	for _, s := range srcCounts {
		switch s.Source {
		case "automated":
			autoEvents = s.N
		default:
			manualEvents = s.N
		}
	}
	totalEvents := manualEvents + autoEvents

	// Volume by table — manual (staff/user) activity, top 10. This is what a
	// human reviewer actually cares about.
	var volumes []volRow
	db.Table("audit_logs").
		Select("table_name, COUNT(*) AS n").
		Where("created_at >= ? AND created_at < ?", dayStart.UTC(), dayEnd.UTC()).
		Where("NOT " + autoActorSQL).
		Group("table_name").
		Order("n DESC").
		Limit(10).
		Scan(&volumes)

	// Volume by table — automated activity, top 5, shown separately for context.
	var autoVolumes []volRow
	db.Table("audit_logs").
		Select("table_name, COUNT(*) AS n").
		Where("created_at >= ? AND created_at < ?", dayStart.UTC(), dayEnd.UTC()).
		Where(autoActorSQL).
		Group("table_name").
		Order("n DESC").
		Limit(5).
		Scan(&autoVolumes)

	// Anomalies by severity
	var sevCounts []sevRow
	db.Table("audit_anomalies").
		Select("severity, COUNT(*) AS n").
		Where("detected_at >= ? AND detected_at < ?", dayStart.UTC(), dayEnd.UTC()).
		Group("severity").
		Scan(&sevCounts)

	// Recent unresolved anomalies — top 10 by severity then recency. Bounded to
	// the same yesterday window as the severity counts above, so the two sections
	// always describe the same day (previously this had no upper bound and pulled
	// today's + future-dated anomalies, disagreeing with the counts).
	var recent []anomaly.Anomaly
	db.Table("audit_anomalies").
		Where("status = ? AND detected_at >= ? AND detected_at < ?", anomaly.StatusNew, dayStart.UTC(), dayEnd.UTC()).
		Order("CASE severity WHEN 'critical' THEN 1 WHEN 'warning' THEN 2 ELSE 3 END, detected_at DESC").
		Limit(10).
		Find(&recent)

	// Chain verification status
	_, brokenAt, _ := audit.VerifyChain(db)
	chainStatus := "INTACT"
	if brokenAt > 0 {
		chainStatus = fmt.Sprintf("BROKEN at row %d", brokenAt)
	}

	html := buildDigestHTML(dayStart, manualEvents, autoEvents, totalEvents, chainStatus, volumes, autoVolumes, sevCounts, recent)
	subject := fmt.Sprintf("📊 MediHub Daily Audit Digest — %s", dayStart.Format("02 Jan 2006"))
	if err := email.SendSimple(recipient(), subject, html); err != nil {
		log.Printf("[alerts] daily digest send failed: %v", err)
	} else {
		log.Printf("[alerts] daily digest sent → %s (manual=%d, automated=%d, anomalies=%d)", recipient(), manualEvents, autoEvents, len(recent))
	}
}

func buildDigestHTML(day time.Time, manualEvents, autoEvents, totalEvents int64, chainStatus string,
	volumes, autoVolumes []volRow, sevCounts []sevRow, recent []anomaly.Anomaly,
) string {
	// Activity by source — placed first (rendered in the severity slot, which the
	// template prints right under the headline) so the manual/automated split is
	// the first thing the reader sees. The big headline number is the grand
	// total; this breakdown explains what it's made of.
	var sourceSummary strings.Builder
	sourceSummary.WriteString(`<h2 style="font-size:15px;color:#111;margin:24px 0 12px;border-bottom:1px solid #e5e7eb;padding-bottom:6px;">Activity by Source</h2>`)
	sourceSummary.WriteString(`<table style="width:100%;border-collapse:collapse;font-size:13px;">`)
	fmt.Fprintf(&sourceSummary, `<tr><td style="padding:6px 8px;color:#111;">👤 Manual — staff &amp; users</td><td style="text-align:right;padding:6px 8px;"><strong style="color:#2563eb;">%d</strong></td></tr>`, manualEvents)
	fmt.Fprintf(&sourceSummary, `<tr><td style="padding:6px 8px;color:#666;">🤖 Automated — system jobs</td><td style="text-align:right;padding:6px 8px;color:#888;">%d</td></tr>`, autoEvents)
	sourceSummary.WriteString(`</table>`)

	var severityBreakdown strings.Builder
	if len(sevCounts) > 0 {
		severityBreakdown.WriteString(`<h2 style="font-size:15px;color:#111;margin:24px 0 12px;border-bottom:1px solid #e5e7eb;padding-bottom:6px;">Anomalies Detected</h2>`)
		severityBreakdown.WriteString(`<table style="width:100%;border-collapse:collapse;font-size:13px;">`)
		for _, s := range sevCounts {
			color := "#666"
			emoji := "🟢"
			switch s.Severity {
			case "critical":
				color, emoji = "#dc2626", "🔴"
			case "warning":
				color, emoji = "#d97706", "🟡"
			case "info":
				color, emoji = "#2563eb", "🔵"
			}
			fmt.Fprintf(&severityBreakdown, `<tr><td style="padding:4px 8px;color:%s;">%s %s</td><td style="text-align:right;padding:4px 8px;"><strong>%d</strong></td></tr>`, color, emoji, strings.ToUpper(s.Severity), s.N)
		}
		severityBreakdown.WriteString(`</table>`)
	}

	var anomaliesList strings.Builder
	if len(recent) > 0 {
		anomaliesList.WriteString(`<h2 style="font-size:15px;color:#111;margin:24px 0 12px;border-bottom:1px solid #e5e7eb;padding-bottom:6px;">Recent Unresolved</h2>`)
		for _, a := range recent {
			color := "#666"
			switch a.Severity {
			case "critical":
				color = "#dc2626"
			case "warning":
				color = "#d97706"
			}
			fmt.Fprintf(&anomaliesList, `<div style="border-left:3px solid %s;padding:8px 12px;margin-bottom:8px;background:#fafafa;">`, color)
			fmt.Fprintf(&anomaliesList, `<div style="font-weight:600;font-size:13px;color:#111;">%s</div>`, a.Title)
			fmt.Fprintf(&anomaliesList, `<div style="font-size:11px;color:#666;margin-top:2px;">Actor #%d (%s) · %s IST</div>`, a.ActorID, a.ActorRole, a.DetectedAt.In(ist).Format("15:04:05"))
			anomaliesList.WriteString(`</div>`)
		}
	} else {
		anomaliesList.WriteString(`<p style="color:#16a34a;font-size:13px;background:#f0fdf4;padding:12px;border-radius:6px;margin:24px 0;">✓ No unresolved anomalies — system is clean.</p>`)
	}

	var volumesTable strings.Builder
	writeVolTable := func(heading string, rows []volRow, muted bool) {
		if len(rows) == 0 {
			return
		}
		valColor := ""
		if muted {
			valColor = "color:#888;"
		}
		fmt.Fprintf(&volumesTable, `<h2 style="font-size:15px;color:#111;margin:24px 0 12px;border-bottom:1px solid #e5e7eb;padding-bottom:6px;">%s</h2>`, heading)
		volumesTable.WriteString(`<table style="width:100%;border-collapse:collapse;font-size:13px;">`)
		for _, v := range rows {
			fmt.Fprintf(&volumesTable, `<tr><td style="padding:4px 8px;color:#444;"><code>%s</code></td><td style="text-align:right;padding:4px 8px;%s">%d events</td></tr>`, v.Table, valColor, v.N)
		}
		volumesTable.WriteString(`</table>`)
	}
	writeVolTable("Top Tables — Manual Activity", volumes, false)
	writeVolTable("Top Tables — Automated Activity", autoVolumes, true)

	chainColor := "#16a34a"
	if !strings.HasPrefix(chainStatus, "INTACT") {
		chainColor = "#dc2626"
	}

	return email.AuditDailyDigestEmail(
		day.Format("Monday, 02 January 2006"),
		fmt.Sprintf("%d", totalEvents),
		chainStatus,
		chainColor,
		volumesTable.String(),
		sourceSummary.String()+severityBreakdown.String(),
		anomaliesList.String(),
	)
}
