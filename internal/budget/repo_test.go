//go:build integration

package budget

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

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

func TestResolve_Baseline(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	// The 2025-01 baseline is always present; any month ≥ 2025-01 resolves to it
	// (or a later version if one has been upserted, which won't happen here since
	// TestUpsert_EffectiveDating uses year 2099 to avoid collision).
	cfg, err := repo.Resolve(ctx, 2026, 6)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.Monthly == 0 {
		t.Errorf("expected non-zero Monthly, got 0")
	}
	if cfg.ShopWeekly == 0 {
		t.Errorf("expected non-zero ShopWeekly, got 0")
	}
}

func TestResolve_NotFound(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	// No row exists before the 2025-01 baseline.
	_, err := repo.Resolve(ctx, 2024, 12)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindNotFound {
		t.Errorf("expected NotFound, got %v", err)
	}
}

func TestUpsert_EffectiveDating(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	// Use far-future year to avoid collision with real data.
	const testYear = 2099
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM amplop.budget_config WHERE effective_year = $1`, testYear) //nolint:errcheck
	})

	// Insert a new version at (2099, 3).
	if err := repo.Upsert(ctx, testYear, 3, 9_000_000, 900_000, 300_000); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	// Month before the new version resolves to the previous row (not 2099-03 values).
	before, err := repo.Resolve(ctx, testYear, 2)
	if err != nil {
		t.Fatalf("Resolve before: %v", err)
	}
	if before.Monthly == 9_000_000 {
		t.Error("month before new version should not return new values")
	}

	// Exactly at and after the new version returns new values.
	for _, m := range []int{3, 4, 12} {
		cfg, err := repo.Resolve(ctx, testYear, m)
		if err != nil {
			t.Fatalf("Resolve(%d, %d): %v", testYear, m, err)
		}
		if cfg.Monthly != 9_000_000 {
			t.Errorf("month %d: Monthly=%d, want 9000000", m, cfg.Monthly)
		}
		if cfg.ShopWeekly != 900_000 {
			t.Errorf("month %d: ShopWeekly=%d, want 900000", m, cfg.ShopWeekly)
		}
	}

	// Re-upsert at the same month updates the row in place.
	if err := repo.Upsert(ctx, testYear, 3, 8_000_000, 800_000, 250_000); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	updated, err := repo.Resolve(ctx, testYear, 3)
	if err != nil {
		t.Fatalf("Resolve after update: %v", err)
	}
	if updated.Monthly != 8_000_000 {
		t.Errorf("after re-upsert: Monthly=%d, want 8000000", updated.Monthly)
	}
}
