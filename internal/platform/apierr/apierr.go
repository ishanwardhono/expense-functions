// Package apierr defines typed errors that the httpx layer maps to HTTP status
// codes (spec §4.3). Services return these; handlers stay thin.
package apierr

import "fmt"

// Kind classifies an error for status-code mapping.
type Kind int

const (
	// KindInvalid is a validation failure / bad input → 400.
	KindInvalid Kind = iota
	// KindNotFound is an unknown id → 404.
	KindNotFound
	// KindConflict is a uniqueness/state conflict (e.g. duplicate subscription
	// payment) → 409.
	KindConflict
)

// Error is a typed API error carrying a Kind and a client-facing message.
type Error struct {
	Kind    Kind
	Message string
}

func (e *Error) Error() string { return e.Message }

// Invalid builds a 400 error.
func Invalid(format string, args ...any) *Error {
	return &Error{Kind: KindInvalid, Message: fmt.Sprintf(format, args...)}
}

// NotFound builds a 404 error.
func NotFound(format string, args ...any) *Error {
	return &Error{Kind: KindNotFound, Message: fmt.Sprintf(format, args...)}
}

// Conflict builds a 409 error.
func Conflict(format string, args ...any) *Error {
	return &Error{Kind: KindConflict, Message: fmt.Sprintf(format, args...)}
}
