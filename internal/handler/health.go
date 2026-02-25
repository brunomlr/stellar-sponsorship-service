package handler

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/stellar-sponsorship-service/internal/stellar"
	"github.com/stellar-sponsorship-service/internal/store"
)

type HealthHandler struct {
	store              store.APIKeyStore
	accounts           *stellar.AccountService
	masterPublicKey    string
	stellarNetwork     string
	startTime          time.Time
}

func NewHealthHandler(s store.APIKeyStore, accounts *stellar.AccountService, masterPublicKey, stellarNetwork string) *HealthHandler {
	return &HealthHandler{
		store:           s,
		accounts:        accounts,
		masterPublicKey: masterPublicKey,
		stellarNetwork:  stellarNetwork,
		startTime:       time.Now(),
	}
}

type HealthResponse struct {
	Status                string `json:"status"`
	Version               string `json:"version"`
	StellarNetwork        string `json:"stellar_network"`
	MasterPublicKey       string `json:"master_public_key"`
	MasterAccountBalance  string `json:"master_account_balance"`
	TotalSponsorAccounts  int    `json:"total_sponsor_accounts"`
	UptimeSeconds         int64  `json:"uptime_seconds"`
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	masterBalance, err := h.accounts.GetRawBalance(h.masterPublicKey)
	if err != nil {
		log.Error().Err(err).Msg("failed to get master account balance")
		masterBalance = "unknown"
	}

	// Count total sponsor accounts (all API keys)
	total, err := h.store.CountAPIKeys(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to count API keys")
		total = 0
	}

	RespondJSON(w, http.StatusOK, HealthResponse{
		Status:               "healthy",
		Version:              "1.0.0",
		StellarNetwork:       h.stellarNetwork,
		MasterPublicKey:      h.masterPublicKey,
		MasterAccountBalance: masterBalance,
		TotalSponsorAccounts: total,
		UptimeSeconds:        int64(time.Since(h.startTime).Seconds()),
	})
}
