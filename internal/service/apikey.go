package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stellar/go-stellar-sdk/amount"

	"github.com/stellar-sponsorship-service/internal/middleware"
	"github.com/stellar-sponsorship-service/internal/model"
	"github.com/stellar-sponsorship-service/internal/store"
	"github.com/stellar-sponsorship-service/internal/validation"
)

const (
	defaultRateLimitMax    = 100
	defaultRateLimitWindow = 60
	maxRateLimitMax        = 10000
	maxRateLimitWindow     = 86400
)

// APIKeyService handles API key business logic.
type APIKeyService struct {
	store   store.APIKeyStore
	network string
}

// NewAPIKeyService creates a new API key service.
func NewAPIKeyService(store store.APIKeyStore, network string) *APIKeyService {
	return &APIKeyService{store: store, network: network}
}

// CreateAPIKeyInput contains the parameters for creating a new API key.
type CreateAPIKeyInput struct {
	Name                  string
	XLMBudget             string
	AllowedOperations     []string
	AllowedSourceAccounts []string
	ExpiresAt             time.Time
	RateLimitMax          *int
	RateLimitWindow       *int
}

// CreateAPIKeyResult contains the output of a successful key creation.
type CreateAPIKeyResult struct {
	APIKey *model.APIKey
	RawKey string
}

// Create validates input, generates a new API key, and persists it.
func (s *APIKeyService) Create(ctx context.Context, input CreateAPIKeyInput) (*CreateAPIKeyResult, error) {
	// Validate input
	if input.Name == "" {
		return nil, NewBadRequest("invalid_request", "name is required")
	}
	if input.XLMBudget == "" {
		return nil, NewBadRequest("invalid_request", "xlm_budget is required")
	}
	if len(input.AllowedOperations) == 0 {
		return nil, NewBadRequest("invalid_request", "allowed_operations is required")
	}
	if err := validation.AllowedOperations(input.AllowedOperations); err != nil {
		return nil, NewBadRequest("invalid_request", err.Error())
	}
	if input.ExpiresAt.IsZero() {
		return nil, NewBadRequest("invalid_request", "expires_at is required")
	}
	if !input.ExpiresAt.After(time.Now().UTC()) {
		return nil, NewBadRequest("invalid_request", "expires_at must be in the future")
	}
	if err := validation.SourceAccounts(input.AllowedSourceAccounts); err != nil {
		return nil, NewBadRequest("invalid_request", err.Error())
	}

	budgetStroops, err := amount.ParseInt64(input.XLMBudget)
	if err != nil {
		return nil, NewBadRequest("invalid_request", "Invalid xlm_budget format")
	}
	if budgetStroops <= 0 {
		return nil, NewBadRequest("invalid_request", "xlm_budget must be positive")
	}

	// Normalize rate limit
	rateLimitMax, rateLimitWindow, err := normalizeRateLimit(input.RateLimitMax, input.RateLimitWindow)
	if err != nil {
		return nil, NewBadRequest("invalid_request", err.Error())
	}

	// Generate API key
	rawKey, err := generateAPIKey(s.network)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate API key")
		return nil, NewInternal("internal_error", "Failed to create API key")
	}
	keyHash := middleware.SHA256Hex(rawKey)
	keyPrefix := rawKey[:16] + "..."

	apiKey := &model.APIKey{
		Name:                  input.Name,
		KeyHash:               keyHash,
		KeyPrefix:             keyPrefix,
		XLMBudget:             budgetStroops,
		AllowedOperations:     input.AllowedOperations,
		AllowedSourceAccounts: input.AllowedSourceAccounts,
		RateLimitMax:          rateLimitMax,
		RateLimitWindow:       rateLimitWindow,
		Status:                model.StatusPendingFunding,
		ExpiresAt:             input.ExpiresAt,
	}

	if err := s.store.CreateAPIKey(ctx, apiKey); err != nil {
		log.Error().Err(err).Msg("failed to create API key")
		return nil, NewInternal("internal_error", "Failed to create API key")
	}

	return &CreateAPIKeyResult{APIKey: apiKey, RawKey: rawKey}, nil
}

// Update validates and applies partial updates to an existing API key.
func (s *APIKeyService) Update(ctx context.Context, id uuid.UUID, updates store.APIKeyUpdates) (*model.APIKey, error) {
	// Validate updates
	if updates.Name != nil && strings.TrimSpace(*updates.Name) == "" {
		return nil, NewBadRequest("invalid_request", "name cannot be empty")
	}
	if updates.AllowedOperations != nil {
		if err := validation.AllowedOperations(updates.AllowedOperations); err != nil {
			return nil, NewBadRequest("invalid_request", err.Error())
		}
	}
	if updates.AllowedSourceAccounts != nil {
		if err := validation.SourceAccounts(updates.AllowedSourceAccounts); err != nil {
			return nil, NewBadRequest("invalid_request", err.Error())
		}
	}
	if updates.RateLimitMax != nil {
		if *updates.RateLimitMax < 1 || *updates.RateLimitMax > maxRateLimitMax {
			return nil, NewBadRequest("invalid_request", "rate_limit_max must be between 1 and 10000")
		}
	}
	if updates.RateLimitWindow != nil {
		if *updates.RateLimitWindow < 1 || *updates.RateLimitWindow > maxRateLimitWindow {
			return nil, NewBadRequest("invalid_request", "rate_limit_window must be between 1 and 86400")
		}
	}
	if updates.ExpiresAt != nil && !updates.ExpiresAt.After(time.Now().UTC()) {
		return nil, NewBadRequest("invalid_request", "expires_at must be in the future")
	}

	if err := s.store.UpdateAPIKey(ctx, id, updates); err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to update API key")
		return nil, NewInternal("internal_error", "Failed to update API key")
	}

	apiKey, err := s.store.GetAPIKeyByID(ctx, id)
	if err != nil {
		return nil, NewNotFound("not_found", "API key not found")
	}

	return apiKey, nil
}

// Revoke marks an API key as revoked.
func (s *APIKeyService) Revoke(ctx context.Context, id uuid.UUID) error {
	apiKey, err := s.store.GetAPIKeyByID(ctx, id)
	if err != nil {
		return NewNotFound("not_found", "API key not found")
	}

	if apiKey.Status == model.StatusRevoked {
		return NewBadRequest("invalid_status", "API key is already revoked")
	}

	if err := s.store.UpdateAPIKeyStatus(ctx, id, model.StatusRevoked); err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to revoke API key")
		return NewInternal("internal_error", "Failed to revoke API key")
	}

	return nil
}

// RegenerateResult contains the output of a successful key regeneration.
type RegenerateResult struct {
	RawKey    string
	KeyPrefix string
}

// Regenerate generates a new API key for an existing key ID.
func (s *APIKeyService) Regenerate(ctx context.Context, id uuid.UUID) (*RegenerateResult, error) {
	apiKey, err := s.store.GetAPIKeyByID(ctx, id)
	if err != nil {
		return nil, NewNotFound("not_found", "API key not found")
	}

	if apiKey.Status == model.StatusRevoked {
		return nil, NewBadRequest("invalid_status", "Cannot regenerate a revoked API key")
	}

	rawKey, err := generateAPIKey(s.network)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate API key")
		return nil, NewInternal("internal_error", "Failed to regenerate API key")
	}
	keyHash := middleware.SHA256Hex(rawKey)
	keyPrefix := rawKey[:16] + "..."

	if err := s.store.RegenerateAPIKey(ctx, id, keyHash, keyPrefix); err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to regenerate API key")
		return nil, NewInternal("internal_error", "Failed to regenerate API key")
	}

	return &RegenerateResult{RawKey: rawKey, KeyPrefix: keyPrefix}, nil
}

func generateAPIKey(network string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand failed: %w", err)
	}
	prefix := "sk_live_"
	if network == "testnet" {
		prefix = "sk_test_"
	}
	return prefix + hex.EncodeToString(b), nil
}

func normalizeRateLimit(maxRequests, windowSeconds *int) (int, int, error) {
	rlMax := defaultRateLimitMax
	rlWindow := defaultRateLimitWindow

	if maxRequests != nil {
		if *maxRequests < 1 || *maxRequests > maxRateLimitMax {
			return 0, 0, fmt.Errorf("rate_limit.max_requests must be between 1 and 10000")
		}
		rlMax = *maxRequests
	}

	if windowSeconds != nil {
		if *windowSeconds < 1 || *windowSeconds > maxRateLimitWindow {
			return 0, 0, fmt.Errorf("rate_limit.window_seconds must be between 1 and 86400")
		}
		rlWindow = *windowSeconds
	}

	return rlMax, rlWindow, nil
}
