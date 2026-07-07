// Package envelope is the pure budgeting engine: it attributes expenses to
// envelopes and computes a month's envelope/budget figures. It knows nothing
// about HTTP, the database, or effective-date versioning — it receives the
// already-resolved config + subscription set for the month (spec §6).
package envelope

import (
	"time"

	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// Category is an expense category.
type Category string

const (
	CatMakan     Category = "Makan"
	CatBelanja   Category = "Belanja"
	CatJajan     Category = "Jajan"
	CatCash      Category = "Cash"
	CatLainnya   Category = "Lainnya"
	CatLangganan Category = "Langganan"
)

// EnvelopeID identifies one of the four derived envelopes.
type EnvelopeID string

const (
	EnvBelanja   EnvelopeID = "belanja"
	EnvWeekend   EnvelopeID = "weekend"
	EnvLangganan EnvelopeID = "langganan"
	EnvFleksibel EnvelopeID = "fleksibel"
)

// Label returns the human-facing label for an envelope (Indonesian).
func (e EnvelopeID) Label() string {
	switch e {
	case EnvBelanja:
		return "Belanja Mingguan"
	case EnvWeekend:
		return "Akhir Pekan"
	case EnvLangganan:
		return "Langganan"
	case EnvFleksibel:
		return "Fleksibel"
	}
	return string(e)
}

// ShortLabel returns the compact badge label used on per-expense rows in the
// day list (spec §7.1, e.g. "BLNJ", "SUBS").
func (e EnvelopeID) ShortLabel() string {
	switch e {
	case EnvBelanja:
		return "BLNJ"
	case EnvWeekend:
		return "WKND"
	case EnvLangganan:
		return "SUBS"
	case EnvFleksibel:
		return "FLEX"
	}
	return string(e)
}

// isWeekend reports whether the date falls on Saturday or Sunday (Asia/Jakarta).
// Normalizing to Loc keeps weekday classification consistent with dayNum's
// date comparison even if a caller passes a timestamp in another zone.
func isWeekend(date time.Time) bool {
	wd := date.In(timeutil.Loc).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// EnvelopeOf attributes an expense to an envelope by category and day-of-week
// (spec §6.1):
//   - Langganan        → langganan (any day)
//   - Belanja / Cash   → belanja (any day)
//   - Makan / Jajan    → weekend on Sat/Sun, else belanja
//   - Lainnya          → fleksibel (any day)
func EnvelopeOf(cat Category, date time.Time) EnvelopeID {
	switch cat {
	case CatLangganan:
		return EnvLangganan
	case CatBelanja, CatCash:
		return EnvBelanja
	case CatMakan, CatJajan:
		if isWeekend(date) {
			return EnvWeekend
		}
		return EnvBelanja
	case CatLainnya:
		return EnvFleksibel
	}
	// Unknown category: treat as flexible (defensive; validation rejects these).
	return EnvFleksibel
}
