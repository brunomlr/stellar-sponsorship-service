-- Add submission status tracking for signed transactions
CREATE TYPE submission_status AS ENUM ('confirmed', 'not_found');

ALTER TABLE transaction_logs
    ADD COLUMN submission_status submission_status,
    ADD COLUMN submission_checked_at TIMESTAMPTZ,
    ADD COLUMN ledger_sequence BIGINT,
    ADD COLUMN submitted_at TIMESTAMPTZ;

-- Index for efficiently finding signed transactions that need checking
CREATE INDEX idx_transaction_logs_needs_check
    ON transaction_logs (created_at DESC)
    WHERE status = 'signed' AND submission_status IS NULL;
