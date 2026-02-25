package service

import (
	"strings"
	"testing"

	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/network"
	"github.com/stellar/go-stellar-sdk/txnbuild"
)

func TestValidateActivateTransaction(t *testing.T) {
	master := randomAddress(t)
	sponsor := randomAddress(t)
	signingKey := randomAddress(t)

	validXDR := buildTransactionXDR(t, master, 1, []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{SponsoredID: sponsor},
		&txnbuild.CreateAccount{Destination: sponsor, Amount: "10.0000000"},
		&txnbuild.SetOptions{
			SourceAccount: sponsor,
			Signer:        &txnbuild.Signer{Address: signingKey, Weight: txnbuild.Threshold(1)},
		},
		&txnbuild.SetOptions{
			SourceAccount: sponsor,
			Signer:        &txnbuild.Signer{Address: master, Weight: txnbuild.Threshold(1)},
		},
		&txnbuild.EndSponsoringFutureReserves{SourceAccount: sponsor},
	})

	t.Run("accepts valid activation transaction", func(t *testing.T) {
		account, err := validateActivateTransaction(validXDR, network.TestNetworkPassphrase, master)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if account != sponsor {
			t.Fatalf("expected sponsor %s, got %s", sponsor, account)
		}
	})

	t.Run("rejects wrong source account", func(t *testing.T) {
		other := randomAddress(t)
		xdr := buildTransactionXDR(t, other, 1, []txnbuild.Operation{
			&txnbuild.BeginSponsoringFutureReserves{SponsoredID: sponsor},
			&txnbuild.CreateAccount{Destination: sponsor, Amount: "10.0000000"},
			&txnbuild.SetOptions{SourceAccount: sponsor},
			&txnbuild.SetOptions{SourceAccount: sponsor},
			&txnbuild.EndSponsoringFutureReserves{SourceAccount: sponsor},
		})

		_, err := validateActivateTransaction(xdr, network.TestNetworkPassphrase, master)
		if err == nil || !strings.Contains(err.Error(), "master account") {
			t.Fatalf("expected master account error, got %v", err)
		}
	})

	t.Run("rejects wrong number of operations", func(t *testing.T) {
		xdr := buildTransactionXDR(t, master, 1, []txnbuild.Operation{
			&txnbuild.Payment{Destination: sponsor, Amount: "10.0000000", Asset: txnbuild.NativeAsset{}},
		})

		_, err := validateActivateTransaction(xdr, network.TestNetworkPassphrase, master)
		if err == nil || !strings.Contains(err.Error(), "5 operations") {
			t.Fatalf("expected 5 operations error, got %v", err)
		}
	})

	t.Run("rejects invalid XDR", func(t *testing.T) {
		_, err := validateActivateTransaction("not-xdr", network.TestNetworkPassphrase, master)
		if err == nil {
			t.Fatal("expected error for invalid XDR")
		}
	})
}

func TestValidateFundTransaction(t *testing.T) {
	master := randomAddress(t)
	sponsor := randomAddress(t)
	other := randomAddress(t)
	issuer := randomAddress(t)

	t.Run("accepts valid native payment", func(t *testing.T) {
		xdr := buildTransactionXDR(t, master, 1, []txnbuild.Operation{
			&txnbuild.Payment{
				Destination: sponsor,
				Amount:      "5.0000000",
				Asset:       txnbuild.NativeAsset{},
			},
		})

		amount, err := validateFundTransaction(xdr, network.TestNetworkPassphrase, master, sponsor)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if amount != "5.0000000" {
			t.Fatalf("unexpected amount %q", amount)
		}
	})

	t.Run("rejects non-master source", func(t *testing.T) {
		xdr := buildTransactionXDR(t, other, 1, []txnbuild.Operation{
			&txnbuild.Payment{Destination: sponsor, Amount: "1.0000000", Asset: txnbuild.NativeAsset{}},
		})

		_, err := validateFundTransaction(xdr, network.TestNetworkPassphrase, master, sponsor)
		if err == nil || !strings.Contains(err.Error(), "source") {
			t.Fatalf("expected source error, got %v", err)
		}
	})

	t.Run("rejects non-native payment", func(t *testing.T) {
		xdr := buildTransactionXDR(t, master, 1, []txnbuild.Operation{
			&txnbuild.Payment{
				Destination: sponsor,
				Amount:      "2.0000000",
				Asset:       txnbuild.CreditAsset{Code: "USDC", Issuer: issuer},
			},
		})

		_, err := validateFundTransaction(xdr, network.TestNetworkPassphrase, master, sponsor)
		if err == nil || !strings.Contains(err.Error(), "native XLM") {
			t.Fatalf("expected native asset error, got %v", err)
		}
	})

	t.Run("rejects wrong destination", func(t *testing.T) {
		xdr := buildTransactionXDR(t, master, 1, []txnbuild.Operation{
			&txnbuild.Payment{Destination: other, Amount: "2.0000000", Asset: txnbuild.NativeAsset{}},
		})

		_, err := validateFundTransaction(xdr, network.TestNetworkPassphrase, master, sponsor)
		if err == nil || !strings.Contains(err.Error(), "destination") {
			t.Fatalf("expected destination error, got %v", err)
		}
	})

	t.Run("rejects multiple operations", func(t *testing.T) {
		xdr := buildTransactionXDR(t, master, 1, []txnbuild.Operation{
			&txnbuild.Payment{Destination: sponsor, Amount: "1.0000000", Asset: txnbuild.NativeAsset{}},
			&txnbuild.Payment{Destination: sponsor, Amount: "1.0000000", Asset: txnbuild.NativeAsset{}},
		})

		_, err := validateFundTransaction(xdr, network.TestNetworkPassphrase, master, sponsor)
		if err == nil || !strings.Contains(err.Error(), "exactly one operation") {
			t.Fatalf("expected op-count error, got %v", err)
		}
	})

	t.Run("rejects non-payment operation", func(t *testing.T) {
		xdr := buildTransactionXDR(t, master, 1, []txnbuild.Operation{
			&txnbuild.ManageData{Name: "k", Value: []byte("v")},
		})

		_, err := validateFundTransaction(xdr, network.TestNetworkPassphrase, master, sponsor)
		if err == nil || !strings.Contains(err.Error(), "payment operation") {
			t.Fatalf("expected payment-op error, got %v", err)
		}
	})

	t.Run("rejects invalid XDR", func(t *testing.T) {
		_, err := validateFundTransaction("not-xdr", network.TestNetworkPassphrase, master, sponsor)
		if err == nil {
			t.Fatal("expected invalid xdr error")
		}
	})
}

func buildTransactionXDR(t *testing.T, source string, sequence int64, ops []txnbuild.Operation) string {
	t.Helper()

	sa := txnbuild.NewSimpleAccount(source, sequence)
	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &sa,
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

func randomAddress(t *testing.T) string {
	t.Helper()

	kp, err := keypair.Random()
	if err != nil {
		t.Fatalf("random keypair: %v", err)
	}
	return kp.Address()
}
