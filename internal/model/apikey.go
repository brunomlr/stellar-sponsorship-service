package model

import (
	"time"

	"github.com/google/uuid"
)

type APIKeyStatus string

const (
	StatusPendingFunding APIKeyStatus = "pending_funding"
	StatusActive         APIKeyStatus = "active"
	StatusRevoked        APIKeyStatus = "revoked"
)

type APIKey struct {
	ID                    uuid.UUID    `json:"id"`
	Name                  string       `json:"name"`
	KeyHash               string       `json:"-"`
	KeyPrefix             string       `json:"key_prefix"`
	SponsorAccount        string       `json:"sponsor_account"`
	XLMBudget             int64        `json:"xlm_budget"`
	AllowedOperations     []string     `json:"allowed_operations"`
	AllowedSourceAccounts []string     `json:"allowed_source_accounts,omitempty"`
	RateLimitMax          int          `json:"rate_limit_max"`
	RateLimitWindow       int          `json:"rate_limit_window"`
	Status                APIKeyStatus `json:"status"`
	ExpiresAt             time.Time    `json:"expires_at"`
	CreatedAt             time.Time    `json:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at"`
}
