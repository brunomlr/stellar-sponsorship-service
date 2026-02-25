package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stellar-sponsorship-service/internal/model"
)

func (p *Postgres) CreateTransactionLog(ctx context.Context, log *model.TransactionLog) error {
	opsJSON, err := json.Marshal(log.Operations)
	if err != nil {
		return fmt.Errorf("marshal operations: %w", err)
	}

	err = p.pool.QueryRow(ctx, `
		INSERT INTO transaction_logs (
			api_key_id, transaction_hash, transaction_xdr,
			operations, source_account, status, rejection_reason, reserves_locked
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`,
		log.APIKeyID, nullString(log.TransactionHash), log.TransactionXDR,
		opsJSON, log.SourceAccount, log.Status, nullString(log.RejectionReason), log.ReservesLocked,
	).Scan(&log.ID, &log.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert transaction_log: %w", err)
	}
	return nil
}

func (p *Postgres) ListTransactionLogs(ctx context.Context, filters TransactionFilters) ([]*model.TransactionLog, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if filters.APIKeyID != nil {
		where += fmt.Sprintf(" AND api_key_id = $%d", argIdx)
		args = append(args, *filters.APIKeyID)
		argIdx++
	}
	if filters.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filters.Status)
		argIdx++
	}
	if filters.From != nil {
		where += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *filters.From)
		argIdx++
	}
	if filters.To != nil {
		where += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *filters.To)
		argIdx++
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM transaction_logs %s", where)
	err := p.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count transaction_logs: %w", err)
	}

	page := filters.Page
	if page < 1 {
		page = 1
	}
	perPage := filters.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	args = append(args, perPage, offset)
	query := fmt.Sprintf(`
		SELECT id, api_key_id, transaction_hash, transaction_xdr,
		       operations, source_account, status, rejection_reason,
		       submission_status, submission_checked_at, ledger_sequence, submitted_at,
		       reserves_locked, created_at
		FROM transaction_logs %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list transaction_logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.TransactionLog
	for rows.Next() {
		var log model.TransactionLog
		var opsJSON []byte
		var txHash, rejReason *string

		err := rows.Scan(
			&log.ID, &log.APIKeyID, &txHash, &log.TransactionXDR,
			&opsJSON, &log.SourceAccount, &log.Status, &rejReason,
			&log.SubmissionStatus, &log.SubmissionCheckedAt, &log.LedgerSequence, &log.SubmittedAt,
			&log.ReservesLocked, &log.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan transaction_log: %w", err)
		}
		if txHash != nil {
			log.TransactionHash = *txHash
		}
		if rejReason != nil {
			log.RejectionReason = *rejReason
		}
		if err := json.Unmarshal(opsJSON, &log.Operations); err != nil {
			return nil, 0, fmt.Errorf("unmarshal operations: %w", err)
		}
		logs = append(logs, &log)
	}
	return logs, total, nil
}

func (p *Postgres) CountTransactionsByAPIKey(ctx context.Context, apiKeyID uuid.UUID) (int64, error) {
	var count int64
	err := p.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM transaction_logs WHERE api_key_id = $1 AND status = 'signed'
	`, apiKeyID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count transactions: %w", err)
	}
	return count, nil
}

func (p *Postgres) GetTransactionLogByID(ctx context.Context, id uuid.UUID) (*model.TransactionLog, error) {
	var log model.TransactionLog
	var opsJSON []byte
	var txHash, rejReason *string

	err := p.pool.QueryRow(ctx, `
		SELECT id, api_key_id, transaction_hash, transaction_xdr,
		       operations, source_account, status, rejection_reason,
		       submission_status, submission_checked_at, ledger_sequence, submitted_at,
		       reserves_locked, created_at
		FROM transaction_logs WHERE id = $1
	`, id).Scan(
		&log.ID, &log.APIKeyID, &txHash, &log.TransactionXDR,
		&opsJSON, &log.SourceAccount, &log.Status, &rejReason,
		&log.SubmissionStatus, &log.SubmissionCheckedAt, &log.LedgerSequence, &log.SubmittedAt,
		&log.ReservesLocked, &log.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get transaction_log: %w", err)
	}
	if txHash != nil {
		log.TransactionHash = *txHash
	}
	if rejReason != nil {
		log.RejectionReason = *rejReason
	}
	if err := json.Unmarshal(opsJSON, &log.Operations); err != nil {
		return nil, fmt.Errorf("unmarshal operations: %w", err)
	}
	return &log, nil
}

func (p *Postgres) UpdateSubmissionStatus(ctx context.Context, id uuid.UUID, status model.SubmissionStatus, ledgerSeq *int64, submittedAt *time.Time) error {
	_, err := p.pool.Exec(ctx, `
		UPDATE transaction_logs
		SET submission_status = $1,
		    submission_checked_at = NOW(),
		    ledger_sequence = $2,
		    submitted_at = $3
		WHERE id = $4
	`, status, ledgerSeq, submittedAt, id)
	if err != nil {
		return fmt.Errorf("update submission status: %w", err)
	}
	return nil
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
