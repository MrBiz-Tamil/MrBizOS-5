-- 004_tenant_template_lock.sql
-- Adds template_id to tenants table so billing design template is locked by Super Admin per tenant company

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS template_id VARCHAR(50) NOT NULL DEFAULT '1';

COMMENT ON COLUMN tenants.template_id IS 'Billing invoice design template (1-10) assigned exclusively by Super Admin';
