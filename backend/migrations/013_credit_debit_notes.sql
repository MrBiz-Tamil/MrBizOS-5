-- Migration 013: Statutory Returns, Credit Notes & Debit Notes (GST Section 34 Compliant)

CREATE TABLE IF NOT EXISTS credit_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    credit_note_number VARCHAR(100) NOT NULL,
    credit_note_date DATE NOT NULL DEFAULT CURRENT_DATE,
    invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
    invoice_number VARCHAR(100) NOT NULL DEFAULT '',
    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    customer_name VARCHAR(255) NOT NULL,
    customer_gstin VARCHAR(15) DEFAULT '',
    reason_code VARCHAR(50) NOT NULL DEFAULT 'SALES_RETURN', -- SALES_RETURN, POST_SALE_DISCOUNT, DEFICIENCY_IN_SERVICE, CORRECTION_IN_INVOICE, OTHER
    reason_description TEXT DEFAULT '',
    subtotal NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    taxable_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    cgst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    sgst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    igst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    round_off NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    grand_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    status VARCHAR(30) NOT NULL DEFAULT 'ISSUED', -- ISSUED, APPLIED, CANCELLED
    restore_inventory BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_credit_note_number UNIQUE (tenant_id, credit_note_number)
);

CREATE INDEX IF NOT EXISTS idx_credit_notes_tenant_date ON credit_notes(tenant_id, credit_note_date DESC);
CREATE INDEX IF NOT EXISTS idx_credit_notes_tenant_invoice ON credit_notes(tenant_id, invoice_id);
CREATE INDEX IF NOT EXISTS idx_credit_notes_tenant_customer ON credit_notes(tenant_id, customer_id);

CREATE TABLE IF NOT EXISTS credit_note_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    credit_note_id UUID NOT NULL REFERENCES credit_notes(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    item_description VARCHAR(255) NOT NULL,
    batch_number VARCHAR(50) DEFAULT '',
    hsn_sac VARCHAR(20) DEFAULT '',
    quantity NUMERIC(12, 4) NOT NULL DEFAULT 0,
    unit_code VARCHAR(20) NOT NULL DEFAULT 'PCS',
    unit_price NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    taxable_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    gst_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    cgst_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    sgst_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    igst_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    total_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_credit_note_items_cn ON credit_note_items(credit_note_id);

CREATE TABLE IF NOT EXISTS debit_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    debit_note_number VARCHAR(100) NOT NULL,
    debit_note_date DATE NOT NULL DEFAULT CURRENT_DATE,
    purchase_bill_id UUID REFERENCES purchase_bills(id) ON DELETE SET NULL,
    bill_number VARCHAR(100) NOT NULL DEFAULT '',
    supplier_id UUID REFERENCES suppliers(id) ON DELETE SET NULL,
    supplier_name VARCHAR(255) NOT NULL,
    supplier_gstin VARCHAR(15) DEFAULT '',
    reason_code VARCHAR(50) NOT NULL DEFAULT 'PURCHASE_RETURN', -- PURCHASE_RETURN, POST_PURCHASE_DISCOUNT, DEFICIENCY_IN_GOODS, OTHER
    reason_description TEXT DEFAULT '',
    subtotal NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    taxable_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    cgst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    sgst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    igst_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    round_off NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    grand_total NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    status VARCHAR(30) NOT NULL DEFAULT 'ISSUED',
    deduct_inventory BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_debit_note_number UNIQUE (tenant_id, debit_note_number)
);

CREATE INDEX IF NOT EXISTS idx_debit_notes_tenant_date ON debit_notes(tenant_id, debit_note_date DESC);
CREATE INDEX IF NOT EXISTS idx_debit_notes_tenant_bill ON debit_notes(tenant_id, purchase_bill_id);
CREATE INDEX IF NOT EXISTS idx_debit_notes_tenant_supplier ON debit_notes(tenant_id, supplier_id);

CREATE TABLE IF NOT EXISTS debit_note_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    debit_note_id UUID NOT NULL REFERENCES debit_notes(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE SET NULL,
    item_description VARCHAR(255) NOT NULL,
    batch_number VARCHAR(50) DEFAULT '',
    hsn_sac VARCHAR(20) DEFAULT '',
    quantity NUMERIC(12, 4) NOT NULL DEFAULT 0,
    unit_code VARCHAR(20) NOT NULL DEFAULT 'PCS',
    unit_price NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    taxable_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    gst_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    cgst_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    sgst_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    igst_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    total_amount NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_debit_note_items_dn ON debit_note_items(debit_note_id);
