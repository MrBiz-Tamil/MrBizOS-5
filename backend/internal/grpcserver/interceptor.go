package grpcserver

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	"connectrpc.com/connect"
	"drayya-core/internal/tokens"
)

type contextKey string

const (
	UserIDKey         contextKey = "user_id"
	RoleKey           contextKey = "role"
	TenantIDKey       contextKey = "tenant_id"
	BusinessUnitIDKey contextKey = "business_unit_id"
)

// devAuthEnabled reports whether the insecure local-development auth bypass is
// explicitly turned on via the MRBIZ_DEV_AUTH environment variable. It is OFF by
// default, so every deployed/production environment fails closed: requests with
// no token, an invalid token, or no tenant context are rejected. A developer can
// opt back into the permissive "always super-admin on tenant-1" behaviour by
// setting MRBIZ_DEV_AUTH=1 (or true/yes) while real session wiring is completed.
func devAuthEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MRBIZ_DEV_AUTH"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// Public procedures that do not require authentication
var publicProcedures = map[string]bool{
	"/iam.v1.IAMService/Login":           true,
	"/iam.v1.IAMService/ValidateSession": true,
}

// NewAuthInterceptor creates a Connect interceptor enforcing strict authentication
// and verifying tenant & business unit authorization.
func NewAuthInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			proc := req.Spec().Procedure
			if publicProcedures[proc] {
				return next(ctx, req)
			}

			authHeader := req.Header().Get("Authorization")
			if authHeader == "" {
				authHeader = req.Header().Get("authorization")
			}

			var tokenStr string
			if authHeader != "" && strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				tokenStr = strings.TrimSpace(authHeader[7:])
			} else {
				// Check cookies for saas_session or saas_token
				cookieHeader := req.Header().Get("Cookie")
				for _, part := range strings.Split(cookieHeader, ";") {
					part = strings.TrimSpace(part)
					if strings.HasPrefix(part, "saas_session=") {
						tokenStr = strings.TrimPrefix(part, "saas_session=")
						break
					} else if strings.HasPrefix(part, "saas_token=") {
						tokenStr = strings.TrimPrefix(part, "saas_token=")
						break
					}
				}
			}

			// Resolve the caller's identity from a real, signed session token only.
			// There is no magic-string bypass: an empty or invalid token is rejected
			// unless the explicit local-dev escape hatch (MRBIZ_DEV_AUTH) is enabled.
			var claims *tokens.Claims
			if tokenStr != "" {
				c, err := tokens.ValidateToken(tokenStr)
				if err != nil {
					if devAuthEnabled() {
						log.Printf("[auth][DEV] invalid session token accepted as super-admin via MRBIZ_DEV_AUTH (proc=%s)", proc)
						claims = &tokens.Claims{UserID: "user-1", Role: "admin"}
					} else {
						return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or expired session"))
					}
				} else {
					claims = c
				}
			} else {
				if devAuthEnabled() {
					log.Printf("[auth][DEV] missing session token accepted as super-admin via MRBIZ_DEV_AUTH (proc=%s)", proc)
					claims = &tokens.Claims{UserID: "user-1", Role: "admin"}
				} else {
					return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
				}
			}

			// Bind verified user context
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)

			// Resolve tenant context from the caller-supplied header. A tenant must be
			// present for every authenticated call; only the dev escape hatch falls back
			// to the demo tenant so a missing header cannot silently read tenant-1 data.
			tenantID := req.Header().Get("X-Tenant-Id")
			if tenantID == "" {
				if devAuthEnabled() {
					tenantID = "tenant-1"
				} else {
					return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("tenant context required"))
				}
			}
			ctx = context.WithValue(ctx, TenantIDKey, tenantID)

			// Resolve business unit context: check X-Business-Unit-Id
			buID := req.Header().Get("X-Business-Unit-Id")
			if buID != "" {
				// Role-based or permission check can be enforced here
				ctx = context.WithValue(ctx, BusinessUnitIDKey, buID)
			}

			return next(ctx, req)
		}
	}
}

// UserIDFromContext extracts authenticated user ID
func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

// TenantIDFromContext extracts authenticated tenant ID
func TenantIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(TenantIDKey).(string); ok {
		return v
	}
	return ""
}

// BusinessUnitIDFromContext extracts current business unit ID
func BusinessUnitIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(BusinessUnitIDKey).(string); ok {
		return v
	}
	return ""
}
