package handler

import (
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/stellar/go-stellar-sdk/amount"

	"github.com/stellar-sponsorship-service/internal/middleware"
	"github.com/stellar-sponsorship-service/internal/stellar"
	"github.com/stellar-sponsorship-service/internal/store"
)

type UsageHandler struct {
	store       store.TransactionLogStore
	accounts    *stellar.AccountService
	rateLimiter *middleware.RateLimiter
}

func NewUsageHandler(s store.TransactionLogStore, accounts *stellar.AccountService, rl *middleware.RateLimiter) *UsageHandler {
	return &UsageHandler{store: s, accounts: accounts, rateLimiter: rl}
}

type UsageResponse struct {
	APIKeyName          string   `json:"api_key_name"`
	SponsorAccount      string   `json:"sponsor_account"`
	XLMBudget           string   `json:"xlm_budget"`
	XLMAvailable        string   `json:"xlm_available"`
	XLMLockedInReserves string   `json:"xlm_locked_in_reserves"`
	AllowedOperations   []string `json:"allowed_operations"`
	ExpiresAt           string   `json:"expires_at"`
	IsActive            bool     `json:"is_active"`
	TransactionsSigned  int64    `json:"transactions_signed"`
	RateLimit           RateLimitInfo `json:"rate_limit"`
}

type RateLimitInfo struct {
	MaxRequests   int `json:"max_requests"`
	WindowSeconds int `json:"window_seconds"`
	Remaining     int `json:"remaining"`
}

func (h *UsageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	apiKey := middleware.GetAPIKey(r.Context())
	if apiKey == nil {
		RespondError(w, http.StatusUnauthorized, "invalid_api_key", "Missing API key")
		return
	}

	// Get on-chain balance
	available, locked, err := h.accounts.GetBalance(apiKey.SponsorAccount)
	if err != nil {
		log.Error().Err(err).Str("sponsor", apiKey.SponsorAccount).Msg("failed to get sponsor balance")
		RespondError(w, http.StatusInternalServerError, "balance_error", "Failed to retrieve balance")
		return
	}

	// Count signed transactions
	txCount, err := h.store.CountTransactionsByAPIKey(r.Context(), apiKey.ID)
	if err != nil {
		log.Error().Err(err).Msg("failed to count transactions")
		txCount = 0
	}

	// Get rate limit remaining (read-only, does not consume a request)
	remaining := h.rateLimiter.Remaining(apiKey)

	RespondJSON(w, http.StatusOK, UsageResponse{
		APIKeyName:          apiKey.Name,
		SponsorAccount:      apiKey.SponsorAccount,
		XLMBudget:           amount.StringFromInt64(apiKey.XLMBudget),
		XLMAvailable:        available,
		XLMLockedInReserves: locked,
		AllowedOperations:   apiKey.AllowedOperations,
		ExpiresAt:           apiKey.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		IsActive:            apiKey.Status == "active",
		TransactionsSigned:  txCount,
		RateLimit: RateLimitInfo{
			MaxRequests:   apiKey.RateLimitMax,
			WindowSeconds: apiKey.RateLimitWindow,
			Remaining:     remaining,
		},
	})
}
