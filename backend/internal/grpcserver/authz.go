package grpcserver

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"drayya-core/internal/authctx"
)

// Authorization (T2/T3) — procedure-level permission enforcement.
//
// The auth interceptor (interceptor.go) proves WHO the caller is and WHICH
// tenant/BU they may act in. This interceptor enforces WHAT they may call: a
// deny-by-policy layer keyed on the RPC procedure.
//
// Policy model: `procedureRoles` maps a procedure to the set of roles permitted
// to call it. A platform super-admin ("superadmin") is ALWAYS allowed. An empty
// role slice means "super-admin only" (platform administration). A procedure that
// is absent from the map requires only a valid authenticated session — which the
// auth interceptor has already enforced — so this layer never weakens existing
// authentication; it only adds restrictions.
//
// This is the extensible mechanism. High-impact financial / payroll / approval
// procedures are gated here regardless of whether the underlying service is real
// yet, so authorization is correct before those services are implemented. As each
// domain service is completed, its remaining mutating procedures get rules added.

var procedureRoles = map[string][]string{
	// Platform administration — super-admin only.
	"/iam.v1.IAMService/CreateTenant": {},
	"/iam.v1.IAMService/ListTenants":  {},

	// Finance / billing — money-moving or ledger-affecting operations.
	"/billing.v1.BillingService/RecordPayment": {"admin", "finance_manager", "accountant", "manager"},
	"/billing.v1.BillingService/CancelInvoice": {"admin", "finance_manager"},

	// Banking reconciliation.
	"/banking.v1.BankingService/RunAutoReconciliation":  {"admin", "finance_manager", "accountant"},
	"/banking.v1.BankingService/ManualMatchTransaction": {"admin", "finance_manager", "accountant"},

	// Payroll — HR-privileged.
	"/hrms.v1.HRMSService/RunMonthlyPayroll":    {"admin", "hr_manager"},
	"/hrms.v1.HRMSService/UpdateSalaryStructure": {"admin", "hr_manager"},

	// Approval queue — managers / finance only.
	"/workflow.v1.WorkflowService/ApproveAction": {"admin", "manager", "finance_manager"},
	"/workflow.v1.WorkflowService/RejectAction":  {"admin", "manager", "finance_manager"},

	// Real-estate deal creation and travel visa decisions — operational managers.
	"/realestate.v1.RealEstateService/CreatePropertyDeal": {"admin", "manager"},
	"/travel.v1.TravelService/UpdateVisaStatus":           {"admin", "manager"},
}

// rolesForProcedure returns the roles permitted for a procedure (besides
// super-admin) and whether an explicit rule exists.
func rolesForProcedure(procedure string) ([]string, bool) {
	r, ok := procedureRoles[procedure]
	return r, ok
}

// roleAllowed reports whether a role satisfies a procedure's allowed-role set. A
// platform super-admin is always allowed.
func roleAllowed(role string, allowed []string) bool {
	if isPlatformSuperAdmin(role) {
		return true
	}
	role = strings.TrimSpace(role)
	for _, a := range allowed {
		if strings.EqualFold(role, a) {
			return true
		}
	}
	return false
}

// requiresSuperAdmin reports whether the procedure is gated to platform
// super-admins (an explicit rule with an empty role set).
func requiresSuperAdmin(procedure string) bool {
	allowed, ok := procedureRoles[procedure]
	return ok && len(allowed) == 0
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
			if allowed, ok := rolesForProcedure(proc); ok {
				if !roleAllowed(authctx.Role(ctx), allowed) {
					return nil, connect.NewError(connect.CodePermissionDenied,
						errors.New("insufficient privileges for this operation"))
				}
			}
			return next(ctx, req)
		}
	}
}
