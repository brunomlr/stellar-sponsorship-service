package stellar

import (
	"testing"

	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/txnbuild"
)

func TestIsXLMTransferPathPayments(t *testing.T) {
	issuer := randomIssuer(t)
	credit := txnbuild.CreditAsset{Code: "USDC", Issuer: issuer}

	t.Run("strict send with native send asset is rejected", func(t *testing.T) {
		op := &txnbuild.PathPaymentStrictSend{
			SendAsset:   txnbuild.NativeAsset{},
			SendAmount:  "1.0000000",
			Destination: randomIssuer(t),
			DestAsset:   credit,
			DestMin:     "1.0000000",
		}

		if !isXLMTransfer(op) {
			t.Fatal("expected XLM transfer detection")
		}
	})

	t.Run("strict receive with native destination asset is rejected", func(t *testing.T) {
		op := &txnbuild.PathPaymentStrictReceive{
			SendAsset:   credit,
			SendMax:     "1.0000000",
			Destination: randomIssuer(t),
			DestAsset:   txnbuild.NativeAsset{},
			DestAmount:  "1.0000000",
		}

		if !isXLMTransfer(op) {
			t.Fatal("expected XLM transfer detection")
		}
	})

	t.Run("non-native path payment is allowed", func(t *testing.T) {
		op := &txnbuild.PathPaymentStrictReceive{
			SendAsset:   credit,
			SendMax:     "1.0000000",
			Destination: randomIssuer(t),
			DestAsset:   txnbuild.CreditAsset{Code: "EURC", Issuer: randomIssuer(t)},
			DestAmount:  "1.0000000",
		}

		if isXLMTransfer(op) {
			t.Fatal("did not expect XLM transfer detection")
		}
	})
}

func randomIssuer(t *testing.T) string {
	t.Helper()
	kp, err := keypair.Random()
	if err != nil {
		t.Fatalf("random keypair: %v", err)
	}
	return kp.Address()
}
