package handler

import (
	"net/http"

	"github.com/stellar-sponsorship-service/internal/httputil"
)

// ErrorResponse is the standard JSON error response body.
type ErrorResponse = httputil.ErrorResponse

// RespondJSON writes a JSON response with the given status code.
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	httputil.RespondJSON(w, status, data)
}

// RespondError writes a JSON error response.
func RespondError(w http.ResponseWriter, status int, code, message string) {
	httputil.RespondError(w, status, code, message)
}
