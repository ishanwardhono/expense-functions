package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
)

// DecodeJSON decodes the request body into v, returning an apierr.Invalid (400)
// for malformed or empty JSON so handlers stay thin (spec §4.3).
func DecodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return apierr.Invalid("request body is required")
	}
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return apierr.Invalid("request body is required")
		}
		return apierr.Invalid("invalid JSON body: %v", err)
	}
	return nil
}

// QueryYearMonth reads the year/month query params, falling back to the given
// defaults (the current Asia/Jakarta month) when a param is absent. A malformed
// value or an out-of-range month returns an apierr.Invalid (400).
func QueryYearMonth(r *http.Request, defYear, defMonth int) (year, month int, err error) {
	year, month = defYear, defMonth
	q := r.URL.Query()
	if v := q.Get("year"); v != "" {
		year, err = strconv.Atoi(v)
		if err != nil {
			return 0, 0, apierr.Invalid("invalid year %q", v)
		}
	}
	if v := q.Get("month"); v != "" {
		month, err = strconv.Atoi(v)
		if err != nil {
			return 0, 0, apierr.Invalid("invalid month %q", v)
		}
	}
	if month < 1 || month > 12 {
		return 0, 0, apierr.Invalid("month must be between 1 and 12, got %d", month)
	}
	return year, month, nil
}

// PathUUID reads a {name} path param and parses it as a UUID, returning an
// apierr.Invalid (400) when it is absent or malformed.
func PathUUID(r *http.Request, name string) (uuid.UUID, error) {
	raw := Param(r, name)
	if raw == "" {
		return uuid.UUID{}, apierr.Invalid("missing %s", name)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, apierr.Invalid("invalid %s %q", name, raw)
	}
	return id, nil
}
