-- 025_trade_foundation.sql
-- Export / import trade: currencies and exchange rates, the company's trade profile, export-invoice trade details,
-- foreign-currency receipts, and shipments with a document checklist, packing list and cost lines.

-- 1. Currencies (reference data) and per-company exchange rates (1 unit of currency = rate INR)
CREATE TABLE IF NOT EXISTS currencies (
    code     CHAR(3) PRIMARY KEY,
    name     VARCHAR(60) NOT NULL,
    symbol   VARCHAR(8) NOT NULL DEFAULT '',
    decimals INT NOT NULL DEFAULT 2
);
INSERT INTO currencies (code, name, symbol, decimals) VALUES
    ('INR', 'Indian Rupee', 'Rs.', 2), ('USD', 'US Dollar', '$', 2), ('EUR', 'Euro', 'EUR', 2), ('GBP', 'Pound Sterling', 'GBP', 2),
    ('AED', 'UAE Dirham', 'AED', 2), ('SAR', 'Saudi Riyal', 'SAR', 2), ('SGD', 'Singapore Dollar', 'SGD', 2), ('AUD', 'Australian Dollar', 'AUD', 2),
    ('CAD', 'Canadian Dollar', 'CAD', 2), ('JPY', 'Japanese Yen', 'JPY', 0), ('CNY', 'Chinese Yuan', 'CNY', 2), ('CHF', 'Swiss Franc', 'CHF', 2),
    ('HKD', 'Hong Kong Dollar', 'HKD', 2), ('MYR', 'Malaysian Ringgit', 'MYR', 2), ('THB', 'Thai Baht', 'THB', 2), ('ZAR', 'South African Rand', 'ZAR', 2),
    ('KWD', 'Kuwaiti Dinar', 'KWD', 3), ('QAR', 'Qatari Riyal', 'QAR', 2), ('NZD', 'New Zealand Dollar', 'NZD', 2), ('BDT', 'Bangladeshi Taka', 'BDT', 2)
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS exchange_rates (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    currency   CHAR(3) NOT NULL REFERENCES currencies(code),
    rate_date  DATE NOT NULL,
    rate       NUMERIC(18,6) NOT NULL CHECK (rate > 0),
    source     VARCHAR(60) NOT NULL DEFAULT 'manual',
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, currency, rate_date)
);

-- 2. Trade profile of the company
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS iec_code VARCHAR(20) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ad_code VARCHAR(20) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS bank_swift VARCHAR(20) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS lut_number VARCHAR(60) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS lut_valid_from DATE,
    ADD COLUMN IF NOT EXISTS lut_valid_to DATE,
    ADD COLUMN IF NOT EXISTS default_port_loading VARCHAR(100) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS default_port_code VARCHAR(10) NOT NULL DEFAULT '';

-- 3. Trade details of an export invoice (1:1 with invoices). Invoice money columns stay in INR (the books);
--    foreign-currency figures live here.
CREATE TABLE IF NOT EXISTS invoice_trade_details (
    invoice_id          UUID PRIMARY KEY REFERENCES invoices(id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    export_type         VARCHAR(10) NOT NULL CHECK (export_type IN ('LUT', 'IGST', 'SEZ_LUT', 'SEZ_IGST')),
    currency            CHAR(3) NOT NULL REFERENCES currencies(code),
    exchange_rate       NUMERIC(18,6) NOT NULL CHECK (exchange_rate > 0),
    fc_subtotal         NUMERIC(18,2) NOT NULL,
    fc_discount         NUMERIC(18,2) NOT NULL DEFAULT 0,
    fc_freight          NUMERIC(18,2) NOT NULL DEFAULT 0,
    fc_insurance        NUMERIC(18,2) NOT NULL DEFAULT 0,
    fc_total            NUMERIC(18,2) NOT NULL,
    fc_lines            JSONB NOT NULL DEFAULT '[]'::JSONB,
    incoterm            VARCHAR(5) NOT NULL DEFAULT '',
    payment_terms       VARCHAR(200) NOT NULL DEFAULT '',
    buyer_country       VARCHAR(80) NOT NULL DEFAULT '',
    buyer_tax_id        VARCHAR(60) NOT NULL DEFAULT '',
    consignee_name      VARCHAR(255) NOT NULL DEFAULT '',
    consignee_address   TEXT NOT NULL DEFAULT '',
    notify_party        TEXT NOT NULL DEFAULT '',
    country_origin      VARCHAR(80) NOT NULL DEFAULT 'India',
    country_destination VARCHAR(80) NOT NULL DEFAULT '',
    port_loading        VARCHAR(100) NOT NULL DEFAULT '',
    port_discharge      VARCHAR(100) NOT NULL DEFAULT '',
    final_destination   VARCHAR(100) NOT NULL DEFAULT '',
    vessel_flight       VARCHAR(100) NOT NULL DEFAULT '',
    container_no        VARCHAR(100) NOT NULL DEFAULT '',
    bl_awb_no           VARCHAR(60) NOT NULL DEFAULT '',
    bl_awb_date         DATE,
    shipping_bill_no    VARCHAR(30) NOT NULL DEFAULT '',
    shipping_bill_date  DATE,
    port_code           VARCHAR(10) NOT NULL DEFAULT '',
    lut_ref             VARCHAR(60) NOT NULL DEFAULT '',
    marks_numbers       TEXT NOT NULL DEFAULT '',
    total_packages      INT NOT NULL DEFAULT 0,
    gross_weight_kg     NUMERIC(14,3) NOT NULL DEFAULT 0,
    net_weight_kg       NUMERIC(14,3) NOT NULL DEFAULT 0,
    igst_refund_claim   BOOLEAN NOT NULL DEFAULT FALSE,   -- IGST paid on the export and claimed back as refund
    shipment_id         UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_trade_details_tenant ON invoice_trade_details(tenant_id);

-- 4. Foreign-currency receipts: the INR settlement value (payments.amount, at the invoice rate) and what the bank
--    actually credited. The difference is booked to forex gain/loss.
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS fc_currency CHAR(3),
    ADD COLUMN IF NOT EXISTS fc_amount NUMERIC(18,2),
    ADD COLUMN IF NOT EXISTS inr_received NUMERIC(15,2),
    ADD COLUMN IF NOT EXISTS forex_gain_loss NUMERIC(15,2) NOT NULL DEFAULT 0;

-- 5. Shipments
CREATE TABLE IF NOT EXISTS shipments (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    shipment_number    VARCHAR(40) NOT NULL,
    direction          VARCHAR(6) NOT NULL CHECK (direction IN ('EXPORT', 'IMPORT')),
    status             VARCHAR(12) NOT NULL DEFAULT 'PLANNED' CHECK (status IN ('PLANNED', 'BOOKED', 'IN_TRANSIT', 'ARRIVED', 'CLEARED', 'COMPLETED', 'CANCELLED')),
    customer_id        UUID REFERENCES customers(id),
    supplier_id        UUID REFERENCES suppliers(id),
    counterparty_name  VARCHAR(255) NOT NULL DEFAULT '',
    invoice_id         UUID REFERENCES invoices(id) ON DELETE SET NULL,
    purchase_bill_id   UUID,
    incoterm           VARCHAR(5) NOT NULL DEFAULT '',
    mode               VARCHAR(10) NOT NULL DEFAULT 'SEA' CHECK (mode IN ('SEA', 'AIR', 'ROAD', 'RAIL', 'COURIER')),
    port_loading       VARCHAR(100) NOT NULL DEFAULT '',
    port_discharge     VARCHAR(100) NOT NULL DEFAULT '',
    vessel_flight      VARCHAR(100) NOT NULL DEFAULT '',
    container_no       VARCHAR(100) NOT NULL DEFAULT '',
    bl_awb_no          VARCHAR(60) NOT NULL DEFAULT '',
    bl_awb_date        DATE,
    etd                DATE,
    eta                DATE,
    customs_ref_no     VARCHAR(30) NOT NULL DEFAULT '',   -- shipping bill (export) or bill of entry (import) number
    customs_ref_date   DATE,
    port_code          VARCHAR(10) NOT NULL DEFAULT '',
    notes              TEXT NOT NULL DEFAULT '',
    created_by         UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, shipment_number)
);
CREATE INDEX IF NOT EXISTS idx_shipments_tenant ON shipments(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS shipment_documents (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    shipment_id  UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    doc_type     VARCHAR(40) NOT NULL,
    label        VARCHAR(120) NOT NULL,
    issued_by    VARCHAR(10) NOT NULL DEFAULT 'US' CHECK (issued_by IN ('US', 'OTHER')),   -- we generate it, or someone else issues it
    status       VARCHAR(10) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'READY', 'SUBMITTED', 'RECEIVED', 'NA')),
    reference_no VARCHAR(80) NOT NULL DEFAULT '',
    doc_date     DATE,
    notes        TEXT NOT NULL DEFAULT '',
    sort_order   INT NOT NULL DEFAULT 0,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (shipment_id, doc_type)
);

CREATE TABLE IF NOT EXISTS shipment_packages (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    shipment_id    UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    line_no        INT NOT NULL,
    package_from   INT NOT NULL DEFAULT 1,
    package_to     INT NOT NULL DEFAULT 1,
    package_type   VARCHAR(40) NOT NULL DEFAULT 'CARTON',
    description    VARCHAR(255) NOT NULL,
    quantity       NUMERIC(14,3) NOT NULL DEFAULT 0,
    unit           VARCHAR(20) NOT NULL DEFAULT '',
    net_weight_kg  NUMERIC(14,3) NOT NULL DEFAULT 0,
    gross_weight_kg NUMERIC(14,3) NOT NULL DEFAULT 0,
    dimensions     VARCHAR(60) NOT NULL DEFAULT '',
    marks          VARCHAR(200) NOT NULL DEFAULT '',
    UNIQUE (shipment_id, line_no)
);

-- Costs of a shipment (freight, duty, clearing ...). For imports these feed the landed-cost calculation.
CREATE TABLE IF NOT EXISTS shipment_costs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    shipment_id  UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    cost_type    VARCHAR(20) NOT NULL CHECK (cost_type IN ('FREIGHT', 'INSURANCE', 'BCD', 'SWS', 'IGST', 'CLEARING', 'PORT', 'DEMURRAGE', 'BANK', 'OTHER')),
    description  VARCHAR(200) NOT NULL DEFAULT '',
    amount_inr   NUMERIC(15,2) NOT NULL CHECK (amount_inr >= 0),
    recoverable  BOOLEAN NOT NULL DEFAULT FALSE,   -- credit is claimable (IGST), so it is not part of the landed cost
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Row-level security, same pattern as migration 015
DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['exchange_rates', 'invoice_trade_details', 'shipments', 'shipment_documents', 'shipment_packages', 'shipment_costs']
    LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);
        EXECUTE format($p$CREATE POLICY tenant_isolation ON %I
            USING (NULLIF(current_setting('app.tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
            WITH CHECK (NULLIF(current_setting('app.tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)$p$, t);
    END LOOP;
END $$;
