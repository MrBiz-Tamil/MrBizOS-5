-- Migration 008: Add logistics, transport and statutory legal terms to invoices table
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS vehicle_number VARCHAR(50) DEFAULT '',
    ADD COLUMN IF NOT EXISTS delivery_place VARCHAR(150) DEFAULT '',
    ADD COLUMN IF NOT EXISTS payment_mode VARCHAR(50) DEFAULT 'Cash',
    ADD COLUMN IF NOT EXISTS po_number VARCHAR(50) DEFAULT '',
    ADD COLUMN IF NOT EXISTS po_date TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS eway_bill VARCHAR(50) DEFAULT '',
    ADD COLUMN IF NOT EXISTS transporter_name VARCHAR(150) DEFAULT '',
    ADD COLUMN IF NOT EXISTS lr_number VARCHAR(50) DEFAULT '',
    ADD COLUMN IF NOT EXISTS jurisdiction_city VARCHAR(100) DEFAULT 'Pune',
    ADD COLUMN IF NOT EXISTS interest_rate VARCHAR(50) DEFAULT '24%',
    ADD COLUMN IF NOT EXISTS return_policy TEXT DEFAULT 'Goods once sold will not be taken back.';

-- Index for searching invoices by vehicle number or PO number
CREATE INDEX IF NOT EXISTS idx_invoices_vehicle_number ON invoices(tenant_id, vehicle_number) WHERE vehicle_number != '';
CREATE INDEX IF NOT EXISTS idx_invoices_po_number ON invoices(tenant_id, po_number) WHERE po_number != '';
