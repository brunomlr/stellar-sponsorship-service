package service

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/stellar/go-stellar-sdk/amount"

	"github.com/stellar-sponsorship-service/internal/model"
	"github.com/stellar-sponsorship-service/internal/stellar"
	"github.com/stellar-sponsorship-service/internal/store"
)

// SigningService handles the core transaction signing business logic.
type SigningService struct {
	store    store.TransactionLogStore
	signer   *stellar.Signer
	verifier *stellar.Verifier
	accounts *stellar.AccountService
}

// NewSigningService creates a new signing service.
func NewSigningService(
	store store.TransactionLogStore,
	signer *stellar.Signer,
	verifier *stellar.Verifier,
	accounts *stellar.AccountService,
) *SigningService {
	return &SigningService{
		store:    store,
		signer:   signer,
		verifier: verifier,
		accounts: accounts,
	}
}

// SignResult contains the output of a successful signing operation.
type SignResult struct {
	SignedXDR      string
	TxHash         string
	SponsorAccount string
	SponsorBalance string
}

// Sign verifies, balance-checks, signs, and logs a transaction.
func (s *SigningService) Sign(ctx context.Context, apiKey *model.APIKey, transactionXDR string) (*SignResult, error) {
	// 1. Verify transaction against API key rules
	result := s.verifier.Verify(transactionXDR, apiKey)
	if !result.Valid {
		// Log the rejection (best effort)
		if err := s.store.CreateTransactionLog(ctx, &model.TransactionLog{
			APIKeyID:        apiKey.ID,
			TransactionXDR:  transactionXDR,
			Operations:      result.Operations,
			SourceAccount:   result.SourceAccount,
			Status:          model.TxStatusRejected,
			RejectionReason: result.ErrorMessage,
		}); err != nil {
			log.Error().Err(err).Str("api_key_id", apiKey.ID.String()).Msg("failed to log rejected transaction")
		}

		return nil, NewBadRequest(result.ErrorCode, result.ErrorMessage)
	}

	// 2. Pre-sign balance check
	available, _, err := s.accounts.GetBalance(apiKey.SponsorAccount)
	if err != nil {
		log.Error().Err(err).Str("sponsor", apiKey.SponsorAccount).Msg("failed to get sponsor balance")
		return nil, NewUnavailable("balance_check_failed", "Unable to verify sponsor account balance")
	}

	requiredStroops := int64(result.ReservesLocked) * stellar.BaseReserveStroops
	availableStroops, err := amount.ParseInt64(available)
	if err != nil {
		log.Error().Err(err).Str("available", available).Msg("failed to parse available balance")
		return nil, NewInternal("balance_check_failed", "Unable to verify sponsor account balance")
	}

	if availableStroops < requiredStroops {
		return nil, NewBadRequest("insufficient_balance",
			"Sponsor account does not have enough available balance to cover the reserves required by this transaction")
	}

	// 3. Sign transaction
	signedXDR, txHash, err := s.signer.Sign(transactionXDR)
	if err != nil {
		log.Error().Err(err).Msg("failed to sign transaction")
		return nil, NewInternal("signing_failed", "Failed to sign transaction")
	}

	// 4. Log signed transaction (best effort)
	reserves := result.ReservesLocked
	if err := s.store.CreateTransactionLog(ctx, &model.TransactionLog{
		APIKeyID:        apiKey.ID,
		TransactionHash: txHash,
		TransactionXDR:  signedXDR,
		Operations:      result.Operations,
		SourceAccount:   result.SourceAccount,
		Status:          model.TxStatusSigned,
		ReservesLocked:  &reserves,
	}); err != nil {
		log.Error().Err(err).Str("api_key_id", apiKey.ID.String()).Msg("failed to log signed transaction")
	}

	return &SignResult{
		SignedXDR:      signedXDR,
		TxHash:         txHash,
		SponsorAccount: apiKey.SponsorAccount,
		SponsorBalance: available,
	}, nil
}
