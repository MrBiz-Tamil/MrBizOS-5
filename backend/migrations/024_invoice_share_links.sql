-- 024_invoice_share_links.sql
-- Secret links that let a customer view an invoice without signing in. Only a hash of the token is stored.
CREATE TABLE IF NOT EXISTS invoice_share_links (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id     UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    token_hash     CHAR(64) NOT NULL UNIQUE,
    expires_at     TIMESTAMPTZ NOT NULL,
    revoked_at     TIMESTAMPTZ,
    views          INT NOT NULL DEFAULT 0,
    last_viewed_at TIMESTAMPTZ,
    created_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_share_links_invoice ON invoice_share_links(invoice_id);

ALTER TABLE invoice_share_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoice_share_links FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON invoice_share_links;
CREATE POLICY tenant_isolation ON invoice_share_links
    USING (NULLIF(current_setting('app.tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (NULLIF(current_setting('app.tenant_id', true), '') IS NULL OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);
