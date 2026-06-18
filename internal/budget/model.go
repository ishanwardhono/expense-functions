package budget

import (
	"time"

	"github.com/google/uuid"
)

// Config is one effective-dated budget configuration row.
type Config struct {
	ID             uuid.UUID `db:"id"`
	EffectiveYear  int16     `db:"effective_year"`
	EffectiveMonth int16     `db:"effective_month"`
	Monthly        int64     `db:"monthly"`
	ShopWeekly     int64     `db:"shop_weekly"`
	WeekendBudget  int64     `db:"weekend_budget"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
