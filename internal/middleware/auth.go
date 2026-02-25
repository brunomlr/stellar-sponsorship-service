package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/stellar-sponsorship-service/internal/model"
	"github.com/stellar-sponsorship-service/internal/store"
)

type contextKey string

const apiKeyContextKey contextKey = "api_key"

// GetAPIKey extracts the authenticated API key from the request context.
func GetAPIKey(ctx context.Context) *model.APIKey {
	key, _ := ctx.Value(apiKeyContextKey).(*model.APIKey)
	return key
}

// APIKeyAuth returns middleware that authenticates requests via Bearer token.
func APIKeyAuth(s store.APIKeyStore, limiter *AuthAttemptLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptKey := clientIPKey(r, "api_key")
			if limiter != nil && !limiter.allow(attemptKey) {
				respondError(w, http.StatusTooManyRequests, "rate_limited", "Too many authentication failures")
				return
			}

			token := extractBearerToken(r)
			if token == "" {
				if limiter != nil {
					limiter.registerFailure(attemptKey)
				}
				respondError(w, http.StatusUnauthorized, "invalid_api_key", "Missing API key")
				return
			}

			keyHash := SHA256Hex(token)
			apiKey, err := s.GetAPIKeyByHash(r.Context(), keyHash)
			if err != nil {
				if limiter != nil {
					limiter.registerFailure(attemptKey)
				}
				respondError(w, http.StatusUnauthorized, "invalid_api_key", "Invalid API key")
				return
			}

			if time.Now().After(apiKey.ExpiresAt) {
				if limiter != nil {
					limiter.registerFailure(attemptKey)
				}
				respondError(w, http.StatusUnauthorized, "invalid_api_key", "API key has expired")
				return
			}

			if apiKey.Status != model.StatusActive {
				if limiter != nil {
					limiter.registerFailure(attemptKey)
				}
				respondError(w, http.StatusForbidden, "key_disabled", "API key is not active")
				return
			}

			if limiter != nil {
				limiter.registerSuccess(attemptKey)
			}
			ctx := context.WithValue(r.Context(), apiKeyContextKey, apiKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

// SHA256Hex returns the hex-encoded SHA-256 hash of the input.
func SHA256Hex(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}
