package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stellar-sponsorship-service/internal/model"
)

func TestRateLimiterAllowAndReset(t *testing.T) {
	rl := NewRateLimiter()
	key := &model.APIKey{ID: uuid.New(), RateLimitMax: 2, RateLimitWindow: 1}

	allowed, remaining, _ := rl.Allow(key)
	if !allowed || remaining != 1 {
		t.Fatalf("unexpected first allow result: allowed=%v remaining=%d", allowed, remaining)
	}

	allowed, remaining, _ = rl.Allow(key)
	if !allowed || remaining != 0 {
		t.Fatalf("unexpected second allow result: allowed=%v remaining=%d", allowed, remaining)
	}

	allowed, remaining, _ = rl.Allow(key)
	if allowed || remaining != 0 {
		t.Fatalf("expected request to be rate-limited: allowed=%v remaining=%d", allowed, remaining)
	}

	time.Sleep(1100 * time.Millisecond)

	allowed, remaining, _ = rl.Allow(key)
	if !allowed || remaining != 1 {
		t.Fatalf("expected reset window allow: allowed=%v remaining=%d", allowed, remaining)
	}
}

func TestRateLimitMiddlewareRejectsInvalidKeyConfig(t *testing.T) {
	rl := NewRateLimiter()
	mw := RateLimitMiddleware(rl)

	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	badKey := &model.APIKey{ID: uuid.New(), RateLimitMax: 0, RateLimitWindow: 60}
	req := httptest.NewRequest(http.MethodGet, "/v1/sign", nil)
	req = req.WithContext(context.WithValue(req.Context(), apiKeyContextKey, badKey))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if called {
		t.Fatal("handler should not be called for invalid key configuration")
	}
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: %d", rr.Code)
	}
}

func TestRateLimiterCleanupRemovesStaleEntries(t *testing.T) {
	rl := NewRateLimiter()
	now := time.Now()

	rl.counters["stale"] = &window{
		count:       1,
		windowStart: now.Add(-48 * time.Hour),
		resetAt:     now.Add(-24 * time.Hour),
		lastSeen:    now.Add(-48 * time.Hour),
	}
	rl.lastCleanup = now.Add(-cleanupInterval - time.Second)

	key := &model.APIKey{ID: uuid.New(), RateLimitMax: 10, RateLimitWindow: 60}
	_, _, _ = rl.Allow(key)

	if _, exists := rl.counters["stale"]; exists {
		t.Fatal("expected stale rate-limit entry to be cleaned up")
	}
}
