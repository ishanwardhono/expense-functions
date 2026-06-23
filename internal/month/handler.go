package month

import (
	"net/http"

	"github.com/ishanwardhono/expense-function/internal/platform/httpx"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// Handler serves GET /month (spec §7.1).
type Handler struct {
	svc *Service
	now timeutil.Clock
}

// NewHandler builds a month Handler. now defaults to timeutil.Now when nil.
func NewHandler(svc *Service, now timeutil.Clock) *Handler {
	if now == nil {
		now = timeutil.Now
	}
	return &Handler{svc: svc, now: now}
}

// Get handles GET /month?year=&month= (defaults to the current month).
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) error {
	year, month := timeutil.CurrentMonth(h.now())
	year, month, err := httpx.QueryYearMonth(r, year, month)
	if err != nil {
		return err
	}
	dash, err := h.svc.Dashboard(r.Context(), year, month)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, dash)
	return nil
}
