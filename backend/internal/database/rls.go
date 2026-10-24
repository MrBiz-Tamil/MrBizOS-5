package database

import (
	"context"
	"errors"
	"strconv"

	"drayya-core/internal/authctx"
	"gorm.io/gorm"
)

// Row-Level Security support (T3 / ADR-5) — per-request tenant GUC.
//
// Application-layer scoping (OrgScope) is the primary enforcement today. RLS is
// the complementary defense-in-depth layer: with the policies in
// migrations/027_drayya_rls_tenant_isolation.sql enabled, PostgreSQL itself
// refuses to return or modify rows outside the current organization, so a missed
// OrgScope in a handler can no longer leak data.
//
// RLS policies read the organization id from a transaction-local GUC,
// `app.current_org`. WithTenantTx sets that GUC (via set_config(..., true), which
// is scoped to the surrounding transaction) from the authenticated tenant on the
// context and runs the caller's work inside that transaction.
//
// IMPORTANT (status): the RLS policies are NOT auto-applied — drayya-core
// currently creates its schema with GORM AutoMigrate, not the SQL migration
// runner, so the 027 migration only takes effect once the SQL-migration mechanism
// is adopted (decision DEC-04). Until then this helper still provides a correct,
// transaction-scoped execution path; the SET is a no-op on non-PostgreSQL drivers
// (e.g. the SQLite used in dev/tests), where OrgScope remains the enforcement.
//
// This helper has NOT been verified against a live PostgreSQL instance in this
// environment (no database available); the Postgres assertion lives in
// rls_test.go and is skipped unless DATABASE_URL is set.

// WithTenantTx runs fn inside a transaction bound to the authenticated caller's
// organization. It fails closed when no tenant is on the context. On PostgreSQL
// it sets the app.current_org GUC so RLS policies enforce isolation; on other
// drivers the GUC step is skipped.
func WithTenantTx(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	if db == nil {
		return errors.New("nil database handle")
	}
	org := OrgIDFromTenant(authctx.TenantID(ctx))
	if org == 0 {
		return errors.New("tenant context required")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if isPostgres(tx) {
			// set_config(key, value, is_local=true) scopes the GUC to this
			// transaction, so it cannot leak to another request on the pooled
			// connection. Parameterised to avoid any injection via the value.
			if err := tx.Exec("SELECT set_config('app.current_org', ?, true)", strconv.FormatUint(uint64(org), 10)).Error; err != nil {
				return err
			}
		}
		return fn(tx)
	})
}

func isPostgres(db *gorm.DB) bool {
	return db != nil && db.Dialector != nil && db.Dialector.Name() == "postgres"
}
