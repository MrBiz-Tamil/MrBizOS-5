-- 016_metering_event_dedup.sql
-- usage_meter_events is range-partitioned by created_at, so a unique constraint on
-- (tenant_id, idempotency_key) is impossible (unique indexes must include the partition key).
-- Idempotency of metering ingestion therefore uses a small side table: the first insert of a key wins,
-- retries are detected and not counted again. Rows older than the dedup window are purged by the app.

CREATE TABLE IF NOT EXISTS usage_meter_event_keys (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    idempotency_key VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (tenant_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_usage_meter_event_keys_created ON usage_meter_event_keys(created_at);

ALTER TABLE usage_meter_event_keys ENABLE ROW LEVEL SECURITY;
ALTER TABLE usage_meter_event_keys FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON usage_meter_event_keys;
CREATE POLICY tenant_isolation ON usage_meter_event_keys
    USING (
        NULLIF(current_setting('app.tenant_id', true), '') IS NULL
        OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
    )
    WITH CHECK (
        NULLIF(current_setting('app.tenant_id', true), '') IS NULL
        OR tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid
    );
