package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stellar/go-stellar-sdk/amount"

	"github.com/stellar-sponsorship-service/internal/handler"
	"github.com/stellar-sponsorship-service/internal/httputil"
	"github.com/stellar-sponsorship-service/internal/model"
	"github.com/stellar-sponsorship-service/internal/service"
	"github.com/stellar-sponsorship-service/internal/stellar"
	"github.com/stellar-sponsorship-service/internal/store"
)

// --- List API Keys ---

type ListAPIKeysHandler struct {
	store    store.APIKeyStore
	accounts *stellar.AccountService
}

func NewListAPIKeysHandler(s store.APIKeyStore, accounts *stellar.AccountService) *ListAPIKeysHandler {
	return &ListAPIKeysHandler{store: s, accounts: accounts}
}

type listAPIKeysResponse struct {
	APIKeys []apiKeyListItem `json:"api_keys"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	PerPage int              `json:"per_page"`
}

type apiKeyListItem struct {
	ID                    uuid.UUID `json:"id"`
	Name                  string    `json:"name"`
	KeyPrefix             string    `json:"key_prefix"`
	SponsorAccount        string    `json:"sponsor_account"`
	XLMBudget             string    `json:"xlm_budget"`
	XLMAvailable          string    `json:"xlm_available"`
	AllowedOperations     []string  `json:"allowed_operations"`
	AllowedSourceAccounts []string  `json:"allowed_source_accounts,omitempty"`
	RateLimitMax          int       `json:"rate_limit_max"`
	RateLimitWindow       int       `json:"rate_limit_window"`
	ExpiresAt             string    `json:"expires_at"`
	Status                string    `json:"status"`
	CreatedAt             string    `json:"created_at"`
}

func (h *ListAPIKeysHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	page, perPage, err := httputil.ParsePagination(r.URL.Query().Get("page"), r.URL.Query().Get("per_page"))
	if err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	keys, total, err := h.store.ListAPIKeys(r.Context(), page, perPage)
	if err != nil {
		log.Error().Err(err).Msg("failed to list API keys")
		handler.RespondError(w, http.StatusInternalServerError, "internal_error", "Failed to list API keys")
		return
	}

	items := make([]apiKeyListItem, 0, len(keys))
	for _, key := range keys {
		available := "0.0000000"
		if key.Status == model.StatusActive {
			avail, _, err := h.accounts.GetBalance(key.SponsorAccount)
			if err != nil {
				log.Error().Err(err).Str("sponsor", key.SponsorAccount).Msg("failed to get balance")
			} else {
				available = avail
			}
		}

		items = append(items, toAPIKeyListItem(key, available))
	}

	handler.RespondJSON(w, http.StatusOK, listAPIKeysResponse{
		APIKeys: items,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	})
}

// --- Get API Key ---

type GetAPIKeyHandler struct {
	store    store.APIKeyStore
	accounts *stellar.AccountService
}

func NewGetAPIKeyHandler(s store.APIKeyStore, accounts *stellar.AccountService) *GetAPIKeyHandler {
	return &GetAPIKeyHandler{store: s, accounts: accounts}
}

func (h *GetAPIKeyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid API key ID")
		return
	}

	key, err := h.store.GetAPIKeyByID(r.Context(), id)
	if err != nil {
		handler.RespondError(w, http.StatusNotFound, "not_found", "API key not found")
		return
	}

	available := "0.0000000"
	if key.Status == model.StatusActive {
		avail, _, err := h.accounts.GetBalance(key.SponsorAccount)
		if err != nil {
			log.Error().Err(err).Str("sponsor", key.SponsorAccount).Msg("failed to get balance")
		} else {
			available = avail
		}
	}

	handler.RespondJSON(w, http.StatusOK, toAPIKeyListItem(key, available))
}

// --- Create API Key ---

type CreateAPIKeyHandler struct {
	svc *service.APIKeyService
}

func NewCreateAPIKeyHandler(svc *service.APIKeyService) *CreateAPIKeyHandler {
	return &CreateAPIKeyHandler{svc: svc}
}

type createAPIKeyRequest struct {
	Name                  string         `json:"name"`
	XLMBudget             string         `json:"xlm_budget"`
	AllowedOperations     []string       `json:"allowed_operations"`
	ExpiresAt             time.Time      `json:"expires_at"`
	RateLimit             *rateLimitJSON `json:"rate_limit,omitempty"`
	AllowedSourceAccounts []string       `json:"allowed_source_accounts,omitempty"`
}

type rateLimitJSON struct {
	MaxRequests   int `json:"max_requests"`
	WindowSeconds int `json:"window_seconds"`
}

type createAPIKeyResponse struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	APIKey            string    `json:"api_key"`
	XLMBudget         string    `json:"xlm_budget"`
	AllowedOperations []string  `json:"allowed_operations"`
	ExpiresAt         string    `json:"expires_at"`
	Status            string    `json:"status"`
	CreatedAt         string    `json:"created_at"`
}

func (h *CreateAPIKeyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req createAPIKeyRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	input := service.CreateAPIKeyInput{
		Name:                  req.Name,
		XLMBudget:             req.XLMBudget,
		AllowedOperations:     req.AllowedOperations,
		AllowedSourceAccounts: req.AllowedSourceAccounts,
		ExpiresAt:             req.ExpiresAt,
	}
	if req.RateLimit != nil {
		input.RateLimitMax = &req.RateLimit.MaxRequests
		input.RateLimitWindow = &req.RateLimit.WindowSeconds
	}

	result, err := h.svc.Create(r.Context(), input)
	if err != nil {
		service.RespondError(w, err)
		return
	}

	handler.RespondJSON(w, http.StatusCreated, createAPIKeyResponse{
		ID:                result.APIKey.ID,
		Name:              result.APIKey.Name,
		APIKey:            result.RawKey,
		XLMBudget:         amount.StringFromInt64(result.APIKey.XLMBudget),
		AllowedOperations: result.APIKey.AllowedOperations,
		ExpiresAt:         result.APIKey.ExpiresAt.Format(time.RFC3339),
		Status:            string(result.APIKey.Status),
		CreatedAt:         result.APIKey.CreatedAt.Format(time.RFC3339),
	})
}

// --- Update API Key ---

type UpdateAPIKeyHandler struct {
	svc *service.APIKeyService
}

func NewUpdateAPIKeyHandler(svc *service.APIKeyService) *UpdateAPIKeyHandler {
	return &UpdateAPIKeyHandler{svc: svc}
}

func (h *UpdateAPIKeyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid API key ID")
		return
	}

	var updates store.APIKeyUpdates
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&updates); err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	apiKey, err := h.svc.Update(r.Context(), id, updates)
	if err != nil {
		service.RespondError(w, err)
		return
	}

	handler.RespondJSON(w, http.StatusOK, apiKey)
}

// --- Revoke API Key ---

type RevokeAPIKeyHandler struct {
	svc *service.APIKeyService
}

func NewRevokeAPIKeyHandler(svc *service.APIKeyService) *RevokeAPIKeyHandler {
	return &RevokeAPIKeyHandler{svc: svc}
}

func (h *RevokeAPIKeyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid API key ID")
		return
	}

	if err := h.svc.Revoke(r.Context(), id); err != nil {
		service.RespondError(w, err)
		return
	}

	handler.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"id":     id,
		"status": "revoked",
	})
}

// --- Regenerate API Key ---

type RegenerateAPIKeyHandler struct {
	svc *service.APIKeyService
}

func NewRegenerateAPIKeyHandler(svc *service.APIKeyService) *RegenerateAPIKeyHandler {
	return &RegenerateAPIKeyHandler{svc: svc}
}

type regenerateAPIKeyResponse struct {
	ID        uuid.UUID `json:"id"`
	APIKey    string    `json:"api_key"`
	KeyPrefix string    `json:"key_prefix"`
}

func (h *RegenerateAPIKeyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid API key ID")
		return
	}

	result, err := h.svc.Regenerate(r.Context(), id)
	if err != nil {
		service.RespondError(w, err)
		return
	}

	handler.RespondJSON(w, http.StatusOK, regenerateAPIKeyResponse{
		ID:        id,
		APIKey:    result.RawKey,
		KeyPrefix: result.KeyPrefix,
	})
}

// --- Helpers ---

func toAPIKeyListItem(key *model.APIKey, available string) apiKeyListItem {
	return apiKeyListItem{
		ID:                    key.ID,
		Name:                  key.Name,
		KeyPrefix:             key.KeyPrefix,
		SponsorAccount:        key.SponsorAccount,
		XLMBudget:             amount.StringFromInt64(key.XLMBudget),
		XLMAvailable:          available,
		AllowedOperations:     key.AllowedOperations,
		AllowedSourceAccounts: key.AllowedSourceAccounts,
		RateLimitMax:          key.RateLimitMax,
		RateLimitWindow:       key.RateLimitWindow,
		ExpiresAt:             key.ExpiresAt.Format(time.RFC3339),
		Status:                string(key.Status),
		CreatedAt:             key.CreatedAt.Format(time.RFC3339),
	}
}
