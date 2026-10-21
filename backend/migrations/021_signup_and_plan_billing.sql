-- 021_signup_and_plan_billing.sql
-- Self-service signup and billing of our own customers (tenants) for their subscription plan.

-- Users created by self-signup must verify their email before signing in, regardless of the global setting.
ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_required BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS signup_source VARCHAR(10) NOT NULL DEFAULT 'ADMIN';
-- Automatic renewal invoices and suspension apply only to tenants with auto_billing (set when they pick a paid plan);
-- companies set up by a platform admin keep being managed manually until switched on.
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS auto_billing BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS trial_reminder_sent_at TIMESTAMPTZ;

-- Invoices WE issue to tenants for their plan. Status: OPEN (awaiting payment), PAID, VOID.
CREATE SEQUENCE IF NOT EXISTS platform_invoice_seq START 1;
CREATE TABLE IF NOT EXISTS platform_invoices (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_number    VARCHAR(30) NOT NULL UNIQUE,
    plan_code         VARCHAR(50) NOT NULL,
    billing_cycle     VARCHAR(20) NOT NULL,
    period_start      DATE NOT NULL,
    period_end        DATE NOT NULL,
    amount            NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
    gst_amount        NUMERIC(12,2) NOT NULL CHECK (gst_amount >= 0),
    total_amount      NUMERIC(12,2) NOT NULL CHECK (total_amount >= 0),
    status            VARCHAR(10) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'PAID', 'VOID')),
    due_date          DATE NOT NULL,
    paid_at           TIMESTAMPTZ,
    payment_method    VARCHAR(30) NOT NULL DEFAULT '',
    payment_reference VARCHAR(100) NOT NULL DEFAULT '',
    reminder_sent_at  TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, plan_code, period_start)
);
CREATE INDEX IF NOT EXISTS idx_platform_invoices_tenant ON platform_invoices(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_invoices_status ON platform_invoices(status, due_date);

-- Tenant isolation, same pattern as migration 015 (platform jobs run with no tenant set and see everything).
ALTER TABLE platform_invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE platform_invoices FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON platform_invoices;
CREATE POLICY tenant_isolation ON platform_invoices
    USING (NULLIF(current_setting('app.tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (NULLIF(current_setting('app.tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
