-- 007_digital_products_and_concurrency_hardening.sql
-- Digital Products, License Key Vault, Atomic Concurrency Sequencing, and Multi-Warehouse Readiness

-- 1. Extend products table for Digital & Service items
ALTER TABLE products 
ADD COLUMN IF NOT EXISTS product_type VARCHAR(20) NOT NULL DEFAULT 'PHYSICAL',
ADD COLUMN IF NOT EXISTS fulfillment_mode VARCHAR(30) NOT NULL DEFAULT 'NONE',
ADD COLUMN IF NOT EXISTS digital_asset_url TEXT DEFAULT '',
ADD COLUMN IF NOT EXISTS download_limit INT DEFAULT 5,
ADD COLUMN IF NOT EXISTS link_expiry_hours INT DEFAULT 48;

CREATE INDEX IF NOT EXISTS idx_products_type ON products(tenant_id, product_type);

-- 2. Digital License Key Vault
CREATE TABLE IF NOT EXISTS digital_license_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    license_key TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'AVAILABLE', -- AVAILABLE, ALLOCATED, REVOKED, EXPIRED
    invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    allocated_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_product_license UNIQUE (tenant_id, product_id, license_key)
);

CREATE INDEX IF NOT EXISTS idx_license_keys_status ON digital_license_keys(tenant_id, product_id, status);
CREATE INDEX IF NOT EXISTS idx_license_keys_invoice ON digital_license_keys(invoice_id);

-- 3. Unified Atomic Concurrency Sequence Engine
CREATE TABLE IF NOT EXISTS tenant_financial_sequences (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sequence_type VARCHAR(20) NOT NULL, -- PAYMENT, EXPENSE, JOURNAL, INVOICE
    period VARCHAR(10) NOT NULL,        -- e.g. '2026', '2026-27', 'ALL'
    prefix VARCHAR(10) NOT NULL,        -- PAY, EXP, JV, INV
    current_number BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, sequence_type, period, prefix)
);

-- Seed initial sequences from existing counts so there is no gap or collision
INSERT INTO tenant_financial_sequences (tenant_id, sequence_type, period, prefix, current_number)
SELECT 
    t.id, 
    'PAYMENT', 
    EXTRACT(YEAR FROM CURRENT_DATE)::TEXT, 
    'PAY', 
    COALESCE(p_count.cnt, 0)
FROM tenants t
LEFT JOIN (
    SELECT tenant_id, COUNT(*) as cnt 
    FROM payments 
    WHERE EXTRACT(YEAR FROM payment_date) = EXTRACT(YEAR FROM CURRENT_DATE)
    GROUP BY tenant_id
) p_count ON t.id = p_count.tenant_id
ON CONFLICT (tenant_id, sequence_type, period, prefix) DO NOTHING;

INSERT INTO tenant_financial_sequences (tenant_id, sequence_type, period, prefix, current_number)
SELECT 
    t.id, 
    'EXPENSE', 
    EXTRACT(YEAR FROM CURRENT_DATE)::TEXT, 
    'EXP', 
    COALESCE(e_count.cnt, 0)
FROM tenants t
LEFT JOIN (
    SELECT tenant_id, COUNT(*) as cnt 
    FROM expenses 
    WHERE EXTRACT(YEAR FROM expense_date) = EXTRACT(YEAR FROM CURRENT_DATE)
    GROUP BY tenant_id
) e_count ON t.id = e_count.tenant_id
ON CONFLICT (tenant_id, sequence_type, period, prefix) DO NOTHING;

-- 4. Multi-Warehouse & Scalability Foundation
CREATE TABLE IF NOT EXISTS warehouses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    address TEXT DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_warehouse_code UNIQUE (tenant_id, code)
);

CREATE INDEX IF NOT EXISTS idx_warehouses_tenant ON warehouses(tenant_id);

-- Auto-seed default warehouse for all tenants (zero setup needed for small stores)
INSERT INTO warehouses (tenant_id, code, name, is_default)
SELECT id, 'MAIN', 'Main Store / Counter', TRUE
FROM tenants
ON CONFLICT (tenant_id, code) DO NOTHING;
