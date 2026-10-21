-- 020_legal_acceptance.sql
-- Records who accepted which version of the Terms of Service / Privacy Policy, when, and from where.
-- Evidence table: rows are only ever inserted.
CREATE TABLE IF NOT EXISTS terms_acceptances (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id   UUID REFERENCES tenants(id) ON DELETE SET NULL,
    document    VARCHAR(20) NOT NULL CHECK (document IN ('TERMS', 'PRIVACY')),
    version     VARCHAR(40) NOT NULL,
    accepted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address  VARCHAR(64) NOT NULL DEFAULT '',
    user_agent  VARCHAR(300) NOT NULL DEFAULT '',
    UNIQUE (user_id, document, version)
);
CREATE INDEX IF NOT EXISTS idx_terms_acceptances_user ON terms_acceptances(user_id);
