package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeTokenVerifier struct {
	claims *IDClaims
	err    error
}

func (f *fakeTokenVerifier) VerifyClaims(_ context.Context, _ string) (*IDClaims, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.claims, nil
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		email := GetAdminEmail(r.Context())
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"email":%q}`, email)
	})
}

func parseErrorResponse(t *testing.T, rec *httptest.ResponseRecorder) (string, string) {
	t.Helper()
	var resp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	return resp.Error, resp.Message
}

func TestGoogleAuth_ValidTokenAllowedEmail(t *testing.T) {
	verifier := &fakeTokenVerifier{
		claims: &IDClaims{
			Email:         "admin@company.com",
			EmailVerified: true,
			HD:            "company.com",
		},
	}

	ga := NewGoogleAuthWithVerifier(verifier, "company.com", []string{"admin@company.com"})
	handler := ga.Middleware(nil)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Email string `json:"email"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Email != "admin@company.com" {
		t.Fatalf("expected admin email in context, got %q", body.Email)
	}
}

func TestGoogleAuth_MissingToken(t *testing.T) {
	verifier := &fakeTokenVerifier{}
	ga := NewGoogleAuthWithVerifier(verifier, "company.com", []string{"admin@company.com"})
	handler := ga.Middleware(nil)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	code, _ := parseErrorResponse(t, rec)
	if code != "unauthorized" {
		t.Fatalf("expected 'unauthorized' error code, got %q", code)
	}
}

func TestGoogleAuth_InvalidToken(t *testing.T) {
	verifier := &fakeTokenVerifier{
		err: fmt.Errorf("invalid token signature"),
	}
	ga := NewGoogleAuthWithVerifier(verifier, "company.com", []string{"admin@company.com"})
	handler := ga.Middleware(nil)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGoogleAuth_UnverifiedEmail(t *testing.T) {
	verifier := &fakeTokenVerifier{
		claims: &IDClaims{
			Email:         "admin@company.com",
			EmailVerified: false,
			HD:            "company.com",
		},
	}
	ga := NewGoogleAuthWithVerifier(verifier, "company.com", []string{"admin@company.com"})
	handler := ga.Middleware(nil)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	code, _ := parseErrorResponse(t, rec)
	if code != "forbidden" {
		t.Fatalf("expected 'forbidden' error code, got %q", code)
	}
}

func TestGoogleAuth_WrongDomain(t *testing.T) {
	verifier := &fakeTokenVerifier{
		claims: &IDClaims{
			Email:         "user@evil.com",
			EmailVerified: true,
			HD:            "evil.com",
		},
	}
	ga := NewGoogleAuthWithVerifier(verifier, "company.com", []string{"admin@company.com"})
	handler := ga.Middleware(nil)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	_, msg := parseErrorResponse(t, rec)
	if msg != "Domain not allowed" {
		t.Fatalf("expected 'Domain not allowed' message, got %q", msg)
	}
}

func TestGoogleAuth_EmailNotInAllowlist(t *testing.T) {
	verifier := &fakeTokenVerifier{
		claims: &IDClaims{
			Email:         "other@company.com",
			EmailVerified: true,
			HD:            "company.com",
		},
	}
	ga := NewGoogleAuthWithVerifier(verifier, "company.com", []string{"admin@company.com"})
	handler := ga.Middleware(nil)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	_, msg := parseErrorResponse(t, rec)
	if msg != "User not authorized" {
		t.Fatalf("expected 'User not authorized' message, got %q", msg)
	}
}

func TestGoogleAuth_RateLimiting(t *testing.T) {
	verifier := &fakeTokenVerifier{
		err: fmt.Errorf("invalid token"),
	}
	limiter := NewAuthAttemptLimiter(3, 5*time.Minute, 15*time.Minute)
	ga := NewGoogleAuthWithVerifier(verifier, "company.com", []string{"admin@company.com"})
	handler := ga.Middleware(limiter)(okHandler())

	// Send 3 failed requests to trigger lockout
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		req.Header.Set("Authorization", "Bearer bad-token")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("request %d: expected 401, got %d", i+1, rec.Code)
		}
	}

	// Next request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("Authorization", "Bearer bad-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after rate limit, got %d", rec.Code)
	}
}
