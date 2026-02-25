package middleware

import (
	"testing"
	"time"
)

func TestAuthAttemptLimiterBlocksAfterThreshold(t *testing.T) {
	limiter := NewAuthAttemptLimiter(3, time.Minute, 150*time.Millisecond)
	key := "api_key:198.51.100.1"

	if !limiter.allow(key) {
		t.Fatal("expected initial request to be allowed")
	}

	limiter.registerFailure(key)
	limiter.registerFailure(key)
	limiter.registerFailure(key)

	if limiter.allow(key) {
		t.Fatal("expected request to be blocked after max failures")
	}

	time.Sleep(200 * time.Millisecond)
	if !limiter.allow(key) {
		t.Fatal("expected request to be allowed after block duration")
	}
}

func TestAuthAttemptLimiterSuccessResetsFailures(t *testing.T) {
	limiter := NewAuthAttemptLimiter(2, time.Minute, time.Minute)
	key := "admin:203.0.113.5"

	limiter.registerFailure(key)
	limiter.registerSuccess(key)
	limiter.registerFailure(key)

	if !limiter.allow(key) {
		t.Fatal("expected success to clear previous failures")
	}
}
