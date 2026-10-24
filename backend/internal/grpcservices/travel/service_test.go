package travel

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	travelv1 "drayya-core/gen/travel/v1"
	"drayya-core/internal/authctx"
	traveldomain "drayya-core/internal/domain/travel"

	"github.com/glebarez/sqlite"
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
	if err := traveldomain.AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Exec("DELETE FROM tour_packages")
	db.Exec("DELETE FROM visa_applications")
	return db
}

func ctxTenant(tenant string) context.Context {
	if tenant == "" {
		return context.Background()
	}
	return authctx.WithTenantID(context.Background(), tenant)
}

// T3 regression: tour packages are tenant-scoped — a caller only sees its own
// org's packages, an unauthenticated context sees none, and a package id from
// another tenant is not readable.
func TestTourPackageTenantIsolation(t *testing.T) {
	db := newTestDB(t)
	p1 := traveldomain.TourPackage{OrganizationID: 1, BusinessUnitID: 1, Title: "ACME Tour", Slug: "acme-1", Days: 3, Nights: 2, Price: 1000, DestinationID: 1, IsActive: true}
	p2 := traveldomain.TourPackage{OrganizationID: 2, BusinessUnitID: 1, Title: "Globex Tour", Slug: "globex-1", Days: 4, Nights: 3, Price: 2000, DestinationID: 1, IsActive: true}
	if err := db.Create(&p1).Error; err != nil {
		t.Fatalf("seed p1: %v", err)
	}
	if err := db.Create(&p2).Error; err != nil {
		t.Fatalf("seed p2: %v", err)
	}
	srv := NewServer(db)

	list := func(tenant string) []*travelv1.TourPackageProto {
		res, err := srv.ListTourPackages(ctxTenant(tenant), connect.NewRequest(&travelv1.ListTourPackagesRequest{}))
		if err != nil {
			t.Fatalf("ListTourPackages(%q): %v", tenant, err)
		}
		return res.Msg.Packages
	}

	if got := list("tenant-1"); len(got) != 1 || got[0].Title != "ACME Tour" {
		t.Fatalf("tenant-1 = %+v, want only ACME Tour", got)
	}
	if got := list("tenant-2"); len(got) != 1 || got[0].Title != "Globex Tour" {
		t.Fatalf("tenant-2 = %+v, want only Globex Tour", got)
	}
	if got := list(""); len(got) != 0 {
		t.Fatalf("no-tenant = %+v, want empty", got)
	}

	// Cross-tenant read of another org's package id must be not-found.
	_, err := srv.GetTourPackage(ctxTenant("tenant-1"), connect.NewRequest(&travelv1.GetTourPackageRequest{Id: fmt.Sprintf("%d", p2.ID)}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("cross-tenant GetTourPackage code = %v, want NotFound", connect.CodeOf(err))
	}
}
