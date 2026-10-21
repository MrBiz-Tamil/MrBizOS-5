-- 012_enterprise_ledger_hardening_and_metering.sql
-- Enterprise Ledger Immutability, Cryptographic Hash Chaining, Partitioned Metering & Security

-- 1. Extend journal_entries with cryptographic audit chain
ALTER TABLE journal_entries 
ADD COLUMN IF NOT EXISTS sequence_num BIGINT,
ADD COLUMN IF NOT EXISTS prev_hash BYTEA,
ADD COLUMN IF NOT EXISTS entry_hash BYTEA;

-- Create sequence for tenant-level journal ordering if not present
CREATE SEQUENCE IF NOT EXISTS journal_entry_global_seq START WITH 1;

-- Backfill sequence_num for existing entries
UPDATE journal_entries
SET sequence_num = nextval('journal_entry_global_seq')
WHERE sequence_num IS NULL;

ALTER TABLE journal_entries 
ALTER COLUMN sequence_num SET DEFAULT nextval('journal_entry_global_seq'),
ALTER COLUMN sequence_num SET NOT NULL;

-- 2. Immutability Trigger: Absolute freeze on journal_entries and journal_lines
CREATE OR REPLACE FUNCTION fn_prevent_ledger_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'CRITICAL SECURITY VIOLATION: Ledger entries and lines are strictly append-only and immutable. Cannot % on table %', TG_OP, TG_TABLE_NAME
        USING ERRCODE = 'integrity_constraint_violation';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_freeze_journal_entries ON journal_entries;
CREATE TRIGGER trg_freeze_journal_entries
BEFORE UPDATE OR DELETE ON journal_entries
FOR EACH ROW EXECUTE FUNCTION fn_prevent_ledger_mutation();

DROP TRIGGER IF EXISTS trg_freeze_journal_lines ON journal_lines;
CREATE TRIGGER trg_freeze_journal_lines
BEFORE UPDATE OR DELETE ON journal_lines
FOR EACH ROW EXECUTE FUNCTION fn_prevent_ledger_mutation();

-- 3. Idempotency Records Table for Webhooks and Financial Transactions
CREATE TABLE IF NOT EXISTS idempotency_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    idempotency_key VARCHAR(255) NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PROCESSING', -- 'PROCESSING', 'SUCCEEDED', 'FAILED'
    response_code INT,
    response_body JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (clock_timestamp() + INTERVAL '24 hours'),
    CONSTRAINT uq_tenant_idempotency_key UNIQUE (tenant_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_idempotency_tenant_key ON idempotency_records(tenant_id, idempotency_key);
CREATE INDEX IF NOT EXISTS idx_idempotency_expires ON idempotency_records(expires_at);

-- 4. High-Throughput Metering Tables
-- Partitioned Raw Events Table
CREATE TABLE IF NOT EXISTS usage_meter_events (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    meter_slug VARCHAR(64) NOT NULL,
    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    value NUMERIC(18, 6) NOT NULL DEFAULT 1.0,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Rollup Aggregation Table for fast billing evaluation
CREATE TABLE IF NOT EXISTS usage_meter_aggregations (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    meter_slug VARCHAR(64) NOT NULL,
    customer_id UUID NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    total_units NUMERIC(20, 6) NOT NULL DEFAULT 0,
    event_count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (tenant_id, meter_slug, customer_id, window_start)
);

CREATE INDEX IF NOT EXISTS idx_usage_meter_billing_lookup 
ON usage_meter_aggregations (tenant_id, customer_id, meter_slug, window_start, window_end);

-- Create active and next-month partitions for usage_meter_events
CREATE TABLE IF NOT EXISTS usage_meter_events_2026_09 PARTITION OF usage_meter_events
    FOR VALUES FROM ('2026-09-01 00:00:00+00') TO ('2026-10-01 00:00:00+00');

CREATE TABLE IF NOT EXISTS usage_meter_events_2026_10 PARTITION OF usage_meter_events
    FOR VALUES FROM ('2026-10-01 00:00:00+00') TO ('2026-11-01 00:00:00+00');

CREATE TABLE IF NOT EXISTS usage_meter_events_default PARTITION OF usage_meter_events DEFAULT;

-- 5. Custom Field Definitions per Tenant
CREATE TABLE IF NOT EXISTS custom_field_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    entity_name VARCHAR(64) NOT NULL, -- 'INVOICE', 'CUSTOMER', 'PRODUCT'
    field_key VARCHAR(64) NOT NULL,
    field_label VARCHAR(128) NOT NULL,
    data_type VARCHAR(32) NOT NULL,   -- 'STRING', 'NUMBER', 'BOOLEAN', 'DATE', 'ENUM'
    validation_regex TEXT DEFAULT '',
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    default_value JSONB,
    options JSONB DEFAULT '[]'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT uq_tenant_entity_field UNIQUE (tenant_id, entity_name, field_key)
);

CREATE INDEX IF NOT EXISTS idx_custom_fields_tenant_entity ON custom_field_definitions(tenant_id, entity_name);

-- 6. Developer API Keys Table
CREATE TABLE IF NOT EXISTS tenant_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_prefix VARCHAR(24) NOT NULL UNIQUE, -- e.g. "sk_live_a1b2c3d4"
    key_hash VARCHAR(64) NOT NULL,          -- Hex-encoded SHA-256
    scopes TEXT[] NOT NULL DEFAULT '{"billing:read", "billing:write"}',
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID REFERENCES users(id),
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_tenant_api_keys_lookup ON tenant_api_keys(key_prefix) WHERE revoked = FALSE;

-- 7. Security & Financial Audit Logs Table
CREATE TABLE IF NOT EXISTS security_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_id UUID,
    actor_type VARCHAR(32) NOT NULL DEFAULT 'USER', -- 'USER', 'API_KEY', 'SYSTEM'
    ip_address VARCHAR(45) DEFAULT '',
    action VARCHAR(64) NOT NULL,
    resource_id UUID,
    resource_type VARCHAR(64) NOT NULL,
    diff_payload JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_security_audit_tenant_time ON security_audit_logs(tenant_id, created_at DESC);
