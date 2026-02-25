package admin

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/stellar-sponsorship-service/internal/handler"
	"github.com/stellar-sponsorship-service/internal/httputil"
	"github.com/stellar-sponsorship-service/internal/model"
	"github.com/stellar-sponsorship-service/internal/stellar"
	"github.com/stellar-sponsorship-service/internal/store"
)

// --- List Transactions ---

type TransactionsHandler struct {
	store   store.TransactionLogStore
	checker *stellar.SubmissionChecker
}

func NewTransactionsHandler(s store.TransactionLogStore, checker *stellar.SubmissionChecker) *TransactionsHandler {
	return &TransactionsHandler{store: s, checker: checker}
}

type transactionsResponse struct {
	Transactions []transactionItem `json:"transactions"`
	Total        int               `json:"total"`
	Page         int               `json:"page"`
	PerPage      int               `json:"per_page"`
}

type transactionItem struct {
	ID                  uuid.UUID `json:"id"`
	APIKeyID            uuid.UUID `json:"api_key_id"`
	TransactionHash     string    `json:"transaction_hash,omitempty"`
	Operations          []string  `json:"operations"`
	SourceAccount       string    `json:"source_account"`
	Status              string    `json:"status"`
	RejectionReason     string    `json:"rejection_reason,omitempty"`
	SubmissionStatus    *string   `json:"submission_status"`
	SubmissionCheckedAt *string   `json:"submission_checked_at,omitempty"`
	LedgerSequence      *int64    `json:"ledger_sequence,omitempty"`
	SubmittedAt         *string   `json:"submitted_at,omitempty"`
	ReservesLocked      *int      `json:"reserves_locked,omitempty"`
	CreatedAt           string    `json:"created_at"`
}

const (
	maxAutoCheckAge   = 24 * time.Hour
	maxConcurrentChecks = 5
)

func (h *TransactionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, perPage, err := httputil.ParsePagination(q.Get("page"), q.Get("per_page"))
	if err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	filters := store.TransactionFilters{
		Page:    page,
		PerPage: perPage,
	}

	if apiKeyIDStr := q.Get("api_key_id"); apiKeyIDStr != "" {
		id, err := uuid.Parse(apiKeyIDStr)
		if err != nil {
			handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid api_key_id")
			return
		}
		filters.APIKeyID = &id
	}

	if statusStr := q.Get("status"); statusStr != "" {
		status := model.TransactionStatus(statusStr)
		filters.Status = &status
	}

	if fromStr := q.Get("from"); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid 'from' date format (use RFC3339)")
			return
		}
		filters.From = &t
	}

	if toStr := q.Get("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid 'to' date format (use RFC3339)")
			return
		}
		filters.To = &t
	}

	logs, total, err := h.store.ListTransactionLogs(r.Context(), filters)
	if err != nil {
		log.Error().Err(err).Msg("failed to list transactions")
		handler.RespondError(w, http.StatusInternalServerError, "internal_error", "Failed to list transactions")
		return
	}

	// Auto-check unchecked signed transactions against Horizon
	h.autoCheckSubmissions(r.Context(), logs)

	items := make([]transactionItem, 0, len(logs))
	for _, l := range logs {
		item := transactionItem{
			ID:              l.ID,
			APIKeyID:        l.APIKeyID,
			TransactionHash: l.TransactionHash,
			Operations:      l.Operations,
			SourceAccount:   l.SourceAccount,
			Status:          string(l.Status),
			RejectionReason: l.RejectionReason,
			CreatedAt:       l.CreatedAt.Format(time.RFC3339),
		}
		if l.SubmissionStatus != nil {
			s := string(*l.SubmissionStatus)
			item.SubmissionStatus = &s
		}
		if l.SubmissionCheckedAt != nil {
			s := l.SubmissionCheckedAt.Format(time.RFC3339)
			item.SubmissionCheckedAt = &s
		}
		if l.LedgerSequence != nil {
			item.LedgerSequence = l.LedgerSequence
		}
		if l.SubmittedAt != nil {
			s := l.SubmittedAt.Format(time.RFC3339)
			item.SubmittedAt = &s
		}
		item.ReservesLocked = l.ReservesLocked
		items = append(items, item)
	}

	handler.RespondJSON(w, http.StatusOK, transactionsResponse{
		Transactions: items,
		Total:        total,
		Page:         page,
		PerPage:      perPage,
	})
}

// autoCheckSubmissions checks Horizon for signed transactions that haven't been checked yet.
// It mutates the log entries in-place with the results and caches them in the DB.
func (h *TransactionsHandler) autoCheckSubmissions(ctx context.Context, logs []*model.TransactionLog) {
	// Collect transactions that need checking
	var toCheck []*model.TransactionLog
	cutoff := time.Now().Add(-maxAutoCheckAge)
	for _, l := range logs {
		if l.Status != model.TxStatusSigned {
			continue
		}
		if l.TransactionHash == "" {
			continue
		}
		if l.SubmissionStatus != nil && *l.SubmissionStatus == model.SubmissionConfirmed {
			continue // already confirmed, no need to recheck
		}
		if l.SubmissionStatus != nil && l.SubmissionCheckedAt != nil && time.Since(*l.SubmissionCheckedAt) < 5*time.Minute {
			continue // checked recently, skip
		}
		if l.CreatedAt.Before(cutoff) && l.SubmissionStatus == nil {
			continue // too old and never checked
		}
		toCheck = append(toCheck, l)
	}

	if len(toCheck) == 0 {
		return
	}

	// Use a short timeout for the batch check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Check concurrently with bounded parallelism
	sem := make(chan struct{}, maxConcurrentChecks)
	var wg sync.WaitGroup

	for _, l := range toCheck {
		wg.Add(1)
		go func(txLog *model.TransactionLog) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-checkCtx.Done():
				return
			}

			result, err := h.checker.CheckTransaction(txLog.TransactionHash)
			if err != nil {
				log.Warn().Err(err).Str("tx_hash", txLog.TransactionHash).Msg("failed to check transaction submission")
				return
			}

			// Update in-memory log entry
			txLog.SubmissionStatus = &result.Status
			now := time.Now()
			txLog.SubmissionCheckedAt = &now
			txLog.LedgerSequence = result.LedgerSequence
			txLog.SubmittedAt = result.SubmittedAt

			// Cache in DB (best effort, don't fail the request)
			if err := h.store.UpdateSubmissionStatus(checkCtx, txLog.ID, result.Status, result.LedgerSequence, result.SubmittedAt); err != nil {
				log.Error().Err(err).Str("tx_hash", txLog.TransactionHash).Msg("failed to cache submission status")
			}
		}(l)
	}

	wg.Wait()
}

// --- Check Single Transaction ---

type CheckTransactionHandler struct {
	store   store.TransactionLogStore
	checker *stellar.SubmissionChecker
}

func NewCheckTransactionHandler(s store.TransactionLogStore, checker *stellar.SubmissionChecker) *CheckTransactionHandler {
	return &CheckTransactionHandler{store: s, checker: checker}
}

type checkTransactionResponse struct {
	ID               uuid.UUID `json:"id"`
	SubmissionStatus string    `json:"submission_status"`
	LedgerSequence   *int64    `json:"ledger_sequence,omitempty"`
	SubmittedAt      *string   `json:"submitted_at,omitempty"`
}

func (h *CheckTransactionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Invalid transaction ID")
		return
	}

	txLog, err := h.store.GetTransactionLogByID(r.Context(), id)
	if err != nil {
		log.Error().Err(err).Str("id", idStr).Msg("failed to get transaction")
		handler.RespondError(w, http.StatusNotFound, "not_found", "Transaction not found")
		return
	}

	if txLog.Status != model.TxStatusSigned || txLog.TransactionHash == "" {
		handler.RespondError(w, http.StatusBadRequest, "invalid_request", "Transaction was not signed or has no hash")
		return
	}

	result, err := h.checker.CheckTransaction(txLog.TransactionHash)
	if err != nil {
		log.Error().Err(err).Str("tx_hash", txLog.TransactionHash).Msg("failed to check transaction on Horizon")
		handler.RespondError(w, http.StatusBadGateway, "horizon_error", "Failed to check transaction on Horizon")
		return
	}

	// Cache in DB
	if err := h.store.UpdateSubmissionStatus(r.Context(), txLog.ID, result.Status, result.LedgerSequence, result.SubmittedAt); err != nil {
		log.Error().Err(err).Msg("failed to update submission status")
	}

	resp := checkTransactionResponse{
		ID:               txLog.ID,
		SubmissionStatus: string(result.Status),
		LedgerSequence:   result.LedgerSequence,
	}
	if result.SubmittedAt != nil {
		s := result.SubmittedAt.Format(time.RFC3339)
		resp.SubmittedAt = &s
	}

	handler.RespondJSON(w, http.StatusOK, resp)
}
