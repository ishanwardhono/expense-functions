package expense

import (
	"time"

	"github.com/google/uuid"

	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// Expense is the full DB representation of one amplop.expense row.
// occurred_year and occurred_month are generated columns stored by the DB from
// occurred_date; they are populated on SELECT/RETURNING but must never appear
// in an INSERT column list.
type Expense struct {
	ID             uuid.UUID  `db:"id"`
	OccurredDate   time.Time  `db:"occurred_date"`
	OccurredTime   *time.Time `db:"occurred_time"` // lib/pq: SQL TIME → time.Time with date 0001-01-01
	Amount         int64      `db:"amount"`
	Category       string     `db:"category"`
	SubscriptionID *uuid.UUID `db:"subscription_id"`
	Note           string     `db:"note"`
	OccurredYear   int16      `db:"occurred_year"`
	OccurredMonth  int16      `db:"occurred_month"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

// OccurredAt reconstructs the Asia/Jakarta wall-clock instant of the expense.
// lib/pq returns SQL DATE as midnight UTC; .In(Loc).Date() extracts the correct
// Jakarta calendar date. SQL TIME clock-face hours are in UTC on the 0001-01-01
// origin, so we read them with .UTC() before combining with the date.
func (e Expense) OccurredAt() time.Time {
	y, m, d := e.OccurredDate.In(timeutil.Loc).Date()
	if e.OccurredTime == nil {
		return time.Date(y, m, d, 0, 0, 0, 0, timeutil.Loc)
	}
	t := e.OccurredTime.UTC()
	return time.Date(y, m, d, t.Hour(), t.Minute(), t.Second(), 0, timeutil.Loc)
}
