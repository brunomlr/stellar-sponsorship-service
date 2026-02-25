package service

import (
	"errors"
	"net/http"

	"github.com/stellar-sponsorship-service/internal/httputil"
)

// HTTPStatus maps an ErrorKind to its corresponding HTTP status code.
func (k ErrorKind) HTTPStatus() int {
	switch k {
	case ErrBadRequest:
		return http.StatusBadRequest
	case ErrNotFound:
		return http.StatusNotFound
	case ErrForbidden:
		return http.StatusForbidden
	case ErrInternal:
		return http.StatusInternalServerError
	case ErrUnavailable:
		return http.StatusServiceUnavailable
	case ErrBadGateway:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// RespondError writes an appropriate HTTP error response for a service error.
// If the error is a *service.Error, it uses the error's kind/code/message.
// Otherwise, it returns a generic 500.
func RespondError(w http.ResponseWriter, err error) {
	var svcErr *Error
	if errors.As(err, &svcErr) {
		httputil.RespondError(w, svcErr.Kind.HTTPStatus(), svcErr.Code, svcErr.Message)
		return
	}
	httputil.RespondError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
}
