package grpcserver

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"drayya-core/internal/authctx"
)

// Authorization (T2) — procedure-level permission enforcement.
//
// The auth interceptor (interceptor.go) proves WHO the caller is and WHICH
// tenant/BU they may act in. This interceptor enforces WHAT they may call: a
// deny-by-policy layer keyed on the RPC procedure.
//
// This is the extensible mechanism, seeded with the procedures whose privilege
// is unambiguous today (platform-level tenant administration). As each domain
// service is made real (replacing the Phase-0 mocks), its procedures get their
// own `<bu>.<resource>.<action>` requirements added to the policy here (or a
// per-service policy registered at mount time). Procedures with no explicit rule
// require only a valid authenticated session — which the auth interceptor has
// already enforced — so this layer never weakens existing authentication.

// superAdminOnlyProcedures lists procedures that require the platform super-admin
// role. These administer the platform itself (tenant lifecycle), not tenant data.
var superAdminOnlyProcedures = map[string]bool{
	"/iam.v1.IAMService/CreateTenant": true,
	"/iam.v1.IAMService/ListTenants":  true,
}

// requiresSuperAdmin reports whether the procedure is gated to platform
// super-admins. Exposed as a package function so it is unit-testable.
func requiresSuperAdmin(procedure string) bool {
	return superAdminOnlyProcedures[procedure]
}

// NewAuthzInterceptor creates the procedure-level authorization interceptor. It
// must be chained AFTER the auth interceptor so the verified role is present in
// the context.
func NewAuthzInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			proc := req.Spec().Procedure
			if publicProcedures[proc] {
				return next(ctx, req)
			}
			if requiresSuperAdmin(proc) {
				if !isPlatformSuperAdmin(authctx.Role(ctx)) {
					return nil, connect.NewError(connect.CodePermissionDenied,
						errors.New("this operation requires platform super-admin privileges"))
				}
			}
			return next(ctx, req)
		}
	}
}
