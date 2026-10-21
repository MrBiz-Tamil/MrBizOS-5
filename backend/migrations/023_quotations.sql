-- 023_quotations.sql
-- Quotations and proforma invoices. They are NOT tax documents: no GST liability, no stock movement, no ledger entry.
-- Converting one creates a real invoice through the normal invoice service.
CREATE TABLE IF NOT EXISTS quotations (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kind                 VARCHAR(10) NOT NULL DEFAULT 'QUOTATION' CHECK (kind IN ('QUOTATION', 'PROFORMA')),
    quote_number         VARCHAR(40) NOT NULL,
    customer_id          UUID NOT NULL REFERENCES customers(id),
    customer_name        VARCHAR(255) NOT NULL,
    customer_gstin       VARCHAR(15) NOT NULL DEFAULT '',
    billing_address      TEXT NOT NULL DEFAULT '',
    supply_state         VARCHAR(100) NOT NULL DEFAULT '',
    quote_date           DATE NOT NULL,
    valid_until          DATE NOT NULL,
    input_items          JSONB NOT NULL,   -- what the user entered; used to create the invoice on conversion
    lines                JSONB NOT NULL,   -- priced lines as shown on the document
    is_inter_state       BOOLEAN NOT NULL DEFAULT FALSE,
    subtotal             NUMERIC(15,2) NOT NULL,
    discount_total       NUMERIC(15,2) NOT NULL,
    taxable_amount       NUMERIC(15,2) NOT NULL,
    cgst_total           NUMERIC(15,2) NOT NULL,
    sgst_total           NUMERIC(15,2) NOT NULL,
    igst_total           NUMERIC(15,2) NOT NULL,
    round_off            NUMERIC(15,2) NOT NULL,
    grand_total          NUMERIC(15,2) NOT NULL,
    notes                TEXT NOT NULL DEFAULT '',
    terms                TEXT NOT NULL DEFAULT '',
    status               VARCHAR(12) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'SENT', 'ACCEPTED', 'REJECTED', 'CONVERTED')),
    converted_invoice_id UUID,
    created_by           UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, quote_number)
);
CREATE INDEX IF NOT EXISTS idx_quotations_tenant ON quotations(tenant_id, created_at DESC);

ALTER TABLE quotations ENABLE ROW LEVEL SECURITY;
ALTER TABLE quotations FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON quotations;
CREATE POLICY tenant_isolation ON quotations
    USING (NULLIF(current_setting('app.tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (NULLIF(current_setting('app.tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
