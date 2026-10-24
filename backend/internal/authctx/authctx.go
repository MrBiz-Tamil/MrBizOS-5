// Package authctx carries the authenticated caller's identity and authorized
// scope (user id, role, tenant, business unit) through the request context.
//
// It exists as its own package so that both the transport layer (the gRPC auth
// interceptor in internal/grpcserver) and the domain services (internal/grpcservices/*)
// can share one canonical, typed set of context keys without an import cycle —
// grpcserver imports the services, so the services cannot import grpcserver.
//
// The values here are set ONLY by the trusted auth interceptor after it has
// validated a signed session and verified tenant/BU authorization. Services read
// them to scope their work; they must never be derived from client-supplied
// request fields or headers.
package authctx

import "context"

type ctxKey string

const (
	userIDKey ctxKey = "auth.user_id"
	roleKey   ctxKey = "auth.role"
	tenantKey ctxKey = "auth.tenant_id"
	buKey     ctxKey = "auth.business_unit_id"
)

// WithUserID returns a context carrying the authenticated user id.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithRole returns a context carrying the authenticated user's role.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

// WithTenantID returns a context carrying the authorized tenant id.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// WithBusinessUnit returns a context carrying the active business unit id.
func WithBusinessUnit(ctx context.Context, buID string) context.Context {
	return context.WithValue(ctx, buKey, buID)
}

// UserID returns the authenticated user id, or "" when unauthenticated.
func UserID(ctx context.Context) string { return str(ctx, userIDKey) }

// Role returns the authenticated user's role, or "" when unauthenticated.
func Role(ctx context.Context) string { return str(ctx, roleKey) }

// TenantID returns the authorized tenant id, or "" when none is bound. A caller
// that persists or reads tenant-owned data MUST treat "" as unauthorized and
// refuse rather than fall back to a default tenant.
func TenantID(ctx context.Context) string { return str(ctx, tenantKey) }

// BusinessUnit returns the active business unit id, or "" when none is bound.
func BusinessUnit(ctx context.Context) string { return str(ctx, buKey) }

func str(ctx context.Context, k ctxKey) string {
	if v, ok := ctx.Value(k).(string); ok {
		return v
	}
	return ""
}
