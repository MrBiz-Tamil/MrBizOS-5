-- 001_initial_schema.sql
-- Production Multi-Tenant SaaS Billing & Accounting Schema

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Tenants (Companies)
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    legal_name VARCHAR(255) NOT NULL,
    gstin VARCHAR(15) NOT NULL,
    address TEXT NOT NULL,
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100) NOT NULL,
    country VARCHAR(100) NOT NULL DEFAULT 'India',
    pincode VARCHAR(10) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    website VARCHAR(255) DEFAULT '',
    logo_url TEXT DEFAULT '',
    invoice_prefix VARCHAR(10) NOT NULL DEFAULT 'INV',
    financial_year_start VARCHAR(10) NOT NULL DEFAULT '04-01',
    currency VARCHAR(10) NOT NULL DEFAULT 'INR',
    subscription_plan VARCHAR(50) NOT NULL DEFAULT 'STARTER',
    subscription_status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenants_is_active ON tenants(is_active);

-- 2. Users (Super Admin has tenant_id = NULL)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'VIEWER',
    is_super_admin BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- 3. User Sessions
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    ip_address VARCHAR(45) DEFAULT '',
    user_agent TEXT DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON user_sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON user_sessions(expires_at);

-- 4. Centralized Units Master (System global units have tenant_id = NULL)
CREATE TABLE IF NOT EXISTS units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    code VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    category VARCHAR(50) NOT NULL, -- WEIGHT, VOLUME, LENGTH, AREA, QUANTITY
    base_unit_code VARCHAR(20) NOT NULL,
    conversion_factor NUMERIC(15, 6) NOT NULL DEFAULT 1.0,
    decimal_precision INT NOT NULL DEFAULT 2,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_units_tenant_code ON units(COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::UUID), code);
CREATE INDEX IF NOT EXISTS idx_units_category ON units(category);

-- 5. Products Master
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    sku VARCHAR(100) NOT NULL,
    hsn_sac VARCHAR(20) NOT NULL DEFAULT '',
    category VARCHAR(100) NOT NULL DEFAULT 'General',
    description TEXT DEFAULT '',
    purchase_unit_id UUID NOT NULL REFERENCES units(id),
    sales_unit_id UUID NOT NULL REFERENCES units(id),
    stock_unit_id UUID NOT NULL REFERENCES units(id),
    base_unit_id UUID NOT NULL REFERENCES units(id),
    tax_rate NUMERIC(5, 2) NOT NULL DEFAULT 18.00,
    purchase_price NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    selling_price NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    min_stock NUMERIC(15, 4) NOT NULL DEFAULT 0.00,
    max_stock NUMERIC(15, 4) NOT NULL DEFAULT 0.00,
    barcode VARCHAR(100) DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_product_sku UNIQUE (tenant_id, sku)
);

CREATE INDEX IF NOT EXISTS idx_products_tenant ON products(tenant_id);
CREATE INDEX IF NOT EXISTS idx_products_sku ON products(tenant_id, sku);

-- 6. Product-Specific / Custom Unit Conversions
CREATE TABLE IF NOT EXISTS unit_conversions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE CASCADE,
    from_unit_id UUID NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    to_unit_id UUID NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    factor NUMERIC(15, 6) NOT NULL, -- 1 from_unit = factor * to_unit
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_product_unit_conversion UNIQUE (tenant_id, product_id, from_unit_id, to_unit_id)
);

-- 7. Customers
CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    company_name VARCHAR(255) DEFAULT '',
    gstin VARCHAR(15) DEFAULT '',
    email VARCHAR(255) DEFAULT '',
    phone VARCHAR(20) DEFAULT '',
    billing_address TEXT NOT NULL DEFAULT '',
    billing_city VARCHAR(100) DEFAULT '',
    billing_state VARCHAR(100) NOT NULL DEFAULT '',
    billing_pincode VARCHAR(10) DEFAULT '',
    shipping_address TEXT DEFAULT '',
    shipping_state VARCHAR(100) DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_customers_tenant ON customers(tenant_id);

-- 8. Suppliers
CREATE TABLE IF NOT EXISTS suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    company_name VARCHAR(255) DEFAULT '',
    gstin VARCHAR(15) DEFAULT '',
    email VARCHAR(255) DEFAULT '',
    phone VARCHAR(20) DEFAULT '',
    address TEXT DEFAULT '',
    state VARCHAR(100) DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_suppliers_tenant ON suppliers(tenant_id);

-- 9. Tenant Invoicing Sequence (Concurrency Safe Row Locking)
CREATE TABLE IF NOT EXISTS tenant_invoice_sequences (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    financial_year VARCHAR(10) NOT NULL,
    prefix VARCHAR(10) NOT NULL DEFAULT 'INV',
    current_number BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, financial_year, prefix)
);

-- 10. Invoices
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_number VARCHAR(50) NOT NULL,
    invoice_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE NOT NULL DEFAULT CURRENT_DATE + INTERVAL '30 days',
    customer_id UUID NOT NULL REFERENCES customers(id),
    customer_name VARCHAR(255) NOT NULL,
    customer_gstin VARCHAR(15) DEFAULT '',
    billing_address TEXT NOT NULL,
    shipping_address TEXT DEFAULT '',
    supply_state VARCHAR(100) NOT NULL,
    is_inter_state BOOLEAN NOT NULL DEFAULT FALSE,
    subtotal NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    discount_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    taxable_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    cgst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    sgst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    igst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    utgst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    round_off NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    grand_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    amount_paid NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    balance_due NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT', -- DRAFT, ISSUED, PARTIALLY_PAID, PAID, CANCELLED
    notes TEXT DEFAULT '',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_invoice_number UNIQUE (tenant_id, invoice_number)
);

CREATE INDEX IF NOT EXISTS idx_invoices_tenant_date ON invoices(tenant_id, invoice_date DESC);
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_status ON invoices(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_invoices_tenant_customer ON invoices(tenant_id, customer_id);

-- 11. Invoice Items
CREATE TABLE IF NOT EXISTS invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id),
    item_description VARCHAR(255) NOT NULL,
    hsn_sac VARCHAR(20) DEFAULT '',
    quantity NUMERIC(15, 4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id),
    unit_code VARCHAR(20) NOT NULL,
    unit_price NUMERIC(15, 2) NOT NULL,
    discount_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    discount_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    taxable_value NUMERIC(15, 2) NOT NULL,
    tax_rate NUMERIC(5, 2) NOT NULL,
    cgst_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    cgst_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    sgst_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    sgst_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    igst_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    igst_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    total_amount NUMERIC(15, 2) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice ON invoice_items(invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_items_tenant ON invoice_items(tenant_id);

-- 12. Payments
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    customer_id UUID NOT NULL REFERENCES customers(id),
    payment_number VARCHAR(50) NOT NULL,
    payment_date DATE NOT NULL DEFAULT CURRENT_DATE,
    amount NUMERIC(15, 2) NOT NULL,
    payment_method VARCHAR(50) NOT NULL, -- CASH, BANK_TRANSFER, UPI, CARD, CHEQUE, OTHER
    reference_number VARCHAR(100) DEFAULT '',
    notes TEXT DEFAULT '',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_payment_number UNIQUE (tenant_id, payment_number)
);

CREATE INDEX IF NOT EXISTS idx_payments_tenant_date ON payments(tenant_id, payment_date DESC);
CREATE INDEX IF NOT EXISTS idx_payments_invoice ON payments(invoice_id);

-- 13. Inventory Stock
CREATE TABLE IF NOT EXISTS inventory_stock (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    current_stock NUMERIC(15, 4) NOT NULL DEFAULT 0.00,
    reserved_stock NUMERIC(15, 4) NOT NULL DEFAULT 0.00,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_product_stock UNIQUE (tenant_id, product_id)
);

-- 14. Stock Transactions
CREATE TABLE IF NOT EXISTS stock_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    transaction_type VARCHAR(50) NOT NULL, -- PURCHASE, SALE, ADJUSTMENT, RETURN
    quantity NUMERIC(15, 4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id),
    reference_type VARCHAR(50) DEFAULT '', -- INVOICE, MANUAL
    reference_id UUID,
    notes TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_tx_tenant_product ON stock_transactions(tenant_id, product_id);

-- 15. Immutable Audit Logs
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(100) DEFAULT '',
    old_values JSONB,
    new_values JSONB,
    ip_address VARCHAR(45) DEFAULT '',
    user_agent TEXT DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'SUCCESS', -- SUCCESS, FORBIDDEN, FAILED
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_tenant_time ON audit_logs(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
