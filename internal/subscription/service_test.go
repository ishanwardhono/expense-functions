package subscription

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
	"github.com/ishanwardhono/expense-function/internal/platform/httpx"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// fakeRepo is an in-memory subscription repository.
type fakeRepo struct {
	idents   map[uuid.UUID]Identity
	versions map[uuid.UUID][]Version // per subscription, append-only
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{idents: map[uuid.UUID]Identity{}, versions: map[uuid.UUID][]Version{}}
}

func (f *fakeRepo) ByID(_ context.Context, id uuid.UUID) (Identity, error) {
	ident, ok := f.idents[id]
	if !ok {
		return Identity{}, apierr.NotFound("subscription %s not found", id)
	}
	return ident, nil
}

func (f *fakeRepo) Resolve(_ context.Context, year, month int) ([]Resolved, error) {
	var out []Resolved
	for id, ident := range f.idents {
		v, err := f.LatestVersion(context.Background(), id, year, month)
		if err != nil || !v.Active {
			continue
		}
		out = append(out, Resolved{ID: id, Name: ident.Name, Color: ident.Color, Alloc: v.Alloc, DueDay: v.DueDay})
	}
	return out, nil
}

func (f *fakeRepo) LatestVersion(_ context.Context, id uuid.UUID, year, month int) (Version, error) {
	best := Version{}
	found := false
	for _, v := range f.versions[id] {
		y, m := int(v.EffectiveYear), int(v.EffectiveMonth)
		if y < year || (y == year && m <= month) {
			if !found || y > int(best.EffectiveYear) || (y == int(best.EffectiveYear) && m > int(best.EffectiveMonth)) {
				best = v
				found = true
			}
		}
	}
	if !found {
		return Version{}, apierr.NotFound("subscription %s has no version for %d-%02d", id, year, month)
	}
	return best, nil
}

func (f *fakeRepo) CreateWithVersion(_ context.Context, name, color string, year, month int, alloc int64, dueDay int16) (Identity, error) {
	id := uuid.New()
	ident := Identity{ID: id, Name: name, Color: color}
	f.idents[id] = ident
	f.versions[id] = append(f.versions[id], Version{
		SubscriptionID: id, EffectiveYear: int16(year), EffectiveMonth: int16(month),
		Alloc: alloc, DueDay: dueDay, Active: true,
	})
	return ident, nil
}

func (f *fakeRepo) UpdateIdentity(_ context.Context, id uuid.UUID, name, color string) (Identity, error) {
	ident, ok := f.idents[id]
	if !ok {
		return Identity{}, apierr.NotFound("subscription %s not found", id)
	}
	ident.Name, ident.Color = name, color
	f.idents[id] = ident
	return ident, nil
}

func (f *fakeRepo) UpsertVersion(_ context.Context, id uuid.UUID, year, month int, alloc int64, dueDay int16, active bool) error {
	for i, v := range f.versions[id] {
		if int(v.EffectiveYear) == year && int(v.EffectiveMonth) == month {
			f.versions[id][i].Alloc = alloc
			f.versions[id][i].DueDay = dueDay
			f.versions[id][i].Active = active
			return nil
		}
	}
	f.versions[id] = append(f.versions[id], Version{
		SubscriptionID: id, EffectiveYear: int16(year), EffectiveMonth: int16(month),
		Alloc: alloc, DueDay: dueDay, Active: active,
	})
	return nil
}

func fixedClock(y, m, d int) timeutil.Clock {
	return func() time.Time { return timeutil.Date(y, m, d) }
}

func TestCreate_Validation(t *testing.T) {
	svc := NewService(newFakeRepo(), fixedClock(2026, 6, 23))
	cases := []CreateRequest{
		{Name: "", Alloc: 1000, DueDay: 5},
		{Name: "Netflix", Alloc: 0, DueDay: 5},
		{Name: "Netflix", Alloc: 1000, DueDay: 0},
		{Name: "Netflix", Alloc: 1000, DueDay: 32},
	}
	for i, req := range cases {
		_, err := svc.Create(context.Background(), req)
		var ae *apierr.Error
		if !errors.As(err, &ae) || ae.Kind != apierr.KindInvalid {
			t.Errorf("case %d: expected Invalid, got %v", i, err)
		}
	}
}

func TestCreate_Then_Resolve(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fixedClock(2026, 6, 23))
	sub, err := svc.Create(context.Background(), CreateRequest{Name: "Netflix", Color: "#c8403c", Alloc: 187_000, DueDay: 5})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sub.ID == (uuid.UUID{}) || sub.Alloc != 187_000 {
		t.Fatalf("unexpected created sub: %+v", sub)
	}
	got, err := svc.Resolve(context.Background(), 2026, 6)
	if err != nil || len(got) != 1 {
		t.Fatalf("Resolve: got %d subs (%v)", len(got), err)
	}
	// Created in June: absent from May (its earliest version > May).
	may, _ := svc.Resolve(context.Background(), 2026, 5)
	if len(may) != 0 {
		t.Errorf("subscription should be absent from May, got %d", len(may))
	}
}

func TestUpdate_PartialPreservesOtherField(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fixedClock(2026, 6, 23))
	sub, _ := svc.Create(context.Background(), CreateRequest{Name: "Netflix", Alloc: 187_000, DueDay: 5})

	newAlloc := int64(200_000)
	updated, err := svc.Update(context.Background(), sub.ID, UpdateRequest{Alloc: &newAlloc})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Alloc != 200_000 {
		t.Errorf("alloc: got %d, want 200000", updated.Alloc)
	}
	if updated.DueDay != 5 {
		t.Errorf("due_day should be preserved at 5, got %d", updated.DueDay)
	}

	newName := "Netflix Premium"
	renamed, err := svc.Update(context.Background(), sub.ID, UpdateRequest{Name: &newName})
	if err != nil {
		t.Fatalf("Update name: %v", err)
	}
	if renamed.Name != "Netflix Premium" || renamed.Alloc != 200_000 {
		t.Errorf("rename should preserve alloc 200000: %+v", renamed)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	svc := NewService(newFakeRepo(), fixedClock(2026, 6, 23))
	name := "x"
	_, err := svc.Update(context.Background(), uuid.New(), UpdateRequest{Name: &name})
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestDelete_SoftEndKeepsPastMonths(t *testing.T) {
	repo := newFakeRepo()
	// Create in May 2026.
	svcMay := NewService(repo, fixedClock(2026, 5, 10))
	sub, _ := svcMay.Create(context.Background(), CreateRequest{Name: "Netflix", Alloc: 187_000, DueDay: 5})

	// Delete in June 2026.
	svcJun := NewService(repo, fixedClock(2026, 6, 10))
	if err := svcJun.Delete(context.Background(), sub.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// May still shows it (frozen); June onward does not.
	may, _ := svcJun.Resolve(context.Background(), 2026, 5)
	if len(may) != 1 {
		t.Errorf("May should still show the subscription, got %d", len(may))
	}
	jun, _ := svcJun.Resolve(context.Background(), 2026, 6)
	if len(jun) != 0 {
		t.Errorf("June should not show the soft-ended subscription, got %d", len(jun))
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := NewService(newFakeRepo(), fixedClock(2026, 6, 23))
	err := svc.Delete(context.Background(), uuid.New())
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestHandler_Update_BadUUID(t *testing.T) {
	h := NewHandler(NewService(newFakeRepo(), fixedClock(2026, 6, 23)), fixedClock(2026, 6, 23))
	// Route through the real router so the {id} path param is populated.
	rt := httpx.NewRouter()
	rt.Handle(http.MethodPut, "/subscriptions/{id}", h.Update)
	req := httptest.NewRequest(http.MethodPut, "/subscriptions/not-a-uuid", strings.NewReader(`{}`))
	err := rt.ServeHTTP(httptest.NewRecorder(), req)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindInvalid {
		t.Fatalf("expected Invalid, got %v", err)
	}
}

func TestHandler_List_OK(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo, fixedClock(2026, 6, 23))
	_, _ = svc.Create(context.Background(), CreateRequest{Name: "Netflix", Alloc: 187_000, DueDay: 5})
	h := NewHandler(svc, fixedClock(2026, 6, 23))

	req := httptest.NewRequest(http.MethodGet, "/subscriptions", nil)
	rec := httptest.NewRecorder()
	if err := h.List(rec, req); err != nil {
		t.Fatalf("List: %v", err)
	}
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Netflix") {
		t.Errorf("unexpected response %d %s", rec.Code, rec.Body.String())
	}
}
