-- 015_row_level_security.sql
-- Defence-in-depth tenant isolation. The application already filters every query by tenant_id; these
-- policies make PostgreSQL enforce it too, so a missing WHERE clause (or a future bug) cannot leak or
-- modify another tenant's rows for tenant-scoped requests.
--
-- How it works: the backend connection pool sets the session variable app.tenant_id for requests made
-- by tenant users / API keys (see internal/database/db.go). A row is visible/writable when
--   * the variable is unset/empty (platform super admin, login, background workers: not tenant scoped), or
--   * the row has no tenant (global reference data such as seeded units), or
--   * the row's tenant_id equals the variable.
-- FORCE ROW LEVEL SECURITY makes the policies apply to the table owner (the application role) as well.
--
-- Not covered: users, user_sessions (looked up before a tenant is known) and tenants (no tenant_id column).
-- New tenant-owned tables created after this migration must repeat the pattern below.

DO $$
DECLARE
    t RECORD;
BEGIN
    FOR t IN
        SELECT c.relname AS table_name
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_attribute a ON a.attrelid = c.oid AND a.attname = 'tenant_id' AND NOT a.attisdropped
        WHERE n.nspname = 'public'
          AND c.relkind IN ('r', 'p')
          AND NOT c.relispartition
          AND c.relname NOT IN ('users', 'user_sessions')
    LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t.table_name);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t.table_name);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t.table_name);
        EXECUTE format($p$
            CREATE POLICY tenant_isolation ON %I
            USING (
                NULLIF(current_setting('app.tenant_id', true), '') IS NULL
                OR tenant_id IS NULL
                OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
            )
            WITH CHECK (
                NULLIF(current_setting('app.tenant_id', true), '') IS NULL
                OR tenant_id IS NULL
                OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
            )
        $p$, t.table_name);
    END LOOP;
END $$;
