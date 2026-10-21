// Package middleware previously hosted the Fiber HTTP middleware layer, which
// was removed together with the REST server. It now retains only the
// request-scoped AuditContext plumbing: this metadata is carried on Go's
// context.Context and consumed by the GORM audit plugin, independent of any
// HTTP framework. Background workers, cron jobs and migrations attach it via
// WithAuditContext; the ConnectRPC interceptors do the same per request.
package middleware

import "context"

// auditCtxKey is the private context key for audit metadata.
// Use GetAuditContext(ctx) to read it.
type auditCtxKey struct{}

// AuditContext carries the actor + request metadata into the GORM session via
// Go's context.Context, so the audit plugin can pick it up automatically.
type AuditContext struct {
	ActorID   uint
	ActorRole string
	IP        string
	UserAgent string
	RequestID string
	Path      string
	Method    string
	// Bulk marks a batched operation — the audit plugin will emit a single
	// summary log instead of per-row events.
	Bulk bool
}

// GetAuditContext extracts the audit context from a Go context. Returns nil
// (not panic) when no context was set — caller should treat that as a system
// action (e.g., cron jobs, migrations).
func GetAuditContext(ctx context.Context) *AuditContext {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(auditCtxKey{}).(*AuditContext)
	return v
}

// WithAuditContext returns a new context with the given audit metadata attached.
// Used by background workers / cron jobs that don't have an HTTP request.
func WithAuditContext(ctx context.Context, ac *AuditContext) context.Context {
	return context.WithValue(ctx, auditCtxKey{}, ac)
}
