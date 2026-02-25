package validation

import (
	"fmt"

	"github.com/stellar/go-stellar-sdk/keypair"

	"github.com/stellar-sponsorship-service/internal/stellar"
)

// AllowedOperations validates that all operations are supported and unique.
func AllowedOperations(ops []string) error {
	if len(ops) == 0 {
		return fmt.Errorf("allowed_operations cannot be empty")
	}

	supported := make(map[string]struct{}, len(stellar.SupportedOperations()))
	for _, op := range stellar.SupportedOperations() {
		supported[op] = struct{}{}
	}

	seen := make(map[string]struct{}, len(ops))
	for _, op := range ops {
		if _, ok := supported[op]; !ok {
			return fmt.Errorf("operation %q is not supported", op)
		}
		if _, exists := seen[op]; exists {
			return fmt.Errorf("duplicate operation %q is not allowed", op)
		}
		seen[op] = struct{}{}
	}

	return nil
}

// SourceAccounts validates that all source accounts are valid Stellar public keys.
func SourceAccounts(accounts []string) error {
	for _, account := range accounts {
		if _, err := keypair.ParseAddress(account); err != nil {
			return fmt.Errorf("invalid source account %q", account)
		}
	}
	return nil
}
