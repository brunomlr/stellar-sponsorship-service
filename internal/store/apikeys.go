package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stellar-sponsorship-service/internal/model"
)

func (p *Postgres) CreateAPIKey(ctx context.Context, key *model.APIKey) error {
	ops, err := json.Marshal(key.AllowedOperations)
	if err != nil {
		return fmt.Errorf("marshal allowed_operations: %w", err)
	}

	var srcAccounts []byte
	if key.AllowedSourceAccounts != nil {
		srcAccounts, err = json.Marshal(key.AllowedSourceAccounts)
		if err != nil {
			return fmt.Errorf("marshal allowed_source_accounts: %w", err)
		}
	}

	// sponsor_account is nullable — pass nil when empty
	var sponsorAccount interface{}
	if key.SponsorAccount != "" {
		sponsorAccount = key.SponsorAccount
	}

	err = p.pool.QueryRow(ctx, `
		INSERT INTO api_keys (
			name, key_hash, key_prefix, sponsor_account, xlm_budget,
			allowed_operations, allowed_source_accounts,
			rate_limit_max, rate_limit_window,
			status, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`,
		key.Name, key.KeyHash, key.KeyPrefix, sponsorAccount, key.XLMBudget,
		ops, srcAccounts,
		key.RateLimitMax, key.RateLimitWindow,
		key.Status, key.ExpiresAt,
	).Scan(&key.ID, &key.CreatedAt, &key.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert api_key: %w", err)
	}
	return nil
}

const apiKeyColumns = `id, name, key_hash, key_prefix, sponsor_account, xlm_budget,
	allowed_operations, allowed_source_accounts,
	rate_limit_max, rate_limit_window, status,
	expires_at, created_at, updated_at`

func (p *Postgres) GetAPIKeyByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	return p.scanAPIKey(ctx, `SELECT `+apiKeyColumns+` FROM api_keys WHERE key_hash = $1`, keyHash)
}

func (p *Postgres) GetAPIKeyByID(ctx context.Context, id uuid.UUID) (*model.APIKey, error) {
	return p.scanAPIKey(ctx, `SELECT `+apiKeyColumns+` FROM api_keys WHERE id = $1`, id)
}

func (p *Postgres) ListAPIKeys(ctx context.Context, page, perPage int) ([]*model.APIKey, int, error) {
	var total int
	err := p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM api_keys`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count api_keys: %w", err)
	}

	offset := (page - 1) * perPage
	rows, err := p.pool.Query(ctx, `
		SELECT `+apiKeyColumns+` FROM api_keys ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list api_keys: %w", err)
	}
	defer rows.Close()

	var keys []*model.APIKey
	for rows.Next() {
		key, err := scanAPIKeyFromRow(rows)
		if err != nil {
			return nil, 0, err
		}
		keys = append(keys, key)
	}
	return keys, total, nil
}

func (p *Postgres) CountAPIKeys(ctx context.Context) (int, error) {
	var count int
	err := p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM api_keys`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count api_keys: %w", err)
	}
	return count, nil
}

func (p *Postgres) UpdateAPIKey(ctx context.Context, id uuid.UUID, updates APIKeyUpdates) error {
	// Build dynamic update query
	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if updates.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *updates.Name)
		argIdx++
	}
	if updates.AllowedOperations != nil {
		ops, err := json.Marshal(updates.AllowedOperations)
		if err != nil {
			return fmt.Errorf("marshal allowed_operations: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("allowed_operations = $%d", argIdx))
		args = append(args, ops)
		argIdx++
	}
	if updates.AllowedSourceAccounts != nil {
		src, err := json.Marshal(updates.AllowedSourceAccounts)
		if err != nil {
			return fmt.Errorf("marshal allowed_source_accounts: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("allowed_source_accounts = $%d", argIdx))
		args = append(args, src)
		argIdx++
	}
	if updates.RateLimitMax != nil {
		setClauses = append(setClauses, fmt.Sprintf("rate_limit_max = $%d", argIdx))
		args = append(args, *updates.RateLimitMax)
		argIdx++
	}
	if updates.RateLimitWindow != nil {
		setClauses = append(setClauses, fmt.Sprintf("rate_limit_window = $%d", argIdx))
		args = append(args, *updates.RateLimitWindow)
		argIdx++
	}
	if updates.ExpiresAt != nil {
		setClauses = append(setClauses, fmt.Sprintf("expires_at = $%d", argIdx))
		args = append(args, *updates.ExpiresAt)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE api_keys SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)

	tag, err := p.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update api_key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

func (p *Postgres) UpdateAPIKeyStatus(ctx context.Context, id uuid.UUID, status model.APIKeyStatus) error {
	tag, err := p.pool.Exec(ctx, `
		UPDATE api_keys SET status = $1, updated_at = NOW() WHERE id = $2
	`, status, id)
	if err != nil {
		return fmt.Errorf("update api_key status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

func (p *Postgres) scanAPIKey(ctx context.Context, query string, args ...interface{}) (*model.APIKey, error) {
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query api_key: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, pgx.ErrNoRows
	}
	return scanAPIKeyFromRow(rows)
}

func scanAPIKeyFromRow(rows pgx.Rows) (*model.APIKey, error) {
	var key model.APIKey
	var opsJSON, srcJSON []byte
	var sponsorAccount *string

	err := rows.Scan(
		&key.ID, &key.Name, &key.KeyHash, &key.KeyPrefix,
		&sponsorAccount, &key.XLMBudget,
		&opsJSON, &srcJSON,
		&key.RateLimitMax, &key.RateLimitWindow,
		&key.Status,
		&key.ExpiresAt, &key.CreatedAt, &key.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan api_key: %w", err)
	}

	if sponsorAccount != nil {
		key.SponsorAccount = *sponsorAccount
	}

	if err := json.Unmarshal(opsJSON, &key.AllowedOperations); err != nil {
		return nil, fmt.Errorf("unmarshal allowed_operations: %w", err)
	}
	if srcJSON != nil {
		if err := json.Unmarshal(srcJSON, &key.AllowedSourceAccounts); err != nil {
			return nil, fmt.Errorf("unmarshal allowed_source_accounts: %w", err)
		}
	}

	return &key, nil
}

func (p *Postgres) SetSponsorAccount(ctx context.Context, id uuid.UUID, sponsorAccount string) error {
	tag, err := p.pool.Exec(ctx, `
		UPDATE api_keys SET sponsor_account = $1, updated_at = NOW() WHERE id = $2
	`, sponsorAccount, id)
	if err != nil {
		return fmt.Errorf("set sponsor_account: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

func (p *Postgres) RegenerateAPIKey(ctx context.Context, id uuid.UUID, keyHash, keyPrefix string) error {
	tag, err := p.pool.Exec(ctx, `
		UPDATE api_keys SET key_hash = $1, key_prefix = $2, updated_at = NOW() WHERE id = $3
	`, keyHash, keyPrefix, id)
	if err != nil {
		return fmt.Errorf("regenerate api_key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

