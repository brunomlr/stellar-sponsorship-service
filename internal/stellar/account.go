package stellar

import (
	"fmt"

	"github.com/stellar/go-stellar-sdk/amount"
	"github.com/stellar/go-stellar-sdk/clients/horizonclient"
)

// BaseReserveStroops is the Stellar base reserve in stroops (0.5 XLM).
const BaseReserveStroops int64 = 5_000_000

// AccountService queries Stellar account data via Horizon.
type AccountService struct {
	horizonClient *horizonclient.Client
}

// NewAccountService creates a new account service.
func NewAccountService(horizonClient *horizonclient.Client) *AccountService {
	return &AccountService{horizonClient: horizonClient}
}

// GetBalance returns the native XLM balance for an account.
// Returns (available, locked, error) as formatted strings like "100.5000000".
func (a *AccountService) GetBalance(accountID string) (string, string, error) {
	account, err := a.horizonClient.AccountDetail(horizonclient.AccountRequest{
		AccountID: accountID,
	})
	if err != nil {
		return "", "", fmt.Errorf("load account %s: %w", accountID, err)
	}

	// Find native balance
	var balanceStroops int64
	for _, b := range account.Balances {
		if b.Asset.Type == "native" {
			balanceStroops, err = amount.ParseInt64(b.Balance)
			if err != nil {
				return "", "", fmt.Errorf("parse balance: %w", err)
			}
			break
		}
	}

	// Calculate minimum balance in stroops:
	// minBalance = (2 + subentryCount + numSponsoring - numSponsored) * baseReserve
	minBalance := (2 + int64(account.SubentryCount) + int64(account.NumSponsoring) - int64(account.NumSponsored)) * BaseReserveStroops

	locked := amount.StringFromInt64(minBalance)
	available := balanceStroops - minBalance
	if available < 0 {
		available = 0
	}

	return amount.StringFromInt64(available), locked, nil
}

// GetRawBalance returns the total native XLM balance string for an account.
func (a *AccountService) GetRawBalance(accountID string) (string, error) {
	account, err := a.horizonClient.AccountDetail(horizonclient.AccountRequest{
		AccountID: accountID,
	})
	if err != nil {
		return "", fmt.Errorf("load account %s: %w", accountID, err)
	}

	for _, b := range account.Balances {
		if b.Asset.Type == "native" {
			return b.Balance, nil
		}
	}
	return "0.0000000", nil
}
