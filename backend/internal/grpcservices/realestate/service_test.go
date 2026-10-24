package realestate

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	realestatev1 "drayya-core/gen/realestate/v1"
	"drayya-core/internal/authctx"
	redomain "drayya-core/internal/domain/realestate"

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
	if err := db.AutoMigrate(&redomain.Property{}, &redomain.SiteVisit{}, &redomain.PropertyDeal{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Exec("DELETE FROM properties")
	return db
}

func ctxTenant(tenant string) context.Context {
	if tenant == "" {
		return context.Background()
	}
	return authctx.WithTenantID(context.Background(), tenant)
}

// T3 regression: properties are tenant-scoped — a caller only sees its own org's
// properties, an unauthenticated context sees none, and a property id from
// another tenant is not readable.
func TestPropertyTenantIsolation(t *testing.T) {
	db := newTestDB(t)
	p1 := redomain.Property{OrganizationID: 1, BusinessUnitID: 3, Title: "ACME Villa", PropertyType: "APARTMENT", TransactionType: "SALE", Price: 1000, City: "Hyderabad", Location: "A", Status: "AVAILABLE"}
	p2 := redomain.Property{OrganizationID: 2, BusinessUnitID: 3, Title: "Globex Tower", PropertyType: "COMMERCIAL", TransactionType: "RENT", Price: 2000, City: "Hyderabad", Location: "B", Status: "AVAILABLE"}
	if err := db.Create(&p1).Error; err != nil {
		t.Fatalf("seed p1: %v", err)
	}
	if err := db.Create(&p2).Error; err != nil {
		t.Fatalf("seed p2: %v", err)
	}
	srv := NewServer(db)

	list := func(tenant string) []*realestatev1.PropertyProto {
		res, err := srv.ListProperties(ctxTenant(tenant), connect.NewRequest(&realestatev1.ListPropertiesRequest{}))
		if err != nil {
			t.Fatalf("ListProperties(%q): %v", tenant, err)
		}
		return res.Msg.Properties
	}

	if got := list("tenant-1"); len(got) != 1 || got[0].Title != "ACME Villa" {
		t.Fatalf("tenant-1 = %+v, want only ACME Villa", got)
	}
	if got := list("tenant-2"); len(got) != 1 || got[0].Title != "Globex Tower" {
		t.Fatalf("tenant-2 = %+v, want only Globex Tower", got)
	}
	if got := list(""); len(got) != 0 {
		t.Fatalf("no-tenant = %+v, want empty", got)
	}

	// Cross-tenant read of another org's property id must be not-found.
	_, err := srv.GetProperty(ctxTenant("tenant-1"), connect.NewRequest(&realestatev1.GetPropertyRequest{Id: fmt.Sprintf("%d", p2.ID)}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("cross-tenant GetProperty code = %v, want NotFound", connect.CodeOf(err))
	}
}
