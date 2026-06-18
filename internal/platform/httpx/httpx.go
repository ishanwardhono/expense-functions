// Package httpx provides the shared HTTP layer for the single routed Expense
// function: a small method+path router, CORS, panic recovery, JSON helpers, and
// the typed-error→status mapping (spec §4.3).
package httpx

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
)

// HandlerFunc is a route handler. It writes the success response itself (via
// WriteJSON / w.WriteHeader) and returns an error to be mapped to a status by
// the middleware. Returning nil means the handler already wrote the response.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// WriteJSON writes v as a JSON body with the given status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpx: encode response: %v", err)
	}
}

// WriteError maps an error to a status code and writes {"error":"message"}.
// For unexpected (500) errors the real detail is logged server-side and only a
// generic message is returned, so internals (DB driver text, SQL, etc.) are not
// exposed to callers — relevant under CORS: *.
func WriteError(w http.ResponseWriter, err error) {
	status, msg := statusFor(err)
	if status == http.StatusInternalServerError {
		log.Printf("httpx: internal error: %v", err)
	}
	WriteJSON(w, status, map[string]string{"error": msg})
}

// statusFor maps typed errors to HTTP status codes (spec §4.3).
func statusFor(err error) (int, string) {
	var ae *apierr.Error
	if errors.As(err, &ae) {
		switch ae.Kind {
		case apierr.KindInvalid:
			return http.StatusBadRequest, ae.Message
		case apierr.KindNotFound:
			return http.StatusNotFound, ae.Message
		case apierr.KindConflict:
			return http.StatusConflict, ae.Message
		}
	}
	return http.StatusInternalServerError, "internal server error"
}

// Middleware wraps a HandlerFunc with CORS, panic recovery, and error mapping.
// It is the single entry point registered as the Cloud Function.
func Middleware(next HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("httpx: panic recovered: %v", rec)
				WriteError(w, errors.New("internal server error"))
			}
		}()

		if err := next(w, r); err != nil {
			WriteError(w, err)
		}
	}
}
