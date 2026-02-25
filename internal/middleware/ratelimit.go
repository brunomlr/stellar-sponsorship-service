package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/stellar-sponsorship-service/internal/model"
)

// RateLimiter implements per-API-key sliding window rate limiting.
type RateLimiter struct {
	mu          sync.Mutex
	counters    map[string]*window
	lastCleanup time.Time
}

type window struct {
	count       int
	windowStart time.Time
	resetAt     time.Time
	lastSeen    time.Time
}

const (
	cleanupInterval    = 5 * time.Minute
	expiredWindowGrace = 10 * time.Minute
	staleEntryTTL      = 24 * time.Hour
)

// NewRateLimiter creates a new in-memory rate limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		counters:    make(map[string]*window),
		lastCleanup: time.Now(),
	}
}

// Allow checks if the API key is within its rate limit.
// Returns (allowed, remaining, resetAt).
func (rl *RateLimiter) Allow(apiKey *model.APIKey) (bool, int, time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	keyID := apiKey.ID.String()
	now := time.Now()
	windowDuration := time.Duration(apiKey.RateLimitWindow) * time.Second

	w, exists := rl.counters[keyID]
	if !exists || now.After(w.resetAt) {
		rl.counters[keyID] = &window{
			count:       1,
			windowStart: now,
			resetAt:     now.Add(windowDuration),
			lastSeen:    now,
		}
		rl.cleanupLocked(now)
		return true, apiKey.RateLimitMax - 1, now.Add(windowDuration)
	}

	w.lastSeen = now
	resetAt := w.resetAt

	if w.count >= apiKey.RateLimitMax {
		rl.cleanupLocked(now)
		return false, 0, resetAt
	}

	w.count++
	rl.cleanupLocked(now)
	return true, apiKey.RateLimitMax - w.count, resetAt
}

// Remaining returns the remaining request count without incrementing.
func (rl *RateLimiter) Remaining(apiKey *model.APIKey) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	keyID := apiKey.ID.String()
	now := time.Now()

	w, exists := rl.counters[keyID]
	if !exists || now.After(w.resetAt) {
		rl.cleanupLocked(now)
		return apiKey.RateLimitMax
	}

	w.lastSeen = now
	remaining := apiKey.RateLimitMax - w.count
	if remaining < 0 {
		rl.cleanupLocked(now)
		return 0
	}

	rl.cleanupLocked(now)
	return remaining
}

// RateLimitMiddleware returns middleware that enforces per-key rate limits.
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := GetAPIKey(r.Context())
			if apiKey == nil {
				next.ServeHTTP(w, r)
				return
			}

			if apiKey.RateLimitMax <= 0 || apiKey.RateLimitWindow <= 0 {
				respondError(w, http.StatusInternalServerError, "invalid_key_configuration", "API key rate limit configuration is invalid")
				return
			}

			allowed, remaining, resetAt := rl.Allow(apiKey)

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(apiKey.RateLimitMax))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

			if !allowed {
				respondError(w, http.StatusTooManyRequests, "rate_limited", "Rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) cleanupLocked(now time.Time) {
	if now.Sub(rl.lastCleanup) < cleanupInterval {
		return
	}

	for keyID, w := range rl.counters {
		if now.Sub(w.lastSeen) > staleEntryTTL || now.After(w.resetAt.Add(expiredWindowGrace)) {
			delete(rl.counters, keyID)
		}
	}

	rl.lastCleanup = now
}
