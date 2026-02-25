CREATE TYPE api_key_status AS ENUM ('pending_funding', 'active', 'revoked');

CREATE TABLE api_keys (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  VARCHAR(255) NOT NULL,
    key_hash              VARCHAR(64) NOT NULL UNIQUE,
    key_prefix            VARCHAR(20) NOT NULL,
    sponsor_account       VARCHAR(56) NOT NULL UNIQUE,
    xlm_budget            BIGINT NOT NULL,
    allowed_operations    JSONB NOT NULL DEFAULT '[]',
    allowed_source_accounts JSONB DEFAULT NULL,
    rate_limit_max        INTEGER NOT NULL DEFAULT 100,
    rate_limit_window     INTEGER NOT NULL DEFAULT 60,
    status                api_key_status NOT NULL DEFAULT 'pending_funding',
    funding_tx_xdr        TEXT,
    expires_at            TIMESTAMPTZ NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_key_hash ON api_keys (key_hash);
CREATE INDEX idx_api_keys_status ON api_keys (status);
CREATE INDEX idx_api_keys_sponsor_account ON api_keys (sponsor_account);
