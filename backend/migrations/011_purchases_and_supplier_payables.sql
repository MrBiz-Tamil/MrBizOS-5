-- Migration 011: Purchase Invoices, Inward Stock Items, and Supplier Payables
-- Enables business trade cycle: Inward vendor billing, batch/expiry creation, stock auto-increment, and supplier Khata

-- 1. Ensure suppliers table has complete contact, banking and balance fields
ALTER TABLE suppliers
    ADD COLUMN IF NOT EXISTS city VARCHAR(100) DEFAULT '',
    ADD COLUMN IF NOT EXISTS pincode VARCHAR(10) DEFAULT '',
    ADD COLUMN IF NOT EXISTS pan VARCHAR(20) DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_name VARCHAR(100) DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_account_no VARCHAR(50) DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_ifsc VARCHAR(20) DEFAULT '',
    ADD COLUMN IF NOT EXISTS opening_balance NUMERIC(15,2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE INDEX IF NOT EXISTS idx_suppliers_tenant_active ON suppliers(tenant_id, is_active);
CREATE INDEX IF NOT EXISTS idx_suppliers_gstin ON suppliers(tenant_id, gstin) WHERE gstin != '';

-- 2. Purchase Bills (Inward Invoices from Suppliers)
CREATE TABLE IF NOT EXISTS purchase_bills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    supplier_id UUID NOT NULL REFERENCES suppliers(id),
    bill_number VARCHAR(100) NOT NULL,
    bill_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE NOT NULL DEFAULT CURRENT_DATE + INTERVAL '30 days',
    supplier_name VARCHAR(255) NOT NULL,
    supplier_gstin VARCHAR(15) DEFAULT '',
    supplier_address TEXT DEFAULT '',
    supplier_state VARCHAR(100) NOT NULL DEFAULT '',
    is_inter_state BOOLEAN NOT NULL DEFAULT FALSE,
    vehicle_number VARCHAR(50) DEFAULT '',
    po_number VARCHAR(50) DEFAULT '',
    subtotal NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    discount_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    taxable_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    cgst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    sgst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    igst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    round_off NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    grand_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    amount_paid NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    balance_due NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    status VARCHAR(30) NOT NULL DEFAULT 'RECEIVED', -- RECEIVED, PARTIALLY_PAID, PAID, CANCELLED
    notes TEXT DEFAULT '',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_supplier_bill UNIQUE (tenant_id, supplier_id, bill_number)
);

CREATE INDEX IF NOT EXISTS idx_purchase_bills_tenant_date ON purchase_bills(tenant_id, bill_date DESC);
CREATE INDEX IF NOT EXISTS idx_purchase_bills_tenant_status ON purchase_bills(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_purchase_bills_tenant_supplier ON purchase_bills(tenant_id, supplier_id);

-- 3. Purchase Items (Line items with Batch & Stock parameters)
CREATE TABLE IF NOT EXISTS purchase_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    purchase_bill_id UUID NOT NULL REFERENCES purchase_bills(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    item_description VARCHAR(255) NOT NULL,
    hsn_sac VARCHAR(20) DEFAULT '',
    batch_number VARCHAR(100) DEFAULT '',
    expiry_date DATE,
    quantity NUMERIC(15, 4) NOT NULL,
    unit_id UUID NOT NULL REFERENCES units(id),
    unit_code VARCHAR(20) NOT NULL,
    purchase_price NUMERIC(15, 2) NOT NULL,
    discount_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    discount_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    taxable_value NUMERIC(15, 2) NOT NULL,
    tax_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    cgst_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    cgst_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    sgst_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    sgst_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    igst_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    igst_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    total_amount NUMERIC(15, 2) NOT NULL,
    mrp NUMERIC(15, 2) DEFAULT 0.00,
    selling_price NUMERIC(15, 2) DEFAULT 0.00,
    rack_location VARCHAR(100) DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_purchase_items_bill ON purchase_items(purchase_bill_id);
CREATE INDEX IF NOT EXISTS idx_purchase_items_tenant_product ON purchase_items(tenant_id, product_id);

-- 4. Supplier Payments (Outgoing Payments against Purchase Bills)
CREATE TABLE IF NOT EXISTS supplier_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    purchase_bill_id UUID REFERENCES purchase_bills(id) ON DELETE SET NULL,
    supplier_id UUID NOT NULL REFERENCES suppliers(id),
    payment_number VARCHAR(50) NOT NULL,
    payment_date DATE NOT NULL DEFAULT CURRENT_DATE,
    amount NUMERIC(15, 2) NOT NULL,
    payment_method VARCHAR(50) NOT NULL, -- CASH, BANK_TRANSFER, UPI, CHEQUE, OTHER
    reference_number VARCHAR(100) DEFAULT '',
    notes TEXT DEFAULT '',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_supplier_payment UNIQUE (tenant_id, payment_number)
);

CREATE INDEX IF NOT EXISTS idx_supplier_payments_tenant_date ON supplier_payments(tenant_id, payment_date DESC);
CREATE INDEX IF NOT EXISTS idx_supplier_payments_bill ON supplier_payments(purchase_bill_id);
CREATE INDEX IF NOT EXISTS idx_supplier_payments_supplier ON supplier_payments(tenant_id, supplier_id);
