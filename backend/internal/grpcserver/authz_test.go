package grpcserver

import "testing"

func TestRoleAllowed(t *testing.T) {
	payroll, ok := rolesForProcedure("/hrms.v1.HRMSService/RunMonthlyPayroll")
	if !ok {
		t.Fatal("RunMonthlyPayroll must have an explicit policy")
	}
	cases := []struct {
		role string
		want bool
	}{
		{"hr_manager", true},
		{"admin", true},
		{"superadmin", true}, // platform super-admin always allowed
		{"sales", false},
		{"finance_manager", false}, // not in the payroll allow-list
		{"", false},
	}
	for _, c := range cases {
		if got := roleAllowed(c.role, payroll); got != c.want {
			t.Errorf("roleAllowed(%q, payroll) = %v, want %v", c.role, got, c.want)
		}
	}
}

func TestFinanceProcedurePolicy(t *testing.T) {
	pay, ok := rolesForProcedure("/billing.v1.BillingService/RecordPayment")
	if !ok {
		t.Fatal("RecordPayment must have a policy")
	}
	if !roleAllowed("finance_manager", pay) || !roleAllowed("accountant", pay) {
		t.Error("finance roles must be allowed to record payments")
	}
	if roleAllowed("patient", pay) {
		t.Error("a patient must not be allowed to record payments")
	}
}

func TestUnlistedProcedureIsAuthenticatedOnly(t *testing.T) {
	// A procedure with no explicit rule is not gated here (authentication only).
	if _, ok := rolesForProcedure("/billing.v1.BillingService/ListInvoices"); ok {
		t.Error("ListInvoices should have no explicit role rule (authenticated-only)")
	}
}

func TestSuperAdminOnlyProcedures(t *testing.T) {
	if !requiresSuperAdmin("/iam.v1.IAMService/CreateTenant") {
		t.Error("CreateTenant must be super-admin only")
	}
	if requiresSuperAdmin("/hrms.v1.HRMSService/RunMonthlyPayroll") {
		t.Error("RunMonthlyPayroll is role-gated, not super-admin-only")
	}
	// super-admin may still call a super-admin-only procedure
	if allowed, _ := rolesForProcedure("/iam.v1.IAMService/CreateTenant"); !roleAllowed("superadmin", allowed) {
		t.Error("super-admin must be allowed on super-admin-only procedures")
	}
	if allowed, _ := rolesForProcedure("/iam.v1.IAMService/CreateTenant"); roleAllowed("admin", allowed) {
		t.Error("tenant admin must NOT pass a super-admin-only procedure")
	}
}
