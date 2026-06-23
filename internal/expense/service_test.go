package expense

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
	"github.com/ishanwardhono/expense-function/internal/platform/httpx"
)

// fakeRepo is an in-memory expense repository.
type fakeRepo struct {
	byID map[uuid.UUID]Expense
}

func newFakeRepo() *fakeRepo { return &fakeRepo{byID: map[uuid.UUID]Expense{}} }

func (f *fakeRepo) ByID(_ context.Context, id uuid.UUID) (Expense, error) {
	e, ok := f.byID[id]
	if !ok {
		return Expense{}, apierr.NotFound("expense %s not found", id)
	}
	return e, nil
}

func (f *fakeRepo) Create(_ context.Context, e Expense) (Expense, error) {
	e.ID = uuid.New()
	e.OccurredYear = int16(e.OccurredDate.Year())
	e.OccurredMonth = int16(e.OccurredDate.Month())
	f.byID[e.ID] = e
	return e, nil
}

func (f *fakeRepo) Update(_ context.Context, e Expense) (Expense, error) {
	if _, ok := f.byID[e.ID]; !ok {
		return Expense{}, apierr.NotFound("expense %s not found", e.ID)
	}
	e.OccurredYear = int16(e.OccurredDate.Year())
	e.OccurredMonth = int16(e.OccurredDate.Month())
	f.byID[e.ID] = e
	return e, nil
}

func (f *fakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return apierr.NotFound("expense %s not found", id)
	}
	delete(f.byID, id)
	return nil
}

func (f *fakeRepo) ExistsForSubscriptionMonth(_ context.Context, subID uuid.UUID, year, month int, excludeID *uuid.UUID) (bool, error) {
	for id, e := range f.byID {
		if excludeID != nil && id == *excludeID {
			continue
		}
		if e.SubscriptionID != nil && *e.SubscriptionID == subID &&
			int(e.OccurredYear) == year && int(e.OccurredMonth) == month {
			return true, nil
		}
	}
	return false, nil
}

type fakeSubs struct{ exists map[uuid.UUID]bool }

func (f fakeSubs) Exists(_ context.Context, id uuid.UUID) (bool, error) {
	return f.exists[id], nil
}

func newService(repo *fakeRepo, subIDs ...uuid.UUID) *Service {
	m := map[uuid.UUID]bool{}
	for _, id := range subIDs {
		m[id] = true
	}
	return NewService(repo, fakeSubs{exists: m})
}

func TestCreate_Validation(t *testing.T) {
	subID := uuid.New()
	svc := newService(newFakeRepo(), subID)
	subStr := subID.String()
	other := uuid.New().String()
	bad := "12:99"

	cases := map[string]WriteRequest{
		"zero amount":        {Date: "2026-06-15", Amount: 0, Category: "Makan"},
		"bad category":       {Date: "2026-06-15", Amount: 1000, Category: "Nope"},
		"bad date":           {Date: "15-06-2026", Amount: 1000, Category: "Makan"},
		"bad time":           {Date: "2026-06-15", Amount: 1000, Category: "Makan", Time: &bad},
		"langganan no sub":   {Date: "2026-06-15", Amount: 1000, Category: "Langganan"},
		"non-langg with sub": {Date: "2026-06-15", Amount: 1000, Category: "Makan", SubscriptionID: &subStr},
		"unknown sub":        {Date: "2026-06-15", Amount: 1000, Category: "Langganan", SubscriptionID: &other},
	}
	for name, req := range cases {
		_, err := svc.Create(context.Background(), req)
		var ae *apierr.Error
		if !errors.As(err, &ae) || ae.Kind != apierr.KindInvalid {
			t.Errorf("%s: expected Invalid, got %v", name, err)
		}
	}
}

func TestCreate_HappyNonLangganan(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)
	tm := "12:10"
	e, err := svc.Create(context.Background(), WriteRequest{
		Date: "2026-06-15", Time: &tm, Amount: 18_000, Category: "Makan", Note: "nasi padang",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if e.ID == (uuid.UUID{}) {
		t.Error("expected non-zero ID")
	}
	at := e.OccurredAt()
	if at.Hour() != 12 || at.Minute() != 10 {
		t.Errorf("occurred_at time: got %v, want 12:10", at)
	}
}

func TestCreate_LanggananOncePerMonthConflict(t *testing.T) {
	repo := newFakeRepo()
	subID := uuid.New()
	svc := newService(repo, subID)
	subStr := subID.String()

	first := WriteRequest{Date: "2026-06-05", Amount: 186_000, Category: "Langganan", SubscriptionID: &subStr}
	if _, err := svc.Create(context.Background(), first); err != nil {
		t.Fatalf("first Langganan: %v", err)
	}
	// Second payment, same sub + month → 409.
	_, err := svc.Create(context.Background(), WriteRequest{
		Date: "2026-06-20", Amount: 186_000, Category: "Langganan", SubscriptionID: &subStr,
	})
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindConflict {
		t.Fatalf("expected Conflict, got %v", err)
	}
	// A different month is allowed.
	if _, err := svc.Create(context.Background(), WriteRequest{
		Date: "2026-07-05", Amount: 186_000, Category: "Langganan", SubscriptionID: &subStr,
	}); err != nil {
		t.Errorf("July payment should succeed, got %v", err)
	}
}

func TestUpdate_ExcludesSelfForOncePerMonth(t *testing.T) {
	repo := newFakeRepo()
	subID := uuid.New()
	svc := newService(repo, subID)
	subStr := subID.String()

	e, err := svc.Create(context.Background(), WriteRequest{
		Date: "2026-06-05", Amount: 186_000, Category: "Langganan", SubscriptionID: &subStr,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Re-saving the same row (amount changed) must not conflict with itself.
	updated, err := svc.Update(context.Background(), e.ID, WriteRequest{
		Date: "2026-06-05", Amount: 190_000, Category: "Langganan", SubscriptionID: &subStr,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Amount != 190_000 {
		t.Errorf("amount: got %d, want 190000", updated.Amount)
	}
}

func TestUpdate_UnknownID(t *testing.T) {
	svc := newService(newFakeRepo())
	_, err := svc.Update(context.Background(), uuid.New(), WriteRequest{
		Date: "2026-06-05", Amount: 1000, Category: "Makan",
	})
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestDelete_UnknownID(t *testing.T) {
	svc := newService(newFakeRepo())
	err := svc.Delete(context.Background(), uuid.New())
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestHandler_Create_OK(t *testing.T) {
	h := NewHandler(newService(newFakeRepo()))
	body := `{"date":"2026-06-15","amount":18000,"category":"Makan","note":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader(body))
	rec := httptest.NewRecorder()
	if err := h.Create(rec, req); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d, want 201", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"envelope":{"id":"belanja","label":"BLNJ"}`) {
		t.Errorf("body missing envelope badge: %s", rec.Body.String())
	}
}

func TestHandler_Create_BadJSON(t *testing.T) {
	h := NewHandler(newService(newFakeRepo()))
	req := httptest.NewRequest(http.MethodPost, "/expenses", strings.NewReader("{bad"))
	err := h.Create(httptest.NewRecorder(), req)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindInvalid {
		t.Fatalf("expected Invalid, got %v", err)
	}
}

func TestHandler_Delete_UnknownID_RoutedNotFound(t *testing.T) {
	h := NewHandler(newService(newFakeRepo()))
	rt := httpx.NewRouter()
	rt.Handle(http.MethodDelete, "/expenses/{id}", h.Delete)
	req := httptest.NewRequest(http.MethodDelete, "/expenses/"+uuid.New().String(), nil)
	err := rt.ServeHTTP(httptest.NewRecorder(), req)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}
