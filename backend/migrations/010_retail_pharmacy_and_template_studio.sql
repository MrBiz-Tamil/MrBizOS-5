-- Migration 010: Retail Supermarket & Pharmacy Batches, Price Management, Bill Operations, and Bill Template Studio

-- 1. Product Batches for Pharmacy / Retail Expiry & Batch-wise Pricing
CREATE TABLE IF NOT EXISTS product_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    batch_number VARCHAR(100) NOT NULL,
    expiry_date DATE NOT NULL,
    mrp NUMERIC(15,2) NOT NULL DEFAULT 0.00,
    selling_price NUMERIC(15,2) NOT NULL DEFAULT 0.00,
    purchase_price NUMERIC(15,2) NOT NULL DEFAULT 0.00,
    current_stock NUMERIC(15,4) NOT NULL DEFAULT 0.0000,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_batches_tenant_product_expiry ON product_batches(tenant_id, product_id, expiry_date);

-- 2. Products table extensions for Pharmacy Generic Salts, Shelf Location, MRP, and Rx Flags
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS generic_name VARCHAR(255) DEFAULT '',
    ADD COLUMN IF NOT EXISTS rack_location VARCHAR(100) DEFAULT '',
    ADD COLUMN IF NOT EXISTS is_prescription_required BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS mrp NUMERIC(15,2) DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS discount_rate NUMERIC(5,2) DEFAULT 0.00;

-- 3. Invoices table extensions for Retail Tender, Change, and Doctor / Patient Details
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS doctor_name VARCHAR(255) DEFAULT '',
    ADD COLUMN IF NOT EXISTS doctor_reg_no VARCHAR(100) DEFAULT '',
    ADD COLUMN IF NOT EXISTS patient_name VARCHAR(255) DEFAULT '',
    ADD COLUMN IF NOT EXISTS patient_age INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tendered_cash NUMERIC(15,2) DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS change_amount NUMERIC(15,2) DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS split_payments JSONB DEFAULT '[]'::JSONB;

-- 4. Invoice items table extensions for Batch Number, Expiry, and MRP
ALTER TABLE invoice_items
    ADD COLUMN IF NOT EXISTS batch_id UUID REFERENCES product_batches(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS batch_number VARCHAR(100) DEFAULT '',
    ADD COLUMN IF NOT EXISTS expiry_date VARCHAR(50) DEFAULT '',
    ADD COLUMN IF NOT EXISTS mrp NUMERIC(15,2) DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS rack_location VARCHAR(100) DEFAULT '';

-- 5. Tenants table extensions for Drug License, FSSAI, and Template Studio Config
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS drug_license_no VARCHAR(150) DEFAULT '',
    ADD COLUMN IF NOT EXISTS fssai_no VARCHAR(100) DEFAULT '',
    ADD COLUMN IF NOT EXISTS template_studio_config JSONB DEFAULT '{}'::JSONB;
