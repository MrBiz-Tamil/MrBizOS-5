-- 027_drayya_rls_tenant_isolation.sql
--
-- Row-Level Security (ADR-5) for the drayya-core tenant-scoped tables. This is
-- the database-layer complement to the application-layer OrgScope filtering: with
-- these policies in force, PostgreSQL refuses to return or modify any row whose
-- organization_id does not match the current transaction's `app.current_org`
-- GUC, so a missed scope in a handler can no longer leak cross-tenant data.
--
-- The GUC is set per request by database.WithTenantTx (internal/database/rls.go)
-- via set_config('app.current_org', <org>, true). When the GUC is unset,
-- current_setting(..., true) returns NULL and every policy evaluates false, so
-- access fails CLOSED.
--
-- STATUS / PREREQUISITES (important):
--   * drayya-core currently builds its schema with GORM AutoMigrate, NOT a SQL
--     migration runner, so this file is NOT applied automatically. It takes
--     effect only once the SQL-migration mechanism is adopted (decision DEC-04).
--   * The application's DB role MUST NOT be a superuser or the table owner with
--     BYPASSRLS, or RLS is skipped. FORCE ROW LEVEL SECURITY below also applies
--     the policy to the table owner.
--   * Every query path must run inside WithTenantTx (or otherwise set
--     app.current_org); plain pooled queries without the GUC will be denied.
--   * NOT YET VERIFIED against a live PostgreSQL instance.
--
-- Safe to re-run: policies are dropped-if-exists before being recreated.

DO $$
DECLARE
    t text;
    tables text[] := ARRAY[
        'tour_packages',
        'visa_applications',
        'travel_bookings',
        'properties',
        'site_visits',
        'property_deals',
        'tasks'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        -- Only touch tables that exist and carry an organization_id column.
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = t AND column_name = 'organization_id'
        ) THEN
            EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
            EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
            EXECUTE format('DROP POLICY IF EXISTS tenant_isolation_%I ON %I', t, t);
            EXECUTE format(
                'CREATE POLICY tenant_isolation_%I ON %I '
                || 'USING (organization_id = current_setting(''app.current_org'', true)::bigint) '
                || 'WITH CHECK (organization_id = current_setting(''app.current_org'', true)::bigint)',
                t, t
            );
        END IF;
    END LOOP;
END $$;
