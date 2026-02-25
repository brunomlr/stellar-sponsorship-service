package httputil

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the standard JSON error response body.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// RespondJSON writes a JSON response with the given status code.
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// RespondError writes a JSON error response.
func RespondError(w http.ResponseWriter, status int, code, message string) {
	RespondJSON(w, status, ErrorResponse{Error: code, Message: message})
}
