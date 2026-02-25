package stellar

import (
	"fmt"

	"github.com/stellar/go-stellar-sdk/amount"
	"github.com/stellar/go-stellar-sdk/clients/horizonclient"
	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/txnbuild"
)

// Builder builds unsigned transactions for admin operations.
type Builder struct {
	horizonClient     *horizonclient.Client
	signingPublicKey  string
	masterPublicKey   string
	networkPassphrase string
}

// NewBuilder creates a new transaction builder.
func NewBuilder(
	horizonClient *horizonclient.Client,
	signingPublicKey string,
	masterPublicKey string,
	networkPassphrase string,
) *Builder {
	return &Builder{
		horizonClient:     horizonClient,
		signingPublicKey:  signingPublicKey,
		masterPublicKey:   masterPublicKey,
		networkPassphrase: networkPassphrase,
	}
}

// BuildCreateSponsorAccount builds a transaction that:
// 1. Begins reserve sponsoring from the master account for the new sponsor account
// 2. Creates a new Stellar account funded with xlmBudget (in stroops)
// 3. Sets both the signing key and the master key as signers on the new account
// 4. Removes the sponsor account's own signing privilege (master weight = 0)
// 5. Ends reserve sponsoring
//
// The master account sponsors all reserves (base account + signer sub-entries),
// so the full XLM budget is available for sponsoring operations.
//
// The transaction is pre-signed with the sponsor keypair (needed for SetOptions
// and EndSponsoringFutureReserves). The returned XDR only needs the master
// account signature (via Freighter) before submission.
func (b *Builder) BuildCreateSponsorAccount(
	sponsorKP *keypair.Full,
	xlmBudget int64,
) (string, error) {
	sponsorAddress := sponsorKP.Address()

	// Load master account from Horizon for sequence number
	masterAccount, err := b.horizonClient.AccountDetail(horizonclient.AccountRequest{
		AccountID: b.masterPublicKey,
	})
	if err != nil {
		return "", fmt.Errorf("load master account: %w", err)
	}

	masterWeight := txnbuild.Threshold(0)
	lowThreshold := txnbuild.Threshold(1)
	medThreshold := txnbuild.Threshold(1)
	highThreshold := txnbuild.Threshold(1)

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &masterAccount,
		IncrementSequenceNum: true,
		BaseFee:              txnbuild.MinBaseFee,
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(300)},
		Operations: []txnbuild.Operation{
			// 1. Master begins sponsoring reserves for the new sponsor account
			&txnbuild.BeginSponsoringFutureReserves{
				SponsoredID: sponsorAddress,
			},
			// 2. Create the sponsor account with the XLM budget
			&txnbuild.CreateAccount{
				Destination: sponsorAddress,
				Amount:      amount.StringFromInt64(xlmBudget),
			},
			// 3. Add signing key as signer on the sponsor account
			&txnbuild.SetOptions{
				SourceAccount: sponsorAddress,
				Signer: &txnbuild.Signer{
					Address: b.signingPublicKey,
					Weight:  txnbuild.Threshold(1),
				},
			},
			// 4. Add master key as signer, remove sponsor's own signing privilege, set thresholds
			&txnbuild.SetOptions{
				SourceAccount: sponsorAddress,
				Signer: &txnbuild.Signer{
					Address: b.masterPublicKey,
					Weight:  txnbuild.Threshold(1),
				},
				MasterWeight:    &masterWeight,
				LowThreshold:    &lowThreshold,
				MediumThreshold: &medThreshold,
				HighThreshold:   &highThreshold,
			},
			// 5. End sponsoring (sponsor account agrees to be sponsored)
			&txnbuild.EndSponsoringFutureReserves{
				SourceAccount: sponsorAddress,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("build create sponsor account tx: %w", err)
	}

	// Pre-sign with the sponsor keypair (needed for ops sourced from sponsor account)
	tx, err = tx.Sign(b.networkPassphrase, sponsorKP)
	if err != nil {
		return "", fmt.Errorf("sign create sponsor account tx: %w", err)
	}

	xdr, err := tx.Base64()
	if err != nil {
		return "", fmt.Errorf("encode transaction: %w", err)
	}
	return xdr, nil
}

// BuildFundTransaction builds an unsigned payment from master to sponsor account.
// The amount is in stroops.
func (b *Builder) BuildFundTransaction(
	sponsorAccount string,
	fundAmount int64,
) (string, error) {
	masterAccount, err := b.horizonClient.AccountDetail(horizonclient.AccountRequest{
		AccountID: b.masterPublicKey,
	})
	if err != nil {
		return "", fmt.Errorf("load master account: %w", err)
	}

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &masterAccount,
		IncrementSequenceNum: true,
		BaseFee:              txnbuild.MinBaseFee,
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(300)},
		Operations: []txnbuild.Operation{
			&txnbuild.Payment{
				Destination: sponsorAccount,
				Amount:      amount.StringFromInt64(fundAmount),
				Asset:       txnbuild.NativeAsset{},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("build fund tx: %w", err)
	}

	xdr, err := tx.Base64()
	if err != nil {
		return "", fmt.Errorf("encode transaction: %w", err)
	}
	return xdr, nil
}

// SweepResult contains the outputs from building a sweep transaction.
type SweepResult struct {
	SignedXDR          string // base64-encoded signed transaction envelope
	XLMSwept           string // amount swept in XLM format (e.g. "10.0000000")
	XLMRemainingLocked string // locked reserves in XLM format
	NothingToSweep     bool   // true when available balance is zero or too small
}

// BuildSweepTransaction builds and signs a payment from sponsor account back to master.
// The caller is responsible for submitting the returned XDR to the Stellar network.
func (b *Builder) BuildSweepTransaction(
	signer *Signer,
	accounts *AccountService,
	sponsorAccount string,
) (*SweepResult, error) {
	available, locked, err := accounts.GetBalance(sponsorAccount)
	if err != nil {
		return nil, fmt.Errorf("get sponsor balance: %w", err)
	}

	availableStroops, err := amount.ParseInt64(available)
	if err != nil {
		return nil, fmt.Errorf("parse available balance: %w", err)
	}

	if availableStroops <= 0 {
		return &SweepResult{
			XLMSwept:           "0.0000000",
			XLMRemainingLocked: locked,
			NothingToSweep:     true,
		}, nil
	}

	// Load sponsor account for sequence number
	sponsorAccountDetail, err := b.horizonClient.AccountDetail(horizonclient.AccountRequest{
		AccountID: sponsorAccount,
	})
	if err != nil {
		return nil, fmt.Errorf("load sponsor account: %w", err)
	}

	// Sweep available XLM back to master, minus fee
	sweepAmount := availableStroops - txnbuild.MinBaseFee
	if sweepAmount <= 0 {
		return &SweepResult{
			XLMSwept:           "0.0000000",
			XLMRemainingLocked: locked,
			NothingToSweep:     true,
		}, nil
	}

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &sponsorAccountDetail,
		IncrementSequenceNum: true,
		BaseFee:              txnbuild.MinBaseFee,
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(300)},
		Operations: []txnbuild.Operation{
			&txnbuild.Payment{
				Destination: b.masterPublicKey,
				Amount:      amount.StringFromInt64(sweepAmount),
				Asset:       txnbuild.NativeAsset{},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("build sweep tx: %w", err)
	}

	// Sign with the service's signing key
	tx, err = tx.Sign(b.networkPassphrase, signer.signingKey)
	if err != nil {
		return nil, fmt.Errorf("sign sweep tx: %w", err)
	}

	signedXDR, err := tx.Base64()
	if err != nil {
		return nil, fmt.Errorf("encode sweep tx: %w", err)
	}

	return &SweepResult{
		SignedXDR:          signedXDR,
		XLMSwept:           amount.StringFromInt64(sweepAmount),
		XLMRemainingLocked: locked,
	}, nil
}
