package middleware

import (
	"net/http"

	"github.com/stellar-sponsorship-service/internal/httputil"
)

func respondError(w http.ResponseWriter, status int, code, message string) {
	httputil.RespondError(w, status, code, message)
}
