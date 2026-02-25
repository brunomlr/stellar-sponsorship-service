package stellar

import (
	"fmt"

	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/txnbuild"
)

// Signer holds the signing key and signs verified transactions.
type Signer struct {
	signingKey        *keypair.Full
	networkPassphrase string
}

// NewSigner creates a new transaction signer.
// The secretKey must be a valid Stellar secret key (S...).
func NewSigner(secretKey string, networkPassphrase string) (*Signer, error) {
	kp, err := keypair.ParseFull(secretKey)
	if err != nil {
		return nil, fmt.Errorf("invalid signing key: %w", err)
	}
	return &Signer{signingKey: kp, networkPassphrase: networkPassphrase}, nil
}

// PublicKey returns the public key (G...) of the signing key.
func (s *Signer) PublicKey() string {
	return s.signingKey.Address()
}

// Sign adds the signing key's signature to the transaction envelope.
// Returns (signedXDR, transactionHashHex, error).
func (s *Signer) Sign(txXDR string) (string, string, error) {
	// 1. Parse transaction from XDR
	genericTx, err := txnbuild.TransactionFromXDR(txXDR)
	if err != nil {
		return "", "", fmt.Errorf("parse transaction XDR: %w", err)
	}

	// 2. Extract the *Transaction (reject fee bump transactions)
	tx, ok := genericTx.Transaction()
	if !ok {
		return "", "", fmt.Errorf("only V1 transaction envelopes are supported")
	}

	// 3. Sign with the signing key
	// IMPORTANT: Sign() returns a new *Transaction — must reassign
	tx, err = tx.Sign(s.networkPassphrase, s.signingKey)
	if err != nil {
		return "", "", fmt.Errorf("sign transaction: %w", err)
	}

	// 4. Get the transaction hash
	hashHex, err := tx.HashHex(s.networkPassphrase)
	if err != nil {
		return "", "", fmt.Errorf("compute transaction hash: %w", err)
	}

	// 5. Get signed XDR
	signedXDR, err := tx.Base64()
	if err != nil {
		return "", "", fmt.Errorf("encode signed transaction: %w", err)
	}

	return signedXDR, hashHex, nil
}
