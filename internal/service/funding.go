package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stellar/go-stellar-sdk/amount"
	"github.com/stellar/go-stellar-sdk/clients/horizonclient"
	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/txnbuild"

	"github.com/stellar-sponsorship-service/internal/model"
	"github.com/stellar-sponsorship-service/internal/stellar"
	"github.com/stellar-sponsorship-service/internal/store"
)

// FundingService handles activation, funding, and sweep operations.
type FundingService struct {
	store             store.APIKeyStore
	builder           *stellar.Builder
	signer            *stellar.Signer
	accounts          *stellar.AccountService
	horizonClient     *horizonclient.Client
	masterPublicKey   string
	networkPassphrase string
}

// NewFundingService creates a new funding service.
func NewFundingService(
	store store.APIKeyStore,
	builder *stellar.Builder,
	signer *stellar.Signer,
	accounts *stellar.AccountService,
	horizonClient *horizonclient.Client,
	masterPublicKey string,
	networkPassphrase string,
) *FundingService {
	return &FundingService{
		store:             store,
		builder:           builder,
		signer:            signer,
		accounts:          accounts,
		horizonClient:     horizonClient,
		masterPublicKey:   masterPublicKey,
		networkPassphrase: networkPassphrase,
	}
}

// BuildActivateResult contains the output of building an activation transaction.
type BuildActivateResult struct {
	SponsorAccount string
	XLMBudget      string
	TransactionXDR string
}

// BuildActivate generates an ephemeral keypair and builds an activation transaction.
func (s *FundingService) BuildActivate(ctx context.Context, id uuid.UUID) (*BuildActivateResult, error) {
	apiKey, err := s.store.GetAPIKeyByID(ctx, id)
	if err != nil {
		return nil, NewNotFound("not_found", "API key not found")
	}

	if apiKey.Status != model.StatusPendingFunding {
		return nil, NewBadRequest("invalid_status", "API key is not pending funding")
	}

	sponsorKP, err := keypair.Random()
	if err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to generate sponsor keypair")
		return nil, NewInternal("internal_error", "Failed to generate sponsor keypair")
	}

	presignedXDR, err := s.builder.BuildCreateSponsorAccount(sponsorKP, apiKey.XLMBudget)
	if err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to build activate transaction")
		return nil, NewInternal("internal_error", "Failed to build activation transaction")
	}

	return &BuildActivateResult{
		SponsorAccount: sponsorKP.Address(),
		XLMBudget:      amount.StringFromInt64(apiKey.XLMBudget),
		TransactionXDR: presignedXDR,
	}, nil
}

// SubmitActivateResult contains the output of a submitted activation transaction.
type SubmitActivateResult struct {
	ID              uuid.UUID
	Status          string
	SponsorAccount  string
	TransactionHash string
}

// SubmitActivate validates and submits a signed activation transaction to Stellar.
func (s *FundingService) SubmitActivate(ctx context.Context, id uuid.UUID, signedXDR string) (*SubmitActivateResult, error) {
	apiKey, err := s.store.GetAPIKeyByID(ctx, id)
	if err != nil {
		return nil, NewNotFound("not_found", "API key not found")
	}

	if apiKey.Status != model.StatusPendingFunding {
		return nil, NewBadRequest("invalid_status", "API key is not pending funding")
	}

	sponsorAccount, err := validateActivateTransaction(signedXDR, s.networkPassphrase, s.masterPublicKey)
	if err != nil {
		return nil, NewBadRequest("invalid_request", err.Error())
	}

	resp, err := s.horizonClient.SubmitTransactionXDR(signedXDR)
	if err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to submit activation transaction")
		return nil, NewBadRequest("submission_failed", "Failed to submit transaction to Stellar: "+err.Error())
	}

	if err := s.store.SetSponsorAccount(ctx, id, sponsorAccount); err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to save sponsor account")
		return nil, NewInternal("internal_error", "Failed to save sponsor account")
	}

	if err := s.store.UpdateAPIKeyStatus(ctx, id, model.StatusActive); err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to activate API key")
		return nil, NewInternal("internal_error", "Failed to activate API key")
	}

	return &SubmitActivateResult{
		ID:              apiKey.ID,
		Status:          "active",
		SponsorAccount:  sponsorAccount,
		TransactionHash: resp.Hash,
	}, nil
}

// BuildFundResult contains the output of building a fund transaction.
type BuildFundResult struct {
	SponsorAccount string
	XLMToAdd       string
	TransactionXDR string
}

// BuildFund builds an unsigned fund transaction for a sponsor account.
func (s *FundingService) BuildFund(ctx context.Context, id uuid.UUID, amountXLM string) (*BuildFundResult, error) {
	fundStroops, err := amount.ParseInt64(amountXLM)
	if err != nil || fundStroops <= 0 {
		return nil, NewBadRequest("invalid_request", "Invalid amount")
	}

	apiKey, err := s.store.GetAPIKeyByID(ctx, id)
	if err != nil {
		return nil, NewNotFound("not_found", "API key not found")
	}

	if apiKey.Status != model.StatusActive {
		return nil, NewBadRequest("invalid_status", "API key must be active to fund")
	}

	unsignedXDR, err := s.builder.BuildFundTransaction(apiKey.SponsorAccount, fundStroops)
	if err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to build fund transaction")
		return nil, NewInternal("internal_error", "Failed to build funding transaction")
	}

	return &BuildFundResult{
		SponsorAccount: apiKey.SponsorAccount,
		XLMToAdd:       amountXLM,
		TransactionXDR: unsignedXDR,
	}, nil
}

// SubmitFundResult contains the output of a submitted fund transaction.
type SubmitFundResult struct {
	SponsorAccount  string
	XLMAdded        string
	XLMAvailable    string
	TransactionHash string
}

// SubmitFund validates and submits a signed fund transaction.
func (s *FundingService) SubmitFund(ctx context.Context, id uuid.UUID, signedXDR string) (*SubmitFundResult, error) {
	apiKey, err := s.store.GetAPIKeyByID(ctx, id)
	if err != nil {
		return nil, NewNotFound("not_found", "API key not found")
	}

	if apiKey.Status != model.StatusActive {
		return nil, NewBadRequest("invalid_status", "API key must be active to fund")
	}

	xlmAdded, err := validateFundTransaction(signedXDR, s.networkPassphrase, s.masterPublicKey, apiKey.SponsorAccount)
	if err != nil {
		return nil, NewBadRequest("invalid_request", err.Error())
	}

	resp, err := s.horizonClient.SubmitTransactionXDR(signedXDR)
	if err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to submit fund transaction")
		return nil, NewBadRequest("submission_failed", "Failed to submit transaction: "+err.Error())
	}

	available, _, err := s.accounts.GetBalance(apiKey.SponsorAccount)
	if err != nil {
		log.Error().Err(err).Msg("failed to get updated balance")
		available = "unknown"
	}

	return &SubmitFundResult{
		SponsorAccount:  apiKey.SponsorAccount,
		XLMAdded:        xlmAdded,
		XLMAvailable:    available,
		TransactionHash: resp.Hash,
	}, nil
}

// SweepResult contains the output of a sweep operation.
type SweepResult struct {
	SponsorAccount     string
	XLMSwept           string
	XLMRemainingLocked string
	Destination        string
	TransactionHash    string
}

// Sweep sweeps available funds from a revoked sponsor account back to master.
func (s *FundingService) Sweep(ctx context.Context, id uuid.UUID) (*SweepResult, error) {
	apiKey, err := s.store.GetAPIKeyByID(ctx, id)
	if err != nil {
		return nil, NewNotFound("not_found", "API key not found")
	}

	if apiKey.Status != model.StatusRevoked {
		return nil, NewBadRequest("invalid_status", "Can only sweep revoked API keys")
	}

	buildResult, err := s.builder.BuildSweepTransaction(s.signer, s.accounts, apiKey.SponsorAccount)
	if err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to build sweep transaction")
		return nil, NewInternal("sweep_failed", "Failed to build sweep transaction: "+err.Error())
	}

	if buildResult.NothingToSweep {
		return &SweepResult{
			SponsorAccount:     apiKey.SponsorAccount,
			XLMSwept:           buildResult.XLMSwept,
			XLMRemainingLocked: buildResult.XLMRemainingLocked,
			Destination:        s.masterPublicKey,
		}, nil
	}

	resp, err := s.horizonClient.SubmitTransactionXDR(buildResult.SignedXDR)
	if err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to submit sweep transaction")
		return nil, NewInternal("sweep_failed", "Failed to submit sweep transaction: "+err.Error())
	}

	return &SweepResult{
		SponsorAccount:     apiKey.SponsorAccount,
		XLMSwept:           buildResult.XLMSwept,
		XLMRemainingLocked: buildResult.XLMRemainingLocked,
		Destination:        s.masterPublicKey,
		TransactionHash:    resp.Hash,
	}, nil
}

// --- Transaction validation helpers ---

func validateActivateTransaction(signedXDR, networkPassphrase, masterPublicKey string) (string, error) {
	tx, err := decodeV1Transaction(signedXDR)
	if err != nil {
		return "", fmt.Errorf("invalid signed_transaction_xdr")
	}

	if tx.SourceAccount().AccountID != masterPublicKey {
		return "", fmt.Errorf("activation transaction source must be the master account")
	}

	ops := tx.Operations()
	if len(ops) != 5 {
		return "", fmt.Errorf("activation transaction must contain exactly 5 operations")
	}

	beginSponsoring, ok := ops[0].(*txnbuild.BeginSponsoringFutureReserves)
	if !ok {
		return "", fmt.Errorf("operation 0 must be BeginSponsoringFutureReserves")
	}

	createAccount, ok := ops[1].(*txnbuild.CreateAccount)
	if !ok {
		return "", fmt.Errorf("operation 1 must be CreateAccount")
	}

	sponsorAccount := createAccount.Destination
	if sponsorAccount == "" {
		return "", fmt.Errorf("CreateAccount destination must not be empty")
	}

	if beginSponsoring.SponsoredID != sponsorAccount {
		return "", fmt.Errorf("BeginSponsoringFutureReserves must target the sponsor account")
	}

	if _, ok := ops[2].(*txnbuild.SetOptions); !ok {
		return "", fmt.Errorf("operation 2 must be SetOptions")
	}

	if _, ok := ops[3].(*txnbuild.SetOptions); !ok {
		return "", fmt.Errorf("operation 3 must be SetOptions")
	}

	endSponsoring, ok := ops[4].(*txnbuild.EndSponsoringFutureReserves)
	if !ok {
		return "", fmt.Errorf("operation 4 must be EndSponsoringFutureReserves")
	}
	if endSponsoring.SourceAccount != sponsorAccount {
		return "", fmt.Errorf("EndSponsoringFutureReserves source must be the sponsor account")
	}

	if _, err := tx.HashHex(networkPassphrase); err != nil {
		return "", fmt.Errorf("invalid signed_transaction_xdr")
	}

	return sponsorAccount, nil
}

func validateFundTransaction(signedXDR, networkPassphrase, masterPublicKey, sponsorAccount string) (string, error) {
	tx, err := decodeV1Transaction(signedXDR)
	if err != nil {
		return "", fmt.Errorf("invalid signed_transaction_xdr")
	}

	if tx.SourceAccount().AccountID != masterPublicKey {
		return "", fmt.Errorf("funding transaction source must be the master account")
	}

	ops := tx.Operations()
	if len(ops) != 1 {
		return "", fmt.Errorf("funding transaction must contain exactly one operation")
	}

	payment, ok := ops[0].(*txnbuild.Payment)
	if !ok {
		return "", fmt.Errorf("funding transaction must be a payment operation")
	}
	if !payment.Asset.IsNative() {
		return "", fmt.Errorf("funding transaction must transfer native XLM")
	}
	if payment.Destination != sponsorAccount {
		return "", fmt.Errorf("funding transaction destination must match the sponsor account")
	}
	if payment.SourceAccount != "" && payment.SourceAccount != masterPublicKey {
		return "", fmt.Errorf("funding operation source must be the master account")
	}

	stroops, err := amount.ParseInt64(payment.Amount)
	if err != nil || stroops <= 0 {
		return "", fmt.Errorf("funding amount must be positive")
	}

	if _, err := tx.HashHex(networkPassphrase); err != nil {
		return "", fmt.Errorf("invalid signed_transaction_xdr")
	}

	return payment.Amount, nil
}

func decodeV1Transaction(txXDR string) (*txnbuild.Transaction, error) {
	genericTx, err := txnbuild.TransactionFromXDR(txXDR)
	if err != nil {
		return nil, err
	}
	tx, ok := genericTx.Transaction()
	if !ok {
		return nil, fmt.Errorf("fee bump transactions are not supported")
	}
	return tx, nil
}
