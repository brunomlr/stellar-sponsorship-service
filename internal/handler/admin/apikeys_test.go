package admin

import (
	"strings"
	"testing"

	"github.com/stellar/go-stellar-sdk/keypair"

	"github.com/stellar-sponsorship-service/internal/validation"
)

func TestValidateAllowedOperations(t *testing.T) {
	t.Run("accepts supported operations", func(t *testing.T) {
		err := validation.AllowedOperations([]string{"CREATE_ACCOUNT", "MANAGE_DATA"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("rejects empty operations", func(t *testing.T) {
		err := validation.AllowedOperations(nil)
		if err == nil || !strings.Contains(err.Error(), "cannot be empty") {
			t.Fatalf("expected empty error, got %v", err)
		}
	})

	t.Run("rejects unsupported operation", func(t *testing.T) {
		err := validation.AllowedOperations([]string{"PAYMENT"})
		if err == nil || !strings.Contains(err.Error(), "not supported") {
			t.Fatalf("expected unsupported op error, got %v", err)
		}
	})

	t.Run("rejects duplicate operation", func(t *testing.T) {
		err := validation.AllowedOperations([]string{"MANAGE_DATA", "MANAGE_DATA"})
		if err == nil || !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("expected duplicate error, got %v", err)
		}
	})
}

func TestValidateSourceAccounts(t *testing.T) {
	kp1, err := keypair.Random()
	if err != nil {
		t.Fatalf("random keypair: %v", err)
	}
	kp2, err := keypair.Random()
	if err != nil {
		t.Fatalf("random keypair: %v", err)
	}

	t.Run("accepts valid accounts", func(t *testing.T) {
		err := validation.SourceAccounts([]string{kp1.Address(), kp2.Address()})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("rejects invalid account", func(t *testing.T) {
		err := validation.SourceAccounts([]string{"not-a-stellar-account"})
		if err == nil || !strings.Contains(err.Error(), "invalid source account") {
			t.Fatalf("expected invalid account error, got %v", err)
		}
	})
}
