-- 005_accounting_and_subscriptions.sql
-- Multi-Tenant Double-Entry Accounting & Subscription Engine

-- 1. Subscription Plans Master
CREATE TABLE IF NOT EXISTS subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE, -- FREE, STARTER, BUSINESS, ENTERPRISE
    name VARCHAR(100) NOT NULL,
    description TEXT DEFAULT '',
    monthly_price NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    yearly_price NUMERIC(12, 2) NOT NULL DEFAULT 0.00,
    max_users INT NOT NULL DEFAULT 5,
    max_invoices_per_month INT NOT NULL DEFAULT 100,
    features JSONB NOT NULL DEFAULT '{}'::JSONB,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed initial subscription plans
INSERT INTO subscription_plans (code, name, description, monthly_price, yearly_price, max_users, max_invoices_per_month, features)
VALUES 
('FREE', 'Free Trial', 'Basic billing for micro-freelancers & trials', 0.00, 0.00, 1, 20, '{"accounting": true, "whatsapp": true, "email": true, "audit": false}'::JSONB),
('STARTER', 'Starter Tier', 'Standard billing & accounting for small businesses', 999.00, 9990.00, 5, 250, '{"accounting": true, "whatsapp": true, "email": true, "audit": true}'::JSONB),
('BUSINESS', 'Business Tier', 'Full enterprise accounting, WhatsApp sharing, and team collaboration', 2499.00, 24990.00, 25, 2500, '{"accounting": true, "whatsapp": true, "email": true, "audit": true, "priority_support": true}'::JSONB),
('ENTERPRISE', 'Enterprise Unlimited', 'Unlimited corporate ledger, custom branding, and SLA', 5999.00, 59990.00, 999, 999999, '{"accounting": true, "whatsapp": true, "email": true, "audit": true, "custom_branding": true, "unlimited": true}'::JSONB)
ON CONFLICT (code) DO NOTHING;

-- 2. Extend Tenants table with subscription tracking
ALTER TABLE tenants 
ADD COLUMN IF NOT EXISTS subscription_plan_id UUID REFERENCES subscription_plans(id),
ADD COLUMN IF NOT EXISTS billing_cycle VARCHAR(20) NOT NULL DEFAULT 'MONTHLY',
ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS renews_at TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '30 days'),
ADD COLUMN IF NOT EXISTS max_users_override INT,
ADD COLUMN IF NOT EXISTS max_invoices_override INT;

-- Associate existing tenants with their plan ID
UPDATE tenants t
SET subscription_plan_id = sp.id
FROM subscription_plans sp
WHERE UPPER(t.subscription_plan) = sp.code AND t.subscription_plan_id IS NULL;

-- 3. Chart of Accounts (COA) per Tenant
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    account_code VARCHAR(20) NOT NULL, -- e.g. 1100, 1200, 1300, 2100, 2210, 4100
    account_name VARCHAR(150) NOT NULL,
    account_type VARCHAR(30) NOT NULL, -- ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE
    sub_category VARCHAR(100) DEFAULT '',
    description TEXT DEFAULT '',
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    current_balance NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_account_code UNIQUE (tenant_id, account_code)
);

CREATE INDEX IF NOT EXISTS idx_accounts_tenant_type ON accounts(tenant_id, account_type);

-- 4. Journal Entries (Double-Entry Header)
CREATE TABLE IF NOT EXISTS journal_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entry_number VARCHAR(50) NOT NULL, -- e.g. JV-2026-0001
    entry_date DATE NOT NULL DEFAULT CURRENT_DATE,
    reference_type VARCHAR(50) NOT NULL, -- INVOICE, PAYMENT, CANCELLATION, MANUAL
    reference_id UUID,
    narration TEXT NOT NULL,
    total_debit NUMERIC(15, 2) NOT NULL,
    total_credit NUMERIC(15, 2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'POSTED', -- POSTED, REVERSED
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_journal_number UNIQUE (tenant_id, entry_number),
    CONSTRAINT chk_balanced_entry CHECK (total_debit = total_credit)
);

CREATE INDEX IF NOT EXISTS idx_journal_tenant_date ON journal_entries(tenant_id, entry_date DESC);
CREATE INDEX IF NOT EXISTS idx_journal_reference ON journal_entries(tenant_id, reference_type, reference_id);

-- 5. Journal Lines (Debits & Credits)
CREATE TABLE IF NOT EXISTS journal_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id),
    customer_id UUID REFERENCES customers(id),
    debit_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    credit_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    line_narration VARCHAR(255) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_debit_or_credit CHECK (
        (debit_amount > 0 AND credit_amount = 0) OR
        (credit_amount > 0 AND debit_amount = 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_journal_lines_entry ON journal_lines(journal_entry_id);
CREATE INDEX IF NOT EXISTS idx_journal_lines_account ON journal_lines(tenant_id, account_id);
CREATE INDEX IF NOT EXISTS idx_journal_lines_customer ON journal_lines(tenant_id, customer_id);
