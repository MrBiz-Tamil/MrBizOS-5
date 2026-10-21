-- 019_account_security.sql
-- Account recovery and login safety: password reset, email verification, TOTP two-factor login.
--
-- All tokens are random 256-bit values; only their SHA-256 hash is stored, they expire, and they are
-- single-use. TOTP secrets are stored AES-256-GCM encrypted (key from APP_SECRET_KEY); recovery codes are
-- stored as SHA-256 hashes.

ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret_enc BYTEA;
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_last_step BIGINT NOT NULL DEFAULT 0;

-- Accounts that exist today were created by an administrator who vouched for the address.
UPDATE users SET email_verified_at = COALESCE(email_verified_at, NOW()) WHERE email_verified_at IS NULL;

CREATE TABLE IF NOT EXISTS auth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose VARCHAR(32) NOT NULL CHECK (purpose IN ('PASSWORD_RESET', 'EMAIL_VERIFY', 'TWO_FA_CHALLENGE')),
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    attempts INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    ip_address VARCHAR(45) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX IF NOT EXISTS idx_auth_tokens_user_purpose ON auth_tokens(user_id, purpose);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_expires ON auth_tokens(expires_at);

CREATE TABLE IF NOT EXISTS user_recovery_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(64) NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (user_id, code_hash)
);
