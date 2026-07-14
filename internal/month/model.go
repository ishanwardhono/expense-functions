// Package month assembles the GET /month dashboard: it resolves the effective
// budget config + subscription set, loads the wide-window expenses, runs the
// pure envelope engine, and shapes the §7.1 payload.
package month

import "github.com/ishanwardhono/expense-function/internal/expense"

// Dashboard is the full GET /month payload (spec §7.1).
type Dashboard struct {
	Period        Period                        `json:"period"`
	Stats         Stats                         `json:"stats"`
	Envelopes     []EnvelopeRow                 `json:"envelopes"`
	BelanjaWeeks  []WeekDTO                     `json:"belanja_weeks"`
	Weekends      []WeekendDTO                  `json:"weekends"`
	Flex          Flex                          `json:"flex"`
	Calendar      []CalendarDay                 `json:"calendar"`
	Days          map[string][]expense.Response `json:"days"`
	Subscriptions []SubscriptionDTO             `json:"subscriptions"`
}

// Period identifies the viewed month.
type Period struct {
	Year      int    `json:"year"`
	Month     int    `json:"month"`
	Label     string `json:"label"` // e.g. "Juni 2026"
	IsCurrent bool   `json:"is_current"`
}

// Stats is the month-level total.
type Stats struct {
	Spent     int64 `json:"spent"`
	Budget    int64 `json:"budget"`
	Remaining int64 `json:"remaining"`
}

// EnvelopeRow is one of the four envelope summary rows.
type EnvelopeRow struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Budget int64  `json:"budget"`
	Spent  int64  `json:"spent"`
	Left   int64  `json:"left"`
	Over   bool   `json:"over"`
}

// WeekDTO is one shopping week (Mon–Sun, owned by its Friday).
type WeekDTO struct {
	Range  string `json:"range"` // e.g. "1–7 Jun"
	Monday string `json:"monday"`
	Friday string `json:"friday"`
	Sunday string `json:"sunday"`
	Budget int64  `json:"budget"`
	Spent  int64  `json:"spent"`
	Left   int64  `json:"left"`
	State  string `json:"state"`
}

// WeekendDTO is one weekend (Sat–Sun, owned by its Saturday).
type WeekendDTO struct {
	Range    string `json:"range"`
	Saturday string `json:"saturday"`
	Sunday   string `json:"sunday"`
	Budget   int64  `json:"budget"`
	Spent    int64  `json:"spent"`
	Left     int64  `json:"left"`
	State    string `json:"state"`
}

// Flex is the flexible-envelope summary. Rollover is the §6.6 sum of closed
// sources' leftover; Left = Budget + Rollover − Spent.
type Flex struct {
	Budget        int64             `json:"budget"`
	Rollover      int64             `json:"rollover"`
	Spent         int64             `json:"spent"`
	Left          int64             `json:"left"`
	RolloverItems []RolloverItemDTO `json:"rollover_items"`
}

// RolloverItemDTO is one rollover group (spec §7.1): the summed contribution
// of every closed source of a type. Types with no closed source are omitted.
type RolloverItemDTO struct {
	Type   string `json:"type"` // week | weekend | subscription
	Amount int64  `json:"amount"`
}

// CalendarDay is one cell of the month calendar grid.
type CalendarDay struct {
	Date      string `json:"date"`
	Dow       int    `json:"dow"` // ISO weekday: Monday=1 .. Sunday=7
	IsWeekend bool   `json:"is_weekend"`
	IsToday   bool   `json:"is_today"`
	Spent     int64  `json:"spent"`
}

// PaidDTO is the derived payment of a subscription this month (nil when unpaid).
type PaidDTO struct {
	Date   string `json:"date"`
	Amount int64  `json:"amount"`
}

// SubscriptionDTO is one resolved subscription with its derived payment status.
type SubscriptionDTO struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Color  string   `json:"color"`
	Alloc  int64    `json:"alloc"`
	DueDay int16    `json:"due_day"`
	Paid   *PaidDTO `json:"paid"`
	Status string   `json:"status"`
}
