package subscription

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
)

// Repo provides access to amplop.subscription and amplop.subscription_version.
type Repo struct {
	db *sqlx.DB
}

// NewRepo creates a new Repo backed by db.
func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

// ListIdentities returns all subscription identity rows ordered by name.
func (r *Repo) ListIdentities(ctx context.Context) ([]Identity, error) {
	const q = `
SELECT id, name, color, created_at, updated_at
FROM amplop.subscription
ORDER BY name`
	var out []Identity
	if err := r.db.SelectContext(ctx, &out, q); err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	return out, nil
}

// ByID returns the subscription identity for id.
func (r *Repo) ByID(ctx context.Context, id uuid.UUID) (Identity, error) {
	const q = `
SELECT id, name, color, created_at, updated_at
FROM amplop.subscription
WHERE id = $1`
	var s Identity
	if err := r.db.GetContext(ctx, &s, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Identity{}, apierr.NotFound("subscription %s not found", id)
		}
		return Identity{}, fmt.Errorf("subscription by id %s: %w", id, err)
	}
	return s, nil
}

// Exists reports whether a subscription identity with id exists. Used by the
// expense service to validate a Langganan expense's subscription_id (spec §7.5).
func (r *Repo) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	const q = `SELECT EXISTS (SELECT 1 FROM amplop.subscription WHERE id = $1)`
	var exists bool
	if err := r.db.GetContext(ctx, &exists, q, id); err != nil {
		return false, fmt.Errorf("subscription exists %s: %w", id, err)
	}
	return exists, nil
}

// LatestVersion returns the most recent version with effective month ≤ (year,
// month), regardless of active state. Used to merge partial edits and to carry
// alloc/due_day forward when soft-ending a subscription (spec §5.2). Returns a
// NotFound when no version exists at or before the month.
func (r *Repo) LatestVersion(ctx context.Context, subscriptionID uuid.UUID, year, month int) (Version, error) {
	const q = `
SELECT id, subscription_id, effective_year, effective_month, alloc, due_day, active,
       created_at, updated_at
FROM amplop.subscription_version
WHERE subscription_id = $1
  AND (effective_year, effective_month) <= ($2, $3)
ORDER BY effective_year DESC, effective_month DESC
LIMIT 1`
	var v Version
	if err := r.db.GetContext(ctx, &v, q, subscriptionID, year, month); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Version{}, apierr.NotFound("subscription %s has no version for %d-%02d", subscriptionID, year, month)
		}
		return Version{}, fmt.Errorf("latest version %s %d-%02d: %w", subscriptionID, year, month, err)
	}
	return v, nil
}

// Resolve returns the active subscription set for (year, month) using the §5.1
// lateral join query. Returns an empty slice when no active subscriptions exist.
func (r *Repo) Resolve(ctx context.Context, year, month int) ([]Resolved, error) {
	const q = `
SELECT s.id, s.name, s.color, v.alloc, v.due_day
FROM amplop.subscription s
JOIN LATERAL (
    SELECT alloc, due_day, active
    FROM amplop.subscription_version v
    WHERE v.subscription_id = s.id
      AND (v.effective_year, v.effective_month) <= ($1, $2)
    ORDER BY v.effective_year DESC, v.effective_month DESC
    LIMIT 1
) v ON true
WHERE v.active = true`
	var out []Resolved
	if err := r.db.SelectContext(ctx, &out, q, year, month); err != nil {
		return nil, fmt.Errorf("resolve subscriptions %d-%02d: %w", year, month, err)
	}
	return out, nil
}

// CreateWithVersion inserts a subscription identity and its first effective-dated
// version in a single transaction. year and month should be the current month
// (from timeutil.CurrentMonth).
func (r *Repo) CreateWithVersion(ctx context.Context, name, color string, year, month int, alloc int64, dueDay int16) (Identity, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return Identity{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	const insertIdentity = `
INSERT INTO amplop.subscription (name, color)
VALUES ($1, $2)
RETURNING id, name, color, created_at, updated_at`
	var ident Identity
	if err := tx.GetContext(ctx, &ident, insertIdentity, name, color); err != nil {
		return Identity{}, fmt.Errorf("insert subscription: %w", err)
	}

	const insertVersion = `
INSERT INTO amplop.subscription_version
    (subscription_id, effective_year, effective_month, alloc, due_day, active)
VALUES ($1, $2, $3, $4, $5, true)`
	if _, err := tx.ExecContext(ctx, insertVersion, ident.ID, year, month, alloc, dueDay); err != nil {
		return Identity{}, fmt.Errorf("insert subscription version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Identity{}, fmt.Errorf("commit subscription create: %w", err)
	}
	return ident, nil
}

// UpdateIdentity updates the cosmetic name and color of a subscription. These
// apply to all months without versioning, per §5.2.
func (r *Repo) UpdateIdentity(ctx context.Context, id uuid.UUID, name, color string) (Identity, error) {
	const q = `
UPDATE amplop.subscription
SET name = $2, color = $3, updated_at = current_timestamp()
WHERE id = $1
RETURNING id, name, color, created_at, updated_at`
	var s Identity
	if err := r.db.GetContext(ctx, &s, q, id, name, color); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Identity{}, apierr.NotFound("subscription %s not found", id)
		}
		return Identity{}, fmt.Errorf("update subscription %s: %w", id, err)
	}
	return s, nil
}

// UpsertVersion writes an effective-dated version for (subscriptionID, year, month),
// per §5.2. Pass active=false to soft-delete a subscription from the current month
// onward. year and month should be the current month (from timeutil.CurrentMonth).
func (r *Repo) UpsertVersion(ctx context.Context, subscriptionID uuid.UUID, year, month int, alloc int64, dueDay int16, active bool) error {
	const q = `
INSERT INTO amplop.subscription_version
    (subscription_id, effective_year, effective_month, alloc, due_day, active, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, current_timestamp())
ON CONFLICT (subscription_id, effective_year, effective_month)
DO UPDATE SET
    alloc      = EXCLUDED.alloc,
    due_day    = EXCLUDED.due_day,
    active     = EXCLUDED.active,
    updated_at = current_timestamp()`
	if _, err := r.db.ExecContext(ctx, q, subscriptionID, year, month, alloc, dueDay, active); err != nil {
		return fmt.Errorf("upsert subscription version %s %d-%02d: %w", subscriptionID, year, month, err)
	}
	return nil
}
