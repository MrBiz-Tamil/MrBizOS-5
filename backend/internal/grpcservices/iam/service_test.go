package iam

import (
	"context"
	"os"
	"testing"

	"connectrpc.com/connect"
	iamv1 "drayya-core/gen/iam/v1"
	"drayya-core/internal/domain/identity"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(&identity.User{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	// Clean slate (shared in-memory db persists across opens within the process).
	db.Exec("DELETE FROM user")
	return db
}

func seedUser(t *testing.T, db *gorm.DB, email, password, role string, org, bu uint, active bool) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	u := identity.User{
		Email:          email,
		PasswordHash:   string(hash),
		Role:           role,
		OrganizationID: org,
		BusinessUnitID: bu,
		IsActive:       active,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	// GORM applies the `default:true` on IsActive for the false zero-value on
	// insert, so force the intended value explicitly after creation.
	if !active {
		if err := db.Model(&identity.User{}).Where("id = ?", u.ID).Update("is_active", false).Error; err != nil {
			t.Fatalf("set inactive: %v", err)
		}
	}
}

// T1 regression: Login must require a real, persisted credential and must never
// mint a super-admin session for an arbitrary or unknown email.
func TestLogin(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests-only-32b")
	db := newTestDB(t)
	seedUser(t, db, "ops@acme.test", "correct-horse", "manager", 1, 1 /*travel*/, true)
	seedUser(t, db, "frozen@acme.test", "correct-horse", "manager", 1, 1, false)
	srv := NewServer(db)
	ctx := context.Background()

	t.Run("valid credentials", func(t *testing.T) {
		res, err := srv.Login(ctx, connect.NewRequest(&iamv1.LoginRequest{Email: "ops@acme.test", Password: "correct-horse"}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		u := res.Msg.User
		if u.IsSuperAdmin {
			t.Error("a manager must NOT be returned as super-admin")
		}
		if u.TenantId != "tenant-1" {
			t.Errorf("tenant = %q, want tenant-1", u.TenantId)
		}
		if u.CurrentBusinessUnitId != "travel" {
			t.Errorf("current BU = %q, want travel", u.CurrentBusinessUnitId)
		}
		if len(u.AllowedBusinessUnits) != 1 || u.AllowedBusinessUnits[0] != "travel" {
			t.Errorf("allowed BUs = %v, want [travel]", u.AllowedBusinessUnits)
		}
		if res.Msg.Token == "" {
			t.Error("expected a signed token")
		}
	})

	t.Run("wrong password rejected", func(t *testing.T) {
		_, err := srv.Login(ctx, connect.NewRequest(&iamv1.LoginRequest{Email: "ops@acme.test", Password: "wrong"}))
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", connect.CodeOf(err))
		}
	})

	t.Run("unknown email yields no session (no super-admin fallback)", func(t *testing.T) {
		_, err := srv.Login(ctx, connect.NewRequest(&iamv1.LoginRequest{Email: "admin@mrbiz.local", Password: "anything"}))
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("code = %v, want Unauthenticated", connect.CodeOf(err))
		}
	})

	t.Run("empty password rejected", func(t *testing.T) {
		_, err := srv.Login(ctx, connect.NewRequest(&iamv1.LoginRequest{Email: "ops@acme.test", Password: ""}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("code = %v, want InvalidArgument", connect.CodeOf(err))
		}
	})

	t.Run("inactive account rejected", func(t *testing.T) {
		_, err := srv.Login(ctx, connect.NewRequest(&iamv1.LoginRequest{Email: "frozen@acme.test", Password: "correct-horse"}))
		if connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("code = %v, want PermissionDenied", connect.CodeOf(err))
		}
	})
}

func TestScopeMappings(t *testing.T) {
	if isPlatformSuperAdmin("admin") {
		t.Error("tenant admin is not a platform super-admin")
	}
	if !isPlatformSuperAdmin("superadmin") {
		t.Error("superadmin is a platform super-admin")
	}
	if got := allowedBUsForRole("manager", 1); len(got) != 1 || got[0] != "travel" {
		t.Errorf("non-admin allowed BUs = %v, want [travel]", got)
	}
	if got := allowedBUsForRole("superadmin", 2); len(got) != len(allBUSlugs) {
		t.Errorf("super-admin allowed BUs = %v, want all", got)
	}
	if tenantStringForOrg(3) != "tenant-3" {
		t.Error("org 3 -> tenant-3")
	}
	if orgIDFromTenant("tenant-7") != 7 {
		t.Error("tenant-7 -> org 7")
	}
}
