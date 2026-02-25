package middleware

import (
	"net/http"
	"strings"
)

// RequireJSON rejects POST/PATCH/PUT requests that don't have a JSON Content-Type.
func RequireJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPatch || r.Method == http.MethodPut {
			ct := r.Header.Get("Content-Type")
			if ct != "" && !strings.HasPrefix(ct, "application/json") {
				respondError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
