//go:build integration

package subscription

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
)

func connectTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	host := os.Getenv("DB_HOST")
	if host == "" {
		t.Skip("DB_HOST not set; skipping integration test")
	}
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslrootcert=%s sslmode=verify-full",
		host, os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSL_ROOT_CERT"),
	)
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestCreateWithVersion_And_Resolve(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	ident, err := repo.CreateWithVersion(ctx, "Test Netflix 2099", "#c8403c", 2099, 1, 187_000, 5)
	if err != nil {
		t.Fatalf("CreateWithVersion: %v", err)
	}
	// CASCADE deletes subscription_version rows too.
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM amplop.subscription WHERE id = $1`, ident.ID) //nolint:errcheck
	})

	if ident.Name != "Test Netflix 2099" {
		t.Errorf("Name: got %q, want %q", ident.Name, "Test Netflix 2099")
	}
	if ident.ID == (uuid.UUID{}) {
		t.Error("ID should be non-zero")
	}

	// Visible at its effective month.
	subs, err := repo.Resolve(ctx, 2099, 1)
	if err != nil {
		t.Fatalf("Resolve(2099,1): %v", err)
	}
	found := findResolved(subs, ident.ID)
	if found == nil {
		t.Fatal("subscription not found in Resolve(2099,1)")
	}
	if found.Alloc != 187_000 {
		t.Errorf("Alloc: got %d, want 187000", found.Alloc)
	}
	if found.DueDay != 5 {
		t.Errorf("DueDay: got %d, want 5", found.DueDay)
	}

	// Not visible before its effective month.
	subs, err = repo.Resolve(ctx, 2098, 12)
	if err != nil {
		t.Fatalf("Resolve(2098,12): %v", err)
	}
	if findResolved(subs, ident.ID) != nil {
		t.Error("subscription must not be visible before its effective month")
	}
}

func TestUpsertVersion_EffectiveDating(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	ident, err := repo.CreateWithVersion(ctx, "Test Spotify 2099", "#1db954", 2099, 1, 100_000, 10)
	if err != nil {
		t.Fatalf("CreateWithVersion: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM amplop.subscription WHERE id = $1`, ident.ID) //nolint:errcheck
	})

	// Insert a newer version at (2099, 3) with higher alloc.
	if err := repo.UpsertVersion(ctx, ident.ID, 2099, 3, 200_000, 10, true); err != nil {
		t.Fatalf("UpsertVersion: %v", err)
	}

	cases := []struct {
		month     int
		wantAlloc int64
	}{
		{1, 100_000},  // original version still in effect
		{2, 100_000},  // still original
		{3, 200_000},  // new version takes over
		{12, 200_000}, // new version applies forward
	}
	for _, tc := range cases {
		subs, err := repo.Resolve(ctx, 2099, tc.month)
		if err != nil {
			t.Fatalf("Resolve(2099,%d): %v", tc.month, err)
		}
		r := findResolved(subs, ident.ID)
		if r == nil {
			t.Fatalf("Resolve(2099,%d): subscription not found", tc.month)
		}
		if r.Alloc != tc.wantAlloc {
			t.Errorf("Resolve(2099,%d): Alloc=%d, want %d", tc.month, r.Alloc, tc.wantAlloc)
		}
	}

	// Re-upsert at the same month updates in place.
	if err := repo.UpsertVersion(ctx, ident.ID, 2099, 3, 250_000, 10, true); err != nil {
		t.Fatalf("second UpsertVersion: %v", err)
	}
	subs, err := repo.Resolve(ctx, 2099, 3)
	if err != nil {
		t.Fatalf("Resolve after re-upsert: %v", err)
	}
	if r := findResolved(subs, ident.ID); r == nil || r.Alloc != 250_000 {
		t.Errorf("after re-upsert: Alloc=%v, want 250000", func() any {
			if r := findResolved(subs, ident.ID); r != nil {
				return r.Alloc
			}
			return "not found"
		}())
	}
}

func TestUpsertVersion_SoftEnd(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	ident, err := repo.CreateWithVersion(ctx, "Test iCloud 2099", "#555555", 2099, 1, 50_000, 1)
	if err != nil {
		t.Fatalf("CreateWithVersion: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM amplop.subscription WHERE id = $1`, ident.ID) //nolint:errcheck
	})

	// Soft-delete at (2099, 2).
	if err := repo.UpsertVersion(ctx, ident.ID, 2099, 2, 50_000, 1, false); err != nil {
		t.Fatalf("UpsertVersion soft-end: %v", err)
	}

	// Still visible before the soft-end month.
	subs, err := repo.Resolve(ctx, 2099, 1)
	if err != nil {
		t.Fatalf("Resolve(2099,1): %v", err)
	}
	if findResolved(subs, ident.ID) == nil {
		t.Error("subscription must be visible before soft-end month")
	}

	// Invisible at and after the soft-end month.
	for _, m := range []int{2, 3, 12} {
		subs, err := repo.Resolve(ctx, 2099, m)
		if err != nil {
			t.Fatalf("Resolve(2099,%d): %v", m, err)
		}
		if findResolved(subs, ident.ID) != nil {
			t.Errorf("subscription must not be visible at month 2099-%d after soft-end", m)
		}
	}
}

func TestByID_NotFound(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	_, err := repo.ByID(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindNotFound {
		t.Errorf("expected NotFound, got %v", err)
	}
}

func TestUpdateIdentity(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	ident, err := repo.CreateWithVersion(ctx, "Old Name", "#000000", 2099, 6, 30_000, 15)
	if err != nil {
		t.Fatalf("CreateWithVersion: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM amplop.subscription WHERE id = $1`, ident.ID) //nolint:errcheck
	})

	updated, err := repo.UpdateIdentity(ctx, ident.ID, "New Name", "#ffffff")
	if err != nil {
		t.Fatalf("UpdateIdentity: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name: got %q, want %q", updated.Name, "New Name")
	}
	if updated.Color != "#ffffff" {
		t.Errorf("Color: got %q, want %q", updated.Color, "#ffffff")
	}

	// Resolved still shows the new name.
	subs, err := repo.Resolve(ctx, 2099, 6)
	if err != nil {
		t.Fatalf("Resolve after UpdateIdentity: %v", err)
	}
	r := findResolved(subs, ident.ID)
	if r == nil {
		t.Fatal("subscription not found after UpdateIdentity")
	}
	if r.Name != "New Name" {
		t.Errorf("Resolved Name: got %q, want %q", r.Name, "New Name")
	}
}

func TestListIdentities(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	ident, err := repo.CreateWithVersion(ctx, "ZZZZ Test List 2099", "#aabbcc", 2099, 1, 10_000, 1)
	if err != nil {
		t.Fatalf("CreateWithVersion: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM amplop.subscription WHERE id = $1`, ident.ID) //nolint:errcheck
	})

	all, err := repo.ListIdentities(ctx)
	if err != nil {
		t.Fatalf("ListIdentities: %v", err)
	}
	var found bool
	for _, s := range all {
		if s.ID == ident.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("created subscription not found in ListIdentities")
	}
}

func findResolved(subs []Resolved, id uuid.UUID) *Resolved {
	for i := range subs {
		if subs[i].ID == id {
			return &subs[i]
		}
	}
	return nil
}
