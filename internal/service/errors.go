package service

import "fmt"

// Error is a domain error returned by service methods.
// Handlers map these to appropriate HTTP responses.
type Error struct {
	Kind    ErrorKind
	Code    string // machine-readable error code (e.g., "invalid_request", "not_found")
	Message string // human-readable message
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ErrorKind classifies domain errors for HTTP status mapping.
type ErrorKind int

const (
	ErrBadRequest   ErrorKind = iota // 400
	ErrNotFound                      // 404
	ErrForbidden                     // 403
	ErrInternal                      // 500
	ErrUnavailable                   // 503
	ErrBadGateway                    // 502
)

func NewBadRequest(code, message string) *Error {
	return &Error{Kind: ErrBadRequest, Code: code, Message: message}
}

func NewNotFound(code, message string) *Error {
	return &Error{Kind: ErrNotFound, Code: code, Message: message}
}

func NewInternal(code, message string) *Error {
	return &Error{Kind: ErrInternal, Code: code, Message: message}
}

func NewUnavailable(code, message string) *Error {
	return &Error{Kind: ErrUnavailable, Code: code, Message: message}
}

func NewBadGateway(code, message string) *Error {
	return &Error{Kind: ErrBadGateway, Code: code, Message: message}
}
