// Package timeutil centralizes Asia/Jakarta time handling and the calendar-date
// helpers the envelope engine and effective-dated writes depend on.
//
// All date logic in the app is anchored to Asia/Jakarta. "Now" is overridable
// via the TIME env var (RFC3339) so tests and effective-dated writes are
// deterministic.
package timeutil

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Loc is the single timezone used for every date/time decision.
var Loc = mustLoadJakarta()

func mustLoadJakarta() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		// Asia/Jakarta is a fixed UTC+7 zone with no DST; fall back rather than panic.
		return time.FixedZone("Asia/Jakarta", 7*60*60)
	}
	return loc
}

// LoadTime returns "now" (Asia/Jakarta), overridden by the TIME env var
// (RFC3339) when set. A malformed TIME is a hard error so misconfiguration
// surfaces loudly — config.Load() calls this at startup, failing fast.
//
// This is the single TIME parser; Now() reuses it.
func LoadTime() (time.Time, error) {
	if v := os.Getenv("TIME"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid TIME environment variable: %w", err)
		}
		return t.In(Loc), nil
	}
	return time.Now().In(Loc), nil
}

// Now returns the current instant in Asia/Jakarta, honoring the TIME override.
// It reuses LoadTime; a malformed TIME is logged and falls back to the wall
// clock. In practice startup validates TIME via config.Load(), so a running
// process never reaches the fallback.
func Now() time.Time {
	t, err := LoadTime()
	if err != nil {
		log.Printf("timeutil: %v; falling back to wall clock", err)
		return time.Now().In(Loc)
	}
	return t
}

// CurrentMonth returns the (year, month) of the given instant in Asia/Jakarta.
// Effective-dated writes stamp this as their effective month.
func CurrentMonth(now time.Time) (year, month int) {
	t := now.In(Loc)
	return t.Year(), int(t.Month())
}

// Date builds a midnight Asia/Jakarta date.
func Date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, Loc)
}

// FirstOfMonth returns the first day of (year, month) at midnight Asia/Jakarta.
func FirstOfMonth(year, month int) time.Time {
	return Date(year, month, 1)
}

// LastOfMonth returns the last day of (year, month) at midnight Asia/Jakarta.
func LastOfMonth(year, month int) time.Time {
	return FirstOfMonth(year, month).AddDate(0, 1, -1)
}

// DatesOfWeekdayInMonth returns each date in (year, month) that falls on the
// given weekday, in ascending order. Used to enumerate the month's Fridays
// (shopping weeks) and Saturdays (weekends) per spec §6.2.
func DatesOfWeekdayInMonth(year, month int, wd time.Weekday) []time.Time {
	var out []time.Time
	last := LastOfMonth(year, month).Day()
	for d := 1; d <= last; d++ {
		t := Date(year, month, d)
		if t.Weekday() == wd {
			out = append(out, t)
		}
	}
	return out
}

// SameDate reports whether a and b are the same calendar day (ignoring time),
// compared in Asia/Jakarta.
func SameDate(a, b time.Time) bool {
	ay, am, ad := a.In(Loc).Date()
	by, bm, bd := b.In(Loc).Date()
	return ay == by && am == bm && ad == bd
}
