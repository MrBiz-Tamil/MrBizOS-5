-- Migration 009: Add banking, branding, statutory terms, and template customization options to tenants and invoices
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS bank_name VARCHAR(100) DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_account_no VARCHAR(50) DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_ifsc VARCHAR(20) DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_branch VARCHAR(100) DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_account_holder VARCHAR(150) DEFAULT '',
    ADD COLUMN IF NOT EXISTS upi_id VARCHAR(100) DEFAULT '',
    ADD COLUMN IF NOT EXISTS business_subtitle VARCHAR(200) DEFAULT '',
    ADD COLUMN IF NOT EXISTS signatory_title VARCHAR(100) DEFAULT 'Authorized Signatory',
    ADD COLUMN IF NOT EXISTS signature_url TEXT DEFAULT '',
    ADD COLUMN IF NOT EXISTS default_terms VARCHAR(100) DEFAULT 'Cash',
    ADD COLUMN IF NOT EXISTS default_jurisdiction VARCHAR(100) DEFAULT 'Pune',
    ADD COLUMN IF NOT EXISTS default_interest_rate VARCHAR(50) DEFAULT '24%',
    ADD COLUMN IF NOT EXISTS default_return_policy TEXT DEFAULT 'Goods once sold will not be taken back.',
    ADD COLUMN IF NOT EXISTS template_options JSONB DEFAULT '{}';

-- Per-invoice specific template and statutory fields
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS reverse_charge BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS subscription_period_start TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS subscription_period_end TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS gross_weight VARCHAR(50) DEFAULT '',
    ADD COLUMN IF NOT EXISTS net_weight VARCHAR(50) DEFAULT '',
    ADD COLUMN IF NOT EXISTS cashier_name VARCHAR(100) DEFAULT '',
    ADD COLUMN IF NOT EXISTS counter_number VARCHAR(50) DEFAULT '',
    ADD COLUMN IF NOT EXISTS project_name VARCHAR(150) DEFAULT '',
    ADD COLUMN IF NOT EXISTS template_options JSONB DEFAULT '{}';
