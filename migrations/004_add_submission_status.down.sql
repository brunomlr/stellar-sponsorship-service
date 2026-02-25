ALTER TABLE transaction_logs
    DROP COLUMN IF EXISTS submission_status,
    DROP COLUMN IF EXISTS submission_checked_at,
    DROP COLUMN IF EXISTS ledger_sequence,
    DROP COLUMN IF EXISTS submitted_at;

DROP TYPE IF EXISTS submission_status;
