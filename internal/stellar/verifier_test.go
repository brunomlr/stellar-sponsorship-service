package stellar

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/network"
	"github.com/stellar/go-stellar-sdk/txnbuild"

	"github.com/stellar-sponsorship-service/internal/model"
)

func TestVerifierRejectsOperationSourceThatDoesNotMatchSponsoredID(t *testing.T) {
	sponsor := randomStellarAddress(t)
	sponsored := randomStellarAddress(t)
	otherSource := randomStellarAddress(t)

	txXDR := buildVerifierTestXDR(t, sponsored, []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SourceAccount: sponsor,
			SponsoredID:   sponsored,
		},
		&txnbuild.ManageData{
			SourceAccount: otherSource,
			Name:          "k",
			Value:         []byte("v"),
		},
		&txnbuild.EndSponsoringFutureReserves{
			SourceAccount: sponsored,
		},
	})

	apiKey := &model.APIKey{
		ID:                uuid.New(),
		SponsorAccount:    sponsor,
		AllowedOperations: []string{"MANAGE_DATA"},
		RateLimitMax:      100,
		RateLimitWindow:   60,
		Status:            model.StatusActive,
		ExpiresAt:         time.Now().UTC().Add(24 * time.Hour),
	}

	v := NewVerifier(network.TestNetworkPassphrase)
	result := v.Verify(txXDR, apiKey)

	if result.Valid {
		t.Fatal("expected verification to fail")
	}
	if result.ErrorCode != "invalid_transaction" {
		t.Fatalf("expected invalid_transaction, got %q", result.ErrorCode)
	}
	if !strings.Contains(result.ErrorMessage, "does not match SponsoredID") {
		t.Fatalf("unexpected error message: %q", result.ErrorMessage)
	}
}

func TestVerifierAcceptsOperationSourceMatchingSponsoredID(t *testing.T) {
	sponsor := randomStellarAddress(t)
	sponsored := randomStellarAddress(t)

	txXDR := buildVerifierTestXDR(t, sponsored, []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SourceAccount: sponsor,
			SponsoredID:   sponsored,
		},
		&txnbuild.ManageData{
			SourceAccount: sponsored,
			Name:          "k",
			Value:         []byte("v"),
		},
		&txnbuild.EndSponsoringFutureReserves{
			SourceAccount: sponsored,
		},
	})

	apiKey := &model.APIKey{
		ID:                uuid.New(),
		SponsorAccount:    sponsor,
		AllowedOperations: []string{"MANAGE_DATA"},
		RateLimitMax:      100,
		RateLimitWindow:   60,
		Status:            model.StatusActive,
		ExpiresAt:         time.Now().UTC().Add(24 * time.Hour),
	}

	v := NewVerifier(network.TestNetworkPassphrase)
	result := v.Verify(txXDR, apiKey)

	if !result.Valid {
		t.Fatalf("expected verification success, got %q", result.ErrorMessage)
	}
	if len(result.Operations) != 1 || result.Operations[0] != "MANAGE_DATA" {
		t.Fatalf("unexpected operations list: %#v", result.Operations)
	}
}

func buildVerifierTestXDR(t *testing.T, source string, ops []txnbuild.Operation) string {
	t.Helper()

	account := txnbuild.NewSimpleAccount(source, 1)
	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &account,
		IncrementSequenceNum: true,
		BaseFee:              txnbuild.MinBaseFee,
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimeout(300)},
		Operations:           ops,
	})
	if err != nil {
		t.Fatalf("build tx: %v", err)
	}

	xdr, err := tx.Base64()
	if err != nil {
		t.Fatalf("encode tx: %v", err)
	}
	return xdr
}

func randomStellarAddress(t *testing.T) string {
	t.Helper()

	kp, err := keypair.Random()
	if err != nil {
		t.Fatalf("random keypair: %v", err)
	}
	return kp.Address()
}
