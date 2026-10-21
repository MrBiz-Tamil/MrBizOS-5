-- 022_recurring_invoices.sql
-- Invoice schedules: a template invoice that is issued automatically every week/month/quarter/year.
CREATE TABLE IF NOT EXISTS recurring_invoices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id     UUID NOT NULL REFERENCES customers(id),
    name            VARCHAR(120) NOT NULL,
    frequency       VARCHAR(10) NOT NULL CHECK (frequency IN ('WEEKLY', 'MONTHLY', 'QUARTERLY', 'YEARLY')),
    start_date      DATE NOT NULL,
    anchor_day      INT NOT NULL,                 -- day of month the schedule started on (month-end clamping keeps it)
    next_run_date   DATE NOT NULL,
    end_date        DATE,
    due_days        INT NOT NULL DEFAULT 15 CHECK (due_days BETWEEN 0 AND 365),
    billing_address TEXT NOT NULL DEFAULT '',
    supply_state    VARCHAR(100) NOT NULL DEFAULT '',
    notes           TEXT NOT NULL DEFAULT '',
    payment_mode    VARCHAR(50) NOT NULL DEFAULT '',
    items           JSONB NOT NULL,
    status          VARCHAR(10) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'PAUSED', 'ENDED')),
    runs_count      INT NOT NULL DEFAULT 0,
    last_run_at     TIMESTAMPTZ,
    last_invoice_id UUID,
    retry_for       DATE,                         -- a run that failed and is waiting to be retried
    retry_count     INT NOT NULL DEFAULT 0,
    last_error      TEXT NOT NULL DEFAULT '',
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_recurring_due ON recurring_invoices(status, next_run_date);
CREATE INDEX IF NOT EXISTS idx_recurring_tenant ON recurring_invoices(tenant_id);

ALTER TABLE recurring_invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE recurring_invoices FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON recurring_invoices;
CREATE POLICY tenant_isolation ON recurring_invoices
    USING (NULLIF(current_setting('app.tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (NULLIF(current_setting('app.tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
