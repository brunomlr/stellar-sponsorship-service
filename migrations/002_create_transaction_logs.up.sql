CREATE TYPE transaction_status AS ENUM ('signed', 'rejected');

CREATE TABLE transaction_logs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id        UUID NOT NULL REFERENCES api_keys(id),
    transaction_hash  VARCHAR(64),
    transaction_xdr   TEXT NOT NULL,
    operations        JSONB NOT NULL DEFAULT '[]',
    source_account    VARCHAR(56) NOT NULL,
    status            transaction_status NOT NULL,
    rejection_reason  VARCHAR(255),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transaction_logs_api_key_id ON transaction_logs (api_key_id);
CREATE INDEX idx_transaction_logs_status ON transaction_logs (status);
CREATE INDEX idx_transaction_logs_created_at ON transaction_logs (created_at);
