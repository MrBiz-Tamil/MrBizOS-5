package grpcserver

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

	"connectrpc.com/connect"
	"drayya-core/internal/authctx"
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

// isPlatformSuperAdmin reports whether a role may cross tenant boundaries. Only
// the explicit platform role "superadmin" qualifies. A tenant-level "admin" is
// deliberately NOT a platform super-admin — this is the Phase 1 (T2) fix for the
// prior behaviour where any admin could read/write any tenant by setting the
// X-Tenant-Id header.
func isPlatformSuperAdmin(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), "superadmin")
}

// Public procedures that do not require authentication
var publicProcedures = map[string]bool{
	"/iam.v1.IAMService/Login":           true,
	"/iam.v1.IAMService/ValidateSession": true,
}

// resolveTenant decides the effective tenant for a request from the VERIFIED JWT
// claims and the client-supplied X-Tenant-Id header. It never trusts the header
// on its own. It returns the tenant to bind, whether this is a cross-tenant
// access by a platform super-admin (so the caller can audit it), or an error to
// reject with. It is a pure function to keep the authorization decision unit
// testable (see interceptor_test.go).
func resolveTenant(role, claimTenant, headerTenant string, devAuth bool) (tenant string, crossTenant bool, err error) {
	claimTenant = strings.TrimSpace(claimTenant)
	headerTenant = strings.TrimSpace(headerTenant)

	if claimTenant != "" {
		if headerTenant != "" && headerTenant != claimTenant {
			// The caller is asking to act in a tenant other than the one their
			// session is bound to. Only a platform super-admin may do so.
			if !isPlatformSuperAdmin(role) {
				return "", false, errors.New("cross-tenant access denied: requested tenant does not match authorized session")
			}
			return headerTenant, true, nil
		}
		return claimTenant, false, nil
	}

	// No tenant bound in the session.
	if headerTenant != "" && isPlatformSuperAdmin(role) {
		// A platform super-admin with no home tenant may target a specific tenant.
		return headerTenant, true, nil
	}
	if devAuth {
		return "tenant-1", false, nil
	}
	return "", false, errors.New("tenant context required")
}

// authorizeBusinessUnit decides the active business unit from the verified claims
// and the client-supplied X-Business-Unit-Id header. Within a tenant, a tenant
// admin or platform super-admin may act in any BU; every other user must have the
// requested BU in their authorized AllowedBusinessUnits. Pure function for tests.
func authorizeBusinessUnit(role, headerBU, currentBU string, allowed []string) (buID string, err error) {
	headerBU = strings.TrimSpace(headerBU)
	if headerBU == "" {
		return strings.TrimSpace(currentBU), nil
	}
	privileged := isPlatformSuperAdmin(role) || strings.EqualFold(strings.TrimSpace(role), "admin")
	if privileged || len(allowed) == 0 {
		// len(allowed)==0: no explicit membership recorded yet (pre-T4) — do not
		// hard-fail; the tenant check above already contains the blast radius.
		return headerBU, nil
	}
	for _, b := range allowed {
		if b == headerBU {
			return headerBU, nil
		}
	}
	return "", errors.New("access denied: requested business unit not in authorized user scope")
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
						claims = &tokens.Claims{UserID: "user-1", Role: "superadmin"}
					} else {
						return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or expired session"))
					}
				} else {
					claims = c
				}
			} else {
				if devAuthEnabled() {
					log.Printf("[auth][DEV] missing session token accepted as super-admin via MRBIZ_DEV_AUTH (proc=%s)", proc)
					claims = &tokens.Claims{UserID: "user-1", Role: "superadmin"}
				} else {
					return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
				}
			}

			// Bind verified user identity.
			ctx = authctx.WithUserID(ctx, claims.UserID)
			ctx = authctx.WithRole(ctx, claims.Role)

			// Resolve & authorize tenant context against the verified claims.
			tenantID, crossTenant, err := resolveTenant(claims.Role, claims.TenantID, req.Header().Get("X-Tenant-Id"), devAuthEnabled())
			if err != nil {
				if strings.Contains(err.Error(), "cross-tenant") {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}
			if crossTenant {
				// Privileged cross-tenant access by a platform super-admin is allowed
				// but always audited so it is never silent.
				log.Printf("[auth][AUDIT] platform super-admin user=%s accessed tenant=%s (proc=%s)", claims.UserID, tenantID, proc)
			}
			ctx = authctx.WithTenantID(ctx, tenantID)

			// Resolve & authorize business unit context.
			buID, err := authorizeBusinessUnit(claims.Role, req.Header().Get("X-Business-Unit-Id"), claims.CurrentBusinessUnit, claims.AllowedBusinessUnits)
			if err != nil {
				return nil, connect.NewError(connect.CodePermissionDenied, err)
			}
			if buID != "" {
				ctx = authctx.WithBusinessUnit(ctx, buID)
			}

			return next(ctx, req)
		}
	}
}

// UserIDFromContext extracts the authenticated user ID.
func UserIDFromContext(ctx context.Context) string { return authctx.UserID(ctx) }

// TenantIDFromContext extracts the authorized tenant ID.
func TenantIDFromContext(ctx context.Context) string { return authctx.TenantID(ctx) }

// BusinessUnitIDFromContext extracts the active business unit ID.
func BusinessUnitIDFromContext(ctx context.Context) string { return authctx.BusinessUnit(ctx) }
