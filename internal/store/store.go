package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stellar-sponsorship-service/internal/model"
)

// APIKeyStore defines operations for API key management.
type APIKeyStore interface {
	CreateAPIKey(ctx context.Context, key *model.APIKey) error
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*model.APIKey, error)
	GetAPIKeyByID(ctx context.Context, id uuid.UUID) (*model.APIKey, error)
	ListAPIKeys(ctx context.Context, page, perPage int) ([]*model.APIKey, int, error)
	CountAPIKeys(ctx context.Context) (int, error)
	UpdateAPIKey(ctx context.Context, id uuid.UUID, updates APIKeyUpdates) error
	UpdateAPIKeyStatus(ctx context.Context, id uuid.UUID, status model.APIKeyStatus) error
	SetSponsorAccount(ctx context.Context, id uuid.UUID, sponsorAccount string) error
	RegenerateAPIKey(ctx context.Context, id uuid.UUID, keyHash, keyPrefix string) error
}

// TransactionLogStore defines operations for transaction log management.
type TransactionLogStore interface {
	CreateTransactionLog(ctx context.Context, log *model.TransactionLog) error
	ListTransactionLogs(ctx context.Context, filters TransactionFilters) ([]*model.TransactionLog, int, error)
	CountTransactionsByAPIKey(ctx context.Context, apiKeyID uuid.UUID) (int64, error)
	GetTransactionLogByID(ctx context.Context, id uuid.UUID) (*model.TransactionLog, error)
	UpdateSubmissionStatus(ctx context.Context, id uuid.UUID, status model.SubmissionStatus, ledgerSeq *int64, submittedAt *time.Time) error
}

// Store combines both APIKeyStore and TransactionLogStore.
type Store interface {
	APIKeyStore
	TransactionLogStore
}

type APIKeyUpdates struct {
	Name                  *string   `json:"name,omitempty"`
	AllowedOperations     []string  `json:"allowed_operations,omitempty"`
	AllowedSourceAccounts []string  `json:"allowed_source_accounts,omitempty"`
	RateLimitMax          *int      `json:"rate_limit_max,omitempty"`
	RateLimitWindow       *int      `json:"rate_limit_window,omitempty"`
	ExpiresAt             *time.Time `json:"expires_at,omitempty"`
}

type TransactionFilters struct {
	APIKeyID *uuid.UUID
	Status   *model.TransactionStatus
	From     *time.Time
	To       *time.Time
	Page     int
	PerPage  int
}
