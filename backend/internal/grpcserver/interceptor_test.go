package grpcserver

import "testing"

// T2 regression: the tenant-resolution decision must reject a forged X-Tenant-Id
// for a non-platform role, and only let a platform super-admin cross tenants.
func TestResolveTenant(t *testing.T) {
	cases := []struct {
		name         string
		role         string
		claimTenant  string
		headerTenant string
		devAuth      bool
		wantTenant   string
		wantCross    bool
		wantErr      bool
	}{
		{"forged header denied for tenant admin", "admin", "tenant-1", "tenant-2", false, "", false, true},
		{"forged header denied for sales", "sales", "tenant-1", "tenant-2", false, "", false, true},
		{"matching header ok", "sales", "tenant-1", "tenant-1", false, "tenant-1", false, false},
		{"empty header uses claim", "sales", "tenant-1", "", false, "tenant-1", false, false},
		{"superadmin may cross (audited)", "superadmin", "tenant-1", "tenant-2", false, "tenant-2", true, false},
		{"missing tenant rejected", "sales", "", "", false, "", false, true},
		{"missing tenant dev fallback", "sales", "", "", true, "tenant-1", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tenant, cross, err := resolveTenant(c.role, c.claimTenant, c.headerTenant, c.devAuth)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if err != nil {
				return
			}
			if tenant != c.wantTenant || cross != c.wantCross {
				t.Fatalf("got (%q, cross=%v), want (%q, cross=%v)", tenant, cross, c.wantTenant, c.wantCross)
			}
		})
	}
}

func TestAuthorizeBusinessUnit(t *testing.T) {
	cases := []struct {
		name      string
		role      string
		headerBU  string
		currentBU string
		allowed   []string
		wantBU    string
		wantErr   bool
	}{
		{"bu outside scope denied", "sales", "travel", "", []string{"healthcare"}, "", true},
		{"bu in scope ok", "sales", "healthcare", "", []string{"healthcare"}, "healthcare", false},
		{"tenant admin may use any bu", "admin", "travel", "", []string{"healthcare"}, "travel", false},
		{"empty header uses current bu", "sales", "", "healthcare", []string{"healthcare"}, "healthcare", false},
		{"no membership recorded passes through", "sales", "travel", "", nil, "travel", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bu, err := authorizeBusinessUnit(c.role, c.headerBU, c.currentBU, c.allowed)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if err == nil && bu != c.wantBU {
				t.Fatalf("got %q, want %q", bu, c.wantBU)
			}
		})
	}
}

func TestRequiresSuperAdmin(t *testing.T) {
	if !requiresSuperAdmin("/iam.v1.IAMService/CreateTenant") {
		t.Fatal("CreateTenant must require super-admin")
	}
	if !requiresSuperAdmin("/iam.v1.IAMService/ListTenants") {
		t.Fatal("ListTenants must require super-admin")
	}
	if requiresSuperAdmin("/billing.v1.BillingService/ListInvoices") {
		t.Fatal("ordinary procedures must not require super-admin")
	}
}

func TestIsPlatformSuperAdmin(t *testing.T) {
	if isPlatformSuperAdmin("admin") {
		t.Fatal("tenant admin must NOT be a platform super-admin")
	}
	if !isPlatformSuperAdmin("superadmin") {
		t.Fatal("superadmin must be a platform super-admin")
	}
}
