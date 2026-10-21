-- 006_plan_tiers_and_expenses.sql
-- Tiered Subscription Gating (Basic Plan vs Next Plan) & Multi-Tenant Expense Management

-- 1. Ensure Basic and Next/Business plans exist with explicit feature gating flags
INSERT INTO subscription_plans (code, name, description, monthly_price, yearly_price, max_users, max_invoices_per_month, features, is_active)
VALUES 
('BASIC', 'Basic Plan', 'Core billing, consumer management, and customer credit & debit ledger', 499.00, 4990.00, 3, 200, '{"billing": true, "customers": true, "customer_ledger": true, "accounting": false, "expenses": false, "audit": false}'::JSONB, true),
('PRO', 'Next Plan (Business)', 'Full double-entry accounting, complete expense management, and compliance audit trail', 1999.00, 19990.00, 20, 2500, '{"billing": true, "customers": true, "customer_ledger": true, "accounting": true, "expenses": true, "audit": true}'::JSONB, true)
ON CONFLICT (code) DO NOTHING;

-- Update existing plans with consistent feature flags
UPDATE subscription_plans 
SET features = '{"billing": true, "customers": true, "customer_ledger": true, "accounting": false, "expenses": false, "audit": false}'::JSONB,
    name = 'Basic Tier (Free / Starter)',
    description = 'Core invoicing, customer management, and customer credit & debit ledger'
WHERE code IN ('FREE', 'STARTER');

UPDATE subscription_plans 
SET features = '{"billing": true, "customers": true, "customer_ledger": true, "accounting": true, "expenses": true, "audit": true}'::JSONB,
    name = 'Next Plan (Business Tier)',
    description = 'Full double-entry accounting, complete expense management, and security audit'
WHERE code IN ('BUSINESS', 'PRO');

UPDATE subscription_plans 
SET features = '{"billing": true, "customers": true, "customer_ledger": true, "accounting": true, "expenses": true, "audit": true, "custom_branding": true, "unlimited": true}'::JSONB
WHERE code = 'ENTERPRISE';

-- 2. Multi-Tenant Expense Management Table
CREATE TABLE IF NOT EXISTS expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    expense_number VARCHAR(50) NOT NULL,
    category VARCHAR(100) NOT NULL, -- Rent, Utilities, Office Supplies, Salaries, Travel, Maintenance, Vendor Services, Other
    amount NUMERIC(15, 2) NOT NULL,
    tax_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    payment_mode VARCHAR(50) NOT NULL DEFAULT 'CASH', -- CASH, BANK_TRANSFER, UPI, CARD, CHEQUE
    payee_name VARCHAR(150) NOT NULL,
    expense_date DATE NOT NULL DEFAULT CURRENT_DATE,
    description TEXT DEFAULT '',
    receipt_url TEXT DEFAULT '',
    account_id UUID REFERENCES accounts(id),
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_expense_number UNIQUE (tenant_id, expense_number)
);

CREATE INDEX IF NOT EXISTS idx_expenses_tenant_date ON expenses(tenant_id, expense_date DESC);
CREATE INDEX IF NOT EXISTS idx_expenses_tenant_cat ON expenses(tenant_id, category);
CREATE INDEX IF NOT EXISTS idx_expenses_tenant_payee ON expenses(tenant_id, payee_name);
