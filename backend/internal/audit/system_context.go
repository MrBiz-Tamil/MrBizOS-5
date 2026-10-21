package audit

import (
	"context"

	"drayya-core/internal/middleware"

	"github.com/google/uuid"
)

// ForSystem returns a Go context with an AuditContext attached that identifies
// the calling code as a NAMED system job, not an HTTP request.
//
// Usage from any background goroutine, cron handler, or migration:
//
//	ctx := audit.ForSystem(context.Background(), "nightly:doctor_score_recalc")
//	database.DB.WithContext(ctx).Save(...)
//
// The resulting audit rows show:
//
//	actor_id   = 0
//	actor_role = "system:nightly:doctor_score_recalc"
//	request_id = "sys-<uuid>"          (unique per ForSystem call)
//	path       = "/internal/nightly:doctor_score_recalc"
//
// Investigators can immediately distinguish a cron-driven update from a
// human-initiated one, and the request_id ties every write within one job
// run together for easy correlation.
//
// jobName convention: "category:specific_action" — e.g.
//
//	"cron:reminders"
//	"cron:instant_timeout"
//	"cron:weekly_payout"
//	"cron:doctor_heartbeat"
//	"cron:nightly_score_recalc"
//	"migration:add_paise_columns"
//	"webhook:razorpay_payment"
func ForSystem(ctx context.Context, jobName string) context.Context {
	if jobName == "" {
		jobName = "unknown"
	}
	return middleware.WithAuditContext(ctx, &middleware.AuditContext{
		ActorID:   0,
		ActorRole: "system:" + jobName,
		RequestID: "sys-" + uuid.NewString(),
		Path:      "/internal/" + jobName,
		Method:    "JOB",
	})
}
