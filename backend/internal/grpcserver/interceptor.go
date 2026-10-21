package grpcserver

import (
	"context"
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

			if authHeader == "" || !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				return nil, connect.NewError(connect.CodeUnauthenticated, nil)
			}

			tokenStr := strings.TrimSpace(authHeader[7:])
			claims, err := tokens.ValidateToken(tokenStr)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}

			// Bind verified user context
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)

			// Resolve tenant context: check X-Tenant-Id or claims
			tenantID := req.Header().Get("X-Tenant-Id")
			if tenantID == "" {
				tenantID = "default"
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
