package admin

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/stellar-sponsorship-service/internal/handler"
	"github.com/stellar-sponsorship-service/internal/service"
)

// --- Build Activate Transaction ---

type BuildActivateHandler struct {
	svc *service.FundingService
}

func NewBuildActivateHandler(svc *service.FundingService) *BuildActivateHandler {
	return &BuildActivateHandler{svc: svc}
}

type buildActivateResponse struct {
	SponsorAccount         string `json:"sponsor_account"`
	XLMBudget              string `json:"xlm_budget"`
	ActivateTransactionXDR string `json:"activate_transaction_xdr"`
}

func (h *BuildActivateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid API key ID")
		return
	}

	result, err := h.svc.BuildActivate(r.Context(), id)
	if err != nil {
		service.RespondError(w, err)
		return
	}

	handler.RespondJSON(w, http.StatusOK, buildActivateResponse{
		SponsorAccount:         result.SponsorAccount,
		XLMBudget:              result.XLMBudget,
		ActivateTransactionXDR: result.TransactionXDR,
	})
}

// --- Submit Activate Transaction ---

type SubmitActivateHandler struct {
	svc *service.FundingService
}

func NewSubmitActivateHandler(svc *service.FundingService) *SubmitActivateHandler {
	return &SubmitActivateHandler{svc: svc}
}

type submitActivateRequest struct {
	SignedTransactionXDR string `json:"signed_transaction_xdr"`
}

type submitActivateResponse struct {
	ID              uuid.UUID `json:"id"`
	Status          string    `json:"status"`
	SponsorAccount  string    `json:"sponsor_account"`
	TransactionHash string    `json:"transaction_hash"`
}

func (h *SubmitActivateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid API key ID")
		return
	}

	var req submitActivateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if req.SignedTransactionXDR == "" {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "signed_transaction_xdr is required")
		return
	}

	result, err := h.svc.SubmitActivate(r.Context(), id, req.SignedTransactionXDR)
	if err != nil {
		service.RespondError(w, err)
		return
	}

	handler.RespondJSON(w, http.StatusOK, submitActivateResponse{
		ID:              result.ID,
		Status:          result.Status,
		SponsorAccount:  result.SponsorAccount,
		TransactionHash: result.TransactionHash,
	})
}

// --- Build Fund Transaction ---

type BuildFundHandler struct {
	svc *service.FundingService
}

func NewBuildFundHandler(svc *service.FundingService) *BuildFundHandler {
	return &BuildFundHandler{svc: svc}
}

type buildFundRequest struct {
	Amount string `json:"amount"`
}

type buildFundResponse struct {
	SponsorAccount        string `json:"sponsor_account"`
	XLMToAdd              string `json:"xlm_to_add"`
	FundingTransactionXDR string `json:"funding_transaction_xdr"`
}

func (h *BuildFundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid API key ID")
		return
	}

	var req buildFundRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	result, err := h.svc.BuildFund(r.Context(), id, req.Amount)
	if err != nil {
		service.RespondError(w, err)
		return
	}

	handler.RespondJSON(w, http.StatusOK, buildFundResponse{
		SponsorAccount:        result.SponsorAccount,
		XLMToAdd:              result.XLMToAdd,
		FundingTransactionXDR: result.TransactionXDR,
	})
}

// --- Submit Fund Transaction ---

type SubmitFundHandler struct {
	svc *service.FundingService
}

func NewSubmitFundHandler(svc *service.FundingService) *SubmitFundHandler {
	return &SubmitFundHandler{svc: svc}
}

type submitFundRequest struct {
	SignedTransactionXDR string `json:"signed_transaction_xdr"`
}

type submitFundResponse struct {
	SponsorAccount  string `json:"sponsor_account"`
	XLMAdded        string `json:"xlm_added"`
	XLMAvailable    string `json:"xlm_available"`
	TransactionHash string `json:"transaction_hash"`
}

func (h *SubmitFundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid API key ID")
		return
	}

	var req submitFundRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if req.SignedTransactionXDR == "" {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "signed_transaction_xdr is required")
		return
	}

	result, err := h.svc.SubmitFund(r.Context(), id, req.SignedTransactionXDR)
	if err != nil {
		service.RespondError(w, err)
		return
	}

	handler.RespondJSON(w, http.StatusOK, submitFundResponse{
		SponsorAccount:  result.SponsorAccount,
		XLMAdded:        result.XLMAdded,
		XLMAvailable:    result.XLMAvailable,
		TransactionHash: result.TransactionHash,
	})
}
