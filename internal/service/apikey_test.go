package service

import (
	"strings"
	"testing"
)

func TestNormalizeRateLimit(t *testing.T) {
	t.Run("uses defaults when nil", func(t *testing.T) {
		max, window, err := normalizeRateLimit(nil, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if max != defaultRateLimitMax || window != defaultRateLimitWindow {
			t.Fatalf("unexpected defaults: max=%d window=%d", max, window)
		}
	})

	t.Run("accepts valid values", func(t *testing.T) {
		maxReq := 500
		windowSec := 120
		max, window, err := normalizeRateLimit(&maxReq, &windowSec)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if max != 500 || window != 120 {
			t.Fatalf("unexpected values: max=%d window=%d", max, window)
		}
	})

	t.Run("rejects invalid max", func(t *testing.T) {
		maxReq := 0
		windowSec := 120
		_, _, err := normalizeRateLimit(&maxReq, &windowSec)
		if err == nil || !strings.Contains(err.Error(), "max_requests") {
			t.Fatalf("expected max_requests error, got %v", err)
		}
	})

	t.Run("rejects invalid window", func(t *testing.T) {
		maxReq := 10
		windowSec := 0
		_, _, err := normalizeRateLimit(&maxReq, &windowSec)
		if err == nil || !strings.Contains(err.Error(), "window_seconds") {
			t.Fatalf("expected window_seconds error, got %v", err)
		}
	})
}

func TestGenerateAPIKeyPrefix(t *testing.T) {
	t.Run("testnet prefix", func(t *testing.T) {
		k, err := generateAPIKey("testnet")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !strings.HasPrefix(k, "sk_test_") {
			t.Fatalf("unexpected prefix: %s", k)
		}
	})

	t.Run("mainnet prefix", func(t *testing.T) {
		k, err := generateAPIKey("mainnet")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !strings.HasPrefix(k, "sk_live_") {
			t.Fatalf("unexpected prefix: %s", k)
		}
	})
}
