package expense

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

const selectCols = `id, occurred_date, occurred_time, amount, category, subscription_id,
       note, occurred_year, occurred_month, created_at, updated_at`

// Repo provides access to the amplop.expense table.
type Repo struct {
	db *sqlx.DB
}

// NewRepo creates a new Repo backed by db.
func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

// ForMonth returns all expenses in the wide boundary window
// [firstOfMonth−7d, lastOfMonth+7d] for (year, month), per §6.2.
// The envelope engine attributes each expense precisely; the wider query ensures
// cross-boundary weeks and weekends are included.
func (r *Repo) ForMonth(ctx context.Context, year, month int) ([]Expense, error) {
	// Format as YYYY-MM-DD strings and cast with ::date so the comparison is
	// DATE-to-DATE, avoiding a timestamptz conversion whose result depends on the
	// DB session timezone (midnight Jakarta ≠ midnight UTC).
	from := timeutil.FirstOfMonth(year, month).AddDate(0, 0, -7).Format("2006-01-02")
	to := timeutil.LastOfMonth(year, month).AddDate(0, 0, 7).Format("2006-01-02")
	const q = `
SELECT ` + selectCols + `
FROM amplop.expense
WHERE occurred_date >= $1::date AND occurred_date <= $2::date
ORDER BY occurred_date, occurred_time NULLS FIRST, created_at`
	var out []Expense
	if err := r.db.SelectContext(ctx, &out, q, from, to); err != nil {
		return nil, fmt.Errorf("expenses for month %d-%02d: %w", year, month, err)
	}
	return out, nil
}

// ByID returns the expense with the given id.
func (r *Repo) ByID(ctx context.Context, id uuid.UUID) (Expense, error) {
	const q = `SELECT ` + selectCols + ` FROM amplop.expense WHERE id = $1`
	var e Expense
	if err := r.db.GetContext(ctx, &e, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Expense{}, apierr.NotFound("expense %s not found", id)
		}
		return Expense{}, fmt.Errorf("expense by id %s: %w", id, err)
	}
	return e, nil
}

// Create inserts a new expense and returns the fully-populated row (DB-generated
// id, occurred_year, occurred_month, timestamps filled via RETURNING).
// occurred_year and occurred_month must not appear in the INSERT column list —
// they are generated columns computed by the DB from occurred_date.
func (r *Repo) Create(ctx context.Context, e Expense) (Expense, error) {
	const q = `
INSERT INTO amplop.expense (occurred_date, occurred_time, amount, category, subscription_id, note)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING ` + selectCols
	var created Expense
	err := r.db.GetContext(ctx, &created, q,
		e.OccurredDate, e.OccurredTime, e.Amount, e.Category, e.SubscriptionID, e.Note)
	if err != nil {
		return Expense{}, mapUniqueErr(err, "create expense")
	}
	return created, nil
}

// Update replaces all mutable fields of an existing expense.
func (r *Repo) Update(ctx context.Context, e Expense) (Expense, error) {
	const q = `
UPDATE amplop.expense
SET occurred_date   = $2,
    occurred_time   = $3,
    amount          = $4,
    category        = $5,
    subscription_id = $6,
    note            = $7,
    updated_at      = current_timestamp()
WHERE id = $1
RETURNING ` + selectCols
	var updated Expense
	err := r.db.GetContext(ctx, &updated, q,
		e.ID, e.OccurredDate, e.OccurredTime, e.Amount, e.Category, e.SubscriptionID, e.Note)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Expense{}, apierr.NotFound("expense %s not found", e.ID)
		}
		return Expense{}, mapUniqueErr(err, "update expense")
	}
	return updated, nil
}

// Delete removes the expense with id.
func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM amplop.expense WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete expense %s: %w", id, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apierr.NotFound("expense %s not found", id)
	}
	return nil
}

// ExistsForSubscriptionMonth reports whether a Langganan expense already exists
// for (subscriptionID, year, month). When excludeID is non-nil the check skips
// that row — used on update to avoid flagging the row being edited as a conflict.
// This is the service-layer pre-check; the DB unique index is the backstop.
func (r *Repo) ExistsForSubscriptionMonth(ctx context.Context, subscriptionID uuid.UUID, year, month int, excludeID *uuid.UUID) (bool, error) {
	var (
		q    string
		args []any
	)
	if excludeID == nil {
		q = `SELECT EXISTS (
    SELECT 1 FROM amplop.expense
    WHERE subscription_id = $1
      AND occurred_year   = $2
      AND occurred_month  = $3
)`
		args = []any{subscriptionID, year, month}
	} else {
		q = `SELECT EXISTS (
    SELECT 1 FROM amplop.expense
    WHERE subscription_id = $1
      AND occurred_year   = $2
      AND occurred_month  = $3
      AND id != $4
)`
		args = []any{subscriptionID, year, month, *excludeID}
	}
	var exists bool
	if err := r.db.GetContext(ctx, &exists, q, args...); err != nil {
		return false, fmt.Errorf("exists for subscription month: %w", err)
	}
	return exists, nil
}

// mapUniqueErr converts the DB unique-index violation on
// expense_one_sub_payment_per_month into an apierr.Conflict as a backstop for
// the once-per-month rule. All other errors are wrapped and returned unchanged.
func mapUniqueErr(err error, op string) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) &&
		pqErr.Code == "23505" &&
		strings.Contains(pqErr.Constraint, "expense_one_sub_payment_per_month") {
		return apierr.Conflict("subscription already paid this month")
	}
	return fmt.Errorf("%s: %w", op, err)
}
