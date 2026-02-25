package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

type adminEmailKey struct{}

// GetAdminEmail extracts the authenticated admin email from the request context.
func GetAdminEmail(ctx context.Context) string {
	email, _ := ctx.Value(adminEmailKey{}).(string)
	return email
}

// IDClaims holds the verified claims from a Google ID token.
type IDClaims struct {
	Email         string
	EmailVerified bool
	HD            string
}

// TokenVerifier verifies an ID token and returns its claims.
type TokenVerifier interface {
	VerifyClaims(ctx context.Context, rawToken string) (*IDClaims, error)
}

// googleTokenVerifier implements TokenVerifier using go-oidc.
type googleTokenVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func (v *googleTokenVerifier) VerifyClaims(ctx context.Context, rawToken string) (*IDClaims, error) {
	idToken, err := v.verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		HD            string `json:"hd"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("parse claims: %w", err)
	}

	return &IDClaims{
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		HD:            claims.HD,
	}, nil
}

// GoogleAuth verifies Google ID tokens and enforces domain + email allowlist restrictions.
type GoogleAuth struct {
	verifier      TokenVerifier
	allowedDomain string
	allowedEmails map[string]struct{}
}

// NewGoogleAuth creates a GoogleAuth middleware that verifies tokens against Google's JWKS.
// It must be called at server startup (it fetches Google's OIDC discovery document).
func NewGoogleAuth(clientID, allowedDomain string, allowedEmails []string) (*GoogleAuth, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("create Google OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})

	return NewGoogleAuthWithVerifier(&googleTokenVerifier{verifier: verifier}, allowedDomain, allowedEmails), nil
}

// NewGoogleAuthWithVerifier creates a GoogleAuth with a custom TokenVerifier.
func NewGoogleAuthWithVerifier(verifier TokenVerifier, allowedDomain string, allowedEmails []string) *GoogleAuth {
	emailSet := make(map[string]struct{}, len(allowedEmails))
	for _, e := range allowedEmails {
		emailSet[e] = struct{}{}
	}

	return &GoogleAuth{
		verifier:      verifier,
		allowedDomain: allowedDomain,
		allowedEmails: emailSet,
	}
}

// Middleware returns an http middleware that authenticates admin requests via Google ID tokens.
func (g *GoogleAuth) Middleware(limiter *AuthAttemptLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptKey := clientIPKey(r, "google_admin")
			if limiter != nil && !limiter.allow(attemptKey) {
				respondError(w, http.StatusTooManyRequests, "rate_limited", "Too many authentication failures")
				return
			}

			token := extractBearerToken(r)
			if token == "" {
				if limiter != nil {
					limiter.registerFailure(attemptKey)
				}
				respondError(w, http.StatusUnauthorized, "unauthorized", "Missing authorization token")
				return
			}

			claims, err := g.verifier.VerifyClaims(r.Context(), token)
			if err != nil {
				if limiter != nil {
					limiter.registerFailure(attemptKey)
				}
				respondError(w, http.StatusUnauthorized, "unauthorized", "Invalid ID token")
				return
			}

			if !claims.EmailVerified {
				if limiter != nil {
					limiter.registerFailure(attemptKey)
				}
				respondError(w, http.StatusForbidden, "forbidden", "Email not verified")
				return
			}

			if claims.HD != g.allowedDomain {
				if limiter != nil {
					limiter.registerFailure(attemptKey)
				}
				respondError(w, http.StatusForbidden, "forbidden", "Domain not allowed")
				return
			}

			if _, ok := g.allowedEmails[claims.Email]; !ok {
				if limiter != nil {
					limiter.registerFailure(attemptKey)
				}
				respondError(w, http.StatusForbidden, "forbidden", "User not authorized")
				return
			}

			if limiter != nil {
				limiter.registerSuccess(attemptKey)
			}
			ctx := context.WithValue(r.Context(), adminEmailKey{}, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
