package stellar

import (
	"fmt"
	"time"

	"github.com/stellar/go-stellar-sdk/clients/horizonclient"

	"github.com/stellar-sponsorship-service/internal/model"
)

type CheckResult struct {
	Status         model.SubmissionStatus
	LedgerSequence *int64
	SubmittedAt    *time.Time
}

type SubmissionChecker struct {
	horizonClient *horizonclient.Client
}

func NewSubmissionChecker(horizonClient *horizonclient.Client) *SubmissionChecker {
	return &SubmissionChecker{horizonClient: horizonClient}
}

func (c *SubmissionChecker) CheckTransaction(txHash string) (*CheckResult, error) {
	resp, err := c.horizonClient.TransactionDetail(txHash)
	if err != nil {
		if horizonclient.IsNotFoundError(err) {
			return &CheckResult{Status: model.SubmissionNotFound}, nil
		}
		return nil, fmt.Errorf("horizon transaction detail: %w", err)
	}

	ledger := int64(resp.Ledger)
	closedAt := resp.LedgerCloseTime

	return &CheckResult{
		Status:         model.SubmissionConfirmed,
		LedgerSequence: &ledger,
		SubmittedAt:    &closedAt,
	}, nil
}
