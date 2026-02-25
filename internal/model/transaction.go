package model

import (
	"time"

	"github.com/google/uuid"
)

type TransactionStatus string

const (
	TxStatusSigned   TransactionStatus = "signed"
	TxStatusRejected TransactionStatus = "rejected"
)

type SubmissionStatus string

const (
	SubmissionConfirmed SubmissionStatus = "confirmed"
	SubmissionNotFound  SubmissionStatus = "not_found"
)

type TransactionLog struct {
	ID                  uuid.UUID         `json:"id"`
	APIKeyID            uuid.UUID         `json:"api_key_id"`
	TransactionHash     string            `json:"transaction_hash,omitempty"`
	TransactionXDR      string            `json:"transaction_xdr"`
	Operations          []string          `json:"operations"`
	SourceAccount       string            `json:"source_account"`
	Status              TransactionStatus `json:"status"`
	RejectionReason     string            `json:"rejection_reason,omitempty"`
	SubmissionStatus    *SubmissionStatus `json:"submission_status,omitempty"`
	SubmissionCheckedAt *time.Time        `json:"submission_checked_at,omitempty"`
	LedgerSequence      *int64            `json:"ledger_sequence,omitempty"`
	SubmittedAt         *time.Time        `json:"submitted_at,omitempty"`
	ReservesLocked      *int              `json:"reserves_locked,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
}
