package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/stellar-sponsorship-service/internal/handler"
	"github.com/stellar-sponsorship-service/internal/service"
)

type SweepHandler struct {
	svc *service.FundingService
}

func NewSweepHandler(svc *service.FundingService) *SweepHandler {
	return &SweepHandler{svc: svc}
}

type sweepResponse struct {
	SponsorAccount     string `json:"sponsor_account"`
	XLMSwept           string `json:"xlm_swept"`
	XLMRemainingLocked string `json:"xlm_remaining_locked"`
	Destination        string `json:"destination"`
	TransactionHash    string `json:"transaction_hash"`
}

func (h *SweepHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid API key ID")
		return
	}

	result, err := h.svc.Sweep(r.Context(), id)
	if err != nil {
		service.RespondError(w, err)
		return
	}

	handler.RespondJSON(w, http.StatusOK, sweepResponse{
		SponsorAccount:     result.SponsorAccount,
		XLMSwept:           result.XLMSwept,
		XLMRemainingLocked: result.XLMRemainingLocked,
		Destination:        result.Destination,
		TransactionHash:    result.TransactionHash,
	})
}
