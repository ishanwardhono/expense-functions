package budget

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// fakeRepo is an in-memory budget repository keyed by (year, month).
type fakeRepo struct {
	rows      map[[2]int]Config
	upsertErr error
}

func newFakeRepo() *fakeRepo { return &fakeRepo{rows: map[[2]int]Config{}} }

func (f *fakeRepo) Resolve(_ context.Context, year, month int) (Config, error) {
	// Find the greatest (y, m) <= (year, month).
	best := [2]int{-1, -1}
	for k := range f.rows {
		if k[0] < year || (k[0] == year && k[1] <= month) {
			if k[0] > best[0] || (k[0] == best[0] && k[1] > best[1]) {
				best = k
			}
		}
	}
	if best[0] == -1 {
		return Config{}, apierr.NotFound("no budget config for %d-%02d", year, month)
	}
	return f.rows[best], nil
}

func (f *fakeRepo) Upsert(_ context.Context, year, month int, monthly, shopWeekly, weekendBudget int64) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.rows[[2]int{year, month}] = Config{
		EffectiveYear: int16(year), EffectiveMonth: int16(month),
		Monthly: monthly, ShopWeekly: shopWeekly, WeekendBudget: weekendBudget,
	}
	return nil
}

func fixedClock(y, m, d int) timeutil.Clock {
	return func() time.Time { return timeutil.Date(y, m, d) }
}

func TestUpdate_StampsCurrentMonthAndResolves(t *testing.T) {
	repo := newFakeRepo()
	repo.rows[[2]int{2025, 1}] = Config{EffectiveYear: 2025, EffectiveMonth: 1, Monthly: 5_000_000, ShopWeekly: 600_000, WeekendBudget: 200_000}
	svc := NewService(repo, fixedClock(2026, 6, 23))

	cfg, err := svc.Update(context.Background(), UpdateRequest{Monthly: 7_000_000, ShopWeekly: 700_000, WeekendBudget: 250_000})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if cfg.EffectiveYear != 2026 || cfg.EffectiveMonth != 6 {
		t.Errorf("effective month: got %d-%d, want 2026-6", cfg.EffectiveYear, cfg.EffectiveMonth)
	}
	if cfg.Monthly != 7_000_000 {
		t.Errorf("monthly: got %d, want 7000000", cfg.Monthly)
	}

	// A past month still resolves to the 2025-01 baseline (frozen).
	past, err := svc.Resolve(context.Background(), 2026, 5)
	if err != nil {
		t.Fatalf("Resolve past: %v", err)
	}
	if past.Monthly != 5_000_000 {
		t.Errorf("past monthly: got %d, want 5000000 (frozen)", past.Monthly)
	}
}

func TestUpdate_NegativeRejected(t *testing.T) {
	svc := NewService(newFakeRepo(), fixedClock(2026, 6, 23))
	_, err := svc.Update(context.Background(), UpdateRequest{Monthly: -1})
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindInvalid {
		t.Fatalf("expected Invalid, got %v", err)
	}
}

func TestHandler_Put_BadJSON(t *testing.T) {
	h := NewHandler(NewService(newFakeRepo(), fixedClock(2026, 6, 23)), fixedClock(2026, 6, 23))
	req := httptest.NewRequest(http.MethodPut, "/budget", strings.NewReader("{not json"))
	err := h.Put(httptest.NewRecorder(), req)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindInvalid {
		t.Fatalf("expected Invalid, got %v", err)
	}
}

func TestHandler_Get_DefaultsToCurrentMonth(t *testing.T) {
	repo := newFakeRepo()
	repo.rows[[2]int{2025, 1}] = Config{EffectiveYear: 2025, EffectiveMonth: 1, Monthly: 5_000_000}
	h := NewHandler(NewService(repo, fixedClock(2026, 6, 23)), fixedClock(2026, 6, 23))

	req := httptest.NewRequest(http.MethodGet, "/budget", nil)
	rec := httptest.NewRecorder()
	if err := h.Get(rec, req); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"monthly":5000000`) {
		t.Errorf("body missing monthly: %s", rec.Body.String())
	}
}

func TestHandler_Get_BadMonth(t *testing.T) {
	h := NewHandler(NewService(newFakeRepo(), fixedClock(2026, 6, 23)), fixedClock(2026, 6, 23))
	req := httptest.NewRequest(http.MethodGet, "/budget?month=13", nil)
	err := h.Get(httptest.NewRecorder(), req)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindInvalid {
		t.Fatalf("expected Invalid, got %v", err)
	}
}
