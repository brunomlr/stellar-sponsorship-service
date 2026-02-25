package handler

import (
	"encoding/json"
	"net/http"

	"github.com/stellar-sponsorship-service/internal/middleware"
	"github.com/stellar-sponsorship-service/internal/service"
)

type SignHandler struct {
	service           *service.SigningService
	networkPassphrase string
}

func NewSignHandler(svc *service.SigningService, networkPassphrase string) *SignHandler {
	return &SignHandler{
		service:           svc,
		networkPassphrase: networkPassphrase,
	}
}

type SignRequest struct {
	TransactionXDR    string `json:"transaction_xdr"`
	NetworkPassphrase string `json:"network_passphrase"`
}

type SignResponse struct {
	SignedTransactionXDR  string `json:"signed_transaction_xdr"`
	SponsorPublicKey     string `json:"sponsor_public_key"`
	SponsorAccountBalance string `json:"sponsor_account_balance"`
}

func (h *SignHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	apiKey := middleware.GetAPIKey(r.Context())
	if apiKey == nil {
		RespondError(w, http.StatusUnauthorized, "invalid_api_key", "Missing API key")
		return
	}

	var req SignRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if req.TransactionXDR == "" {
		RespondError(w, http.StatusBadRequest, "invalid_request", "transaction_xdr is required")
		return
	}
	if req.NetworkPassphrase == "" {
		RespondError(w, http.StatusBadRequest, "invalid_request", "network_passphrase is required")
		return
	}
	if req.NetworkPassphrase != h.networkPassphrase {
		RespondError(w, http.StatusBadRequest, "invalid_network", "network_passphrase does not match the configured network")
		return
	}

	result, err := h.service.Sign(r.Context(), apiKey, req.TransactionXDR)
	if err != nil {
		service.RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, SignResponse{
		SignedTransactionXDR:  result.SignedXDR,
		SponsorPublicKey:     result.SponsorAccount,
		SponsorAccountBalance: result.SponsorBalance,
	})
}
