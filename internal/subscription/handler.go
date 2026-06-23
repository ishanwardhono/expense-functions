package subscription

import (
	"net/http"

	"github.com/ishanwardhono/expense-function/internal/platform/httpx"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// Handler serves the subscription definition endpoints (spec §7.3).
type Handler struct {
	svc *Service
	now timeutil.Clock
}

// NewHandler builds a subscription Handler. now defaults to timeutil.Now when nil.
func NewHandler(svc *Service, now timeutil.Clock) *Handler {
	if now == nil {
		now = timeutil.Now
	}
	return &Handler{svc: svc, now: now}
}

// Response is the JSON shape of a resolved subscription.
type Response struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Color  string `json:"color"`
	Alloc  int64  `json:"alloc"`
	DueDay int16  `json:"due_day"`
}

func toResponse(r Resolved) Response {
	return Response{
		ID:     r.ID.String(),
		Name:   r.Name,
		Color:  r.Color,
		Alloc:  r.Alloc,
		DueDay: r.DueDay,
	}
}

// List handles GET /subscriptions?year=&month= (defaults to the current month).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) error {
	year, month := timeutil.CurrentMonth(h.now())
	year, month, err := httpx.QueryYearMonth(r, year, month)
	if err != nil {
		return err
	}
	subs, err := h.svc.Resolve(r.Context(), year, month)
	if err != nil {
		return err
	}
	out := make([]Response, 0, len(subs))
	for _, s := range subs {
		out = append(out, toResponse(s))
	}
	httpx.WriteJSON(w, http.StatusOK, out)
	return nil
}

// Create handles POST /subscriptions.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	var req CreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	sub, err := h.svc.Create(r.Context(), req)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, toResponse(sub))
	return nil
}

// Update handles PUT /subscriptions/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathUUID(r, "id")
	if err != nil {
		return err
	}
	var req UpdateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	sub, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(sub))
	return nil
}

// Delete handles DELETE /subscriptions/{id} → 204.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathUUID(r, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusNoContent, nil)
	return nil
}
