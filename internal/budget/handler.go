package budget

import (
	"net/http"

	"github.com/ishanwardhono/expense-function/internal/platform/httpx"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// Handler serves the budget config endpoints (spec §7.4).
type Handler struct {
	svc *Service
	now timeutil.Clock
}

// NewHandler builds a budget Handler. now defaults to timeutil.Now when nil.
func NewHandler(svc *Service, now timeutil.Clock) *Handler {
	if now == nil {
		now = timeutil.Now
	}
	return &Handler{svc: svc, now: now}
}

// Response is the JSON shape returned for a resolved budget config.
type Response struct {
	EffectiveYear  int16 `json:"effective_year"`
	EffectiveMonth int16 `json:"effective_month"`
	Monthly        int64 `json:"monthly"`
	ShopWeekly     int64 `json:"shop_weekly"`
	WeekendBudget  int64 `json:"weekend_budget"`
}

func toResponse(c Config) Response {
	return Response{
		EffectiveYear:  c.EffectiveYear,
		EffectiveMonth: c.EffectiveMonth,
		Monthly:        c.Monthly,
		ShopWeekly:     c.ShopWeekly,
		WeekendBudget:  c.WeekendBudget,
	}
}

// Get handles GET /budget?year=&month= (defaults to the current month).
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) error {
	year, month := timeutil.CurrentMonth(h.now())
	year, month, err := httpx.QueryYearMonth(r, year, month)
	if err != nil {
		return err
	}
	cfg, err := h.svc.Resolve(r.Context(), year, month)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(cfg))
	return nil
}

// Put handles PUT /budget — upserts a config effective from the current month.
func (h *Handler) Put(w http.ResponseWriter, r *http.Request) error {
	var req UpdateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	cfg, err := h.svc.Update(r.Context(), req)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(cfg))
	return nil
}
