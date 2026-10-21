-- 026_business_types_and_weighment.sql
-- Business type presets (set at signup), shorter e-invoice-safe invoice numbers, and weighbridge slips for
-- quarry / blue-metal / aggregate sellers.
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS business_type VARCHAR(20) NOT NULL DEFAULT 'OTHER',
    ADD COLUMN IF NOT EXISTS invoice_number_style VARCHAR(5) NOT NULL DEFAULT 'LONG';
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenants_number_style;
ALTER TABLE tenants ADD CONSTRAINT chk_tenants_number_style CHECK (invoice_number_style IN ('LONG', 'SHORT'));

CREATE TABLE IF NOT EXISTS weighment_slips (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    slip_number    VARCHAR(40) NOT NULL,
    slip_date      DATE NOT NULL,
    vehicle_no     VARCHAR(20) NOT NULL,
    driver_name    VARCHAR(100) NOT NULL DEFAULT '',
    driver_phone   VARCHAR(20) NOT NULL DEFAULT '',
    customer_id    UUID REFERENCES customers(id),
    customer_name  VARCHAR(255) NOT NULL DEFAULT '',
    product_id     UUID REFERENCES products(id),
    material       VARCHAR(255) NOT NULL,
    hsn_sac        VARCHAR(20) NOT NULL DEFAULT '2517',
    gross_kg       NUMERIC(12,2) NOT NULL CHECK (gross_kg > 0),
    tare_kg        NUMERIC(12,2) NOT NULL CHECK (tare_kg >= 0),
    net_kg         NUMERIC(12,2) NOT NULL CHECK (net_kg > 0),
    rate_per_tonne NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (rate_per_tonne >= 0),
    tax_rate       NUMERIC(5,2) NOT NULL DEFAULT 5 CHECK (tax_rate >= 0 AND tax_rate <= 100),
    destination    VARCHAR(255) NOT NULL DEFAULT '',
    permit_no      VARCHAR(80) NOT NULL DEFAULT '',   -- transit pass / permit / royalty challan reference
    notes          TEXT NOT NULL DEFAULT '',
    status         VARCHAR(10) NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'INVOICED', 'CANCELLED')),
    invoice_id     UUID REFERENCES invoices(id) ON DELETE SET NULL,
    created_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, slip_number),
    CHECK (net_kg = gross_kg - tare_kg)
);
CREATE INDEX IF NOT EXISTS idx_weighment_tenant_date ON weighment_slips(tenant_id, slip_date DESC);

ALTER TABLE weighment_slips ENABLE ROW LEVEL SECURITY;
ALTER TABLE weighment_slips FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON weighment_slips;
CREATE POLICY tenant_isolation ON weighment_slips
    USING (NULLIF(current_setting('app.tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (NULLIF(current_setting('app.tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
