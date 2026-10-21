package middleware

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// auditCtxKey is the private context key for audit metadata.
// Use middleware.GetAuditContext(ctx) to read it.
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
// Used by background workers / cron jobs that don't have a Fiber request.
func WithAuditContext(ctx context.Context, ac *AuditContext) context.Context {
	return context.WithValue(ctx, auditCtxKey{}, ac)
}

// AuditCtx is a Fiber middleware that builds an AuditContext from the current
// request and stuffs it into the Go context. It is safe to mount globally
// (before per-route Protected()) because it calls parseJWT itself — the same
// idempotent helper used by Protected() — so actor_id is always populated for
// authenticated requests regardless of middleware ordering.
//
// Every db.WithContext(c.UserContext()) call downstream will then carry the
// audit metadata automatically — the GORM audit plugin reads it from there.
func AuditCtx() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Parse the JWT now so actor_id/role are available even when AuditCtx
		// runs before per-route Protected() in the middleware chain.
		parseJWT(c)

		actorID, _ := c.Locals("user_id").(uint)
		role, _ := c.Locals("role").(string)

		ac := &AuditContext{
			ActorID:   actorID,
			ActorRole: role,
			IP:        c.IP(),
			UserAgent: c.Get("User-Agent"),
			RequestID: uuid.NewString(),
			Path:      c.Path(),
			Method:    c.Method(),
		}

		// Push into Go context so downstream db.WithContext() calls carry it.
		goCtx := WithAuditContext(c.UserContext(), ac)
		c.SetUserContext(goCtx)

		// Echo request ID back so the frontend / Sentry / nginx can correlate.
		c.Set("X-Request-ID", ac.RequestID)
		return c.Next()
	}
}
