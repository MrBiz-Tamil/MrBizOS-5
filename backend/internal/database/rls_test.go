package database

import (
	"context"
	"os"
	"testing"

	"drayya-core/internal/authctx"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func rlsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	return db
}

func TestWithTenantTxFailsClosed(t *testing.T) {
	db := rlsTestDB(t)
	// No tenant on the context -> must refuse rather than run unscoped.
	err := WithTenantTx(context.Background(), db, func(tx *gorm.DB) error { return nil })
	if err == nil {
		t.Fatal("expected error when no tenant is bound to context")
	}
}

func TestWithTenantTxRunsInsideTransaction(t *testing.T) {
	db := rlsTestDB(t)
	ctx := authctx.WithTenantID(context.Background(), "tenant-1")
	ran := false
	err := WithTenantTx(ctx, db, func(tx *gorm.DB) error {
		ran = true
		// The callback receives a usable transaction handle.
		return tx.Exec("SELECT 1").Error
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ran {
		t.Fatal("callback did not run")
	}
}

// TestWithTenantTxSetsPostgresGUC verifies the app.current_org GUC is actually
// set inside the transaction on PostgreSQL. It is skipped unless DATABASE_URL
// points at a reachable Postgres, so it runs in a DB-backed CI but not in this
// environment (ADR-5 verification is pending a Postgres instance).
func TestWithTenantTxSetsPostgresGUC(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set; skipping PostgreSQL RLS GUC verification")
	}
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	ctx := authctx.WithTenantID(context.Background(), "tenant-7")
	var got string
	err = WithTenantTx(ctx, db, func(tx *gorm.DB) error {
		return tx.Raw("SELECT current_setting('app.current_org', true)").Scan(&got).Error
	})
	if err != nil {
		t.Fatalf("WithTenantTx: %v", err)
	}
	if got != "7" {
		t.Fatalf("app.current_org = %q, want 7", got)
	}
}
