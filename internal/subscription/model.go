package subscription

import (
	"time"

	"github.com/google/uuid"
)

// Identity is the stable amplop.subscription row. Name and color are cosmetic
// fields that are not effective-dated; they apply across all months.
type Identity struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	Color     string    `db:"color"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Version is one effective-dated attribute snapshot for a subscription.
// A new Version is written each time alloc, due_day, or active changes;
// reads use the latest Version with (effective_year, effective_month) ≤ viewed month.
type Version struct {
	ID             uuid.UUID `db:"id"`
	SubscriptionID uuid.UUID `db:"subscription_id"`
	EffectiveYear  int16     `db:"effective_year"`
	EffectiveMonth int16     `db:"effective_month"`
	Alloc          int64     `db:"alloc"`
	DueDay         int16     `db:"due_day"`
	Active         bool      `db:"active"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// Resolved is an Identity joined to its effective Version for a given month,
// as returned by the §5.1 lateral resolution query. Active is omitted — the
// query already filters WHERE v.active = true.
type Resolved struct {
	ID     uuid.UUID `db:"id"`
	Name   string    `db:"name"`
	Color  string    `db:"color"`
	Alloc  int64     `db:"alloc"`
	DueDay int16     `db:"due_day"`
}
