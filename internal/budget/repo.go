package budget

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
)

// Repo provides access to the amplop.budget_config table.
type Repo struct {
	db *sqlx.DB
}

// NewRepo creates a new Repo backed by db.
func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

// Resolve returns the effective budget config for (year, month) using the §5.1
// tuple-comparison resolution query. The 2025-01 baseline ensures a row always
// exists for any viewable month; NotFound is returned as a defensive fallback.
func (r *Repo) Resolve(ctx context.Context, year, month int) (Config, error) {
	const q = `
SELECT id, effective_year, effective_month, monthly, shop_weekly, weekend_budget,
       created_at, updated_at
FROM amplop.budget_config
WHERE (effective_year, effective_month) <= ($1, $2)
ORDER BY effective_year DESC, effective_month DESC
LIMIT 1`
	var cfg Config
	if err := r.db.GetContext(ctx, &cfg, q, year, month); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Config{}, apierr.NotFound("no budget config for %d-%02d", year, month)
		}
		return Config{}, fmt.Errorf("budget resolve %d-%02d: %w", year, month, err)
	}
	return cfg, nil
}

// Upsert writes a budget config version effective from (year, month), per §5.2.
// Calling this with the current month freezes past months and applies forward.
func (r *Repo) Upsert(ctx context.Context, year, month int, monthly, shopWeekly, weekendBudget int64) error {
	const q = `
INSERT INTO amplop.budget_config
    (effective_year, effective_month, monthly, shop_weekly, weekend_budget, updated_at)
VALUES ($1, $2, $3, $4, $5, current_timestamp())
ON CONFLICT (effective_year, effective_month)
DO UPDATE SET
    monthly        = EXCLUDED.monthly,
    shop_weekly    = EXCLUDED.shop_weekly,
    weekend_budget = EXCLUDED.weekend_budget,
    updated_at     = current_timestamp()`
	if _, err := r.db.ExecContext(ctx, q, year, month, monthly, shopWeekly, weekendBudget); err != nil {
		return fmt.Errorf("budget upsert %d-%02d: %w", year, month, err)
	}
	return nil
}
