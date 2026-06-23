package expense

import (
	"net/http"
	"time"

	"github.com/ishanwardhono/expense-function/internal/envelope"
	"github.com/ishanwardhono/expense-function/internal/platform/httpx"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// EnvelopeRef is the derived envelope badge on an expense row (spec §7.1).
type EnvelopeRef struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Response is the JSON shape of one expense row (spec §7.1/§7.2). It is reused
// by the month dashboard's day list.
type Response struct {
	ID             string      `json:"id"`
	Date           string      `json:"date"`        // YYYY-MM-DD (day-group key)
	OccurredAt     string      `json:"occurred_at"` // RFC3339 (Asia/Jakarta offset)
	Amount         int64       `json:"amount"`
	Category       string      `json:"category"`
	SubscriptionID *string     `json:"subscription_id"`
	Note           string      `json:"note"`
	Envelope       EnvelopeRef `json:"envelope"`
}

// ToResponse maps a DB expense to its API representation, deriving the envelope
// from category + date (spec §6.1).
func ToResponse(e Expense) Response {
	env := envelope.EnvelopeOf(envelope.Category(e.Category), e.OccurredDate)
	var subID *string
	if e.SubscriptionID != nil {
		s := e.SubscriptionID.String()
		subID = &s
	}
	return Response{
		ID:             e.ID.String(),
		Date:           e.OccurredDate.In(timeutil.Loc).Format("2006-01-02"),
		OccurredAt:     e.OccurredAt().Format(time.RFC3339),
		Amount:         e.Amount,
		Category:       e.Category,
		SubscriptionID: subID,
		Note:           e.Note,
		Envelope:       EnvelopeRef{ID: string(env), Label: env.ShortLabel()},
	}
}

// Handler serves the expense endpoints (spec §7.2).
type Handler struct {
	svc *Service
}

// NewHandler builds an expense Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Create handles POST /expenses.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) error {
	var req WriteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	e, err := h.svc.Create(r.Context(), req)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, ToResponse(e))
	return nil
}

// Update handles PUT /expenses/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.PathUUID(r, "id")
	if err != nil {
		return err
	}
	var req WriteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		return err
	}
	e, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, ToResponse(e))
	return nil
}

// Delete handles DELETE /expenses/{id} → 204.
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
