package envelope

import (
	"time"

	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// Expense is one transaction as the engine sees it. Date is the calendar date;
// SubscriptionID is set only for Langganan expenses.
type Expense struct {
	Date           time.Time
	Amount         int64
	Category       Category
	SubscriptionID string
}

// Config is the resolved budget configuration for the viewed month.
type Config struct {
	Monthly       int64
	ShopWeekly    int64
	WeekendBudget int64
}

// Subscription is a resolved subscription (latest active version) for the month.
type Subscription struct {
	ID     string
	Name   string
	Color  string
	Alloc  int64
	DueDay int
}

// State classifies a week/weekend pill relative to "today" (spec §6.3).
type State string

const (
	StatePast    State = "past"
	StateCurrent State = "current"
	StateFuture  State = "future"
)

// Week is one shopping week (Mon–Sun), owned by the month of its Friday.
type Week struct {
	Monday time.Time
	Friday time.Time
	Sunday time.Time
	Budget int64
	Spent  int64
	Left   int64
	State  State
}

// Weekend is one weekend (Sat–Sun), owned by the month of its Saturday.
type Weekend struct {
	Saturday time.Time
	Sunday   time.Time
	Budget   int64
	Spent    int64
	Left     int64
	State    State
}

// RolloverType identifies the kind of closed source behind a RolloverItem.
type RolloverType string

const (
	RolloverWeek         RolloverType = "week"
	RolloverWeekend      RolloverType = "weekend"
	RolloverSubscription RolloverType = "subscription"
)

// RolloverItem is one closed source's contribution to the fleksibel rollover
// (spec §6.6). Week/weekend items carry Start/End; subscription items carry
// Name. Zero amounts are kept — the breakdown is a complete audit trail.
type RolloverItem struct {
	Type   RolloverType
	Start  time.Time
	End    time.Time
	Name   string
	Amount int64
}

// Row is one envelope summary row.
type Row struct {
	ID     EnvelopeID
	Label  string
	Budget int64
	Spent  int64
	Left   int64
	Over   bool
}

// MonthInput is the resolved context the engine computes over. Expenses must
// cover the wide boundary window [firstOfMonth−7d, lastOfMonth+7d] (spec §6.2)
// so cross-boundary weeks/weekends are attributed correctly.
type MonthInput struct {
	Year          int
	Month         int
	Today         time.Time
	Config        Config
	Subscriptions []Subscription
	Expenses      []Expense
}

// MonthResult is the computed month dashboard data (spec §6.3).
type MonthResult struct {
	Weeks    []Week
	Weekends []Weekend

	SubsAlloc      int64
	LanggananSpent int64
	FlexSpent      int64

	// Rollover (§6.6): Σ leftover of closed sources — past week/weekend pills
	// and paid subscriptions (alloc − paid) — both signs. It raises/lowers the
	// fleksibel row's Left only; FlexBudget and Sisa are untouched.
	Rollover      int64
	RolloverItems []RolloverItem

	ShopBudget int64
	ShopSpent  int64
	WkndBudget int64
	WkndSpent  int64

	FlexBudget int64
	TotalSpent int64
	Sisa       int64

	Rows []Row
}

// dayNum collapses a date to a comparable YYYYMMDD integer in Asia/Jakarta,
// ignoring any time component.
func dayNum(t time.Time) int {
	y, m, d := t.In(timeutil.Loc).Date()
	return y*10000 + int(m)*100 + d
}

// ComputeMonth runs the envelope engine over the resolved month context.
func ComputeMonth(in MonthInput) MonthResult {
	cfg := in.Config
	todayN := dayNum(in.Today)

	// Shopping weeks: one per Friday in the month; range Mon..Sun.
	fridays := timeutil.DatesOfWeekdayInMonth(in.Year, in.Month, time.Friday)
	weeks := make([]Week, len(fridays))
	for i, fri := range fridays {
		mon := fri.AddDate(0, 0, -4)
		sun := fri.AddDate(0, 0, 2)
		monN, sunN := dayNum(mon), dayNum(sun)
		var spent int64
		for _, e := range in.Expenses {
			if EnvelopeOf(e.Category, e.Date) != EnvBelanja {
				continue
			}
			n := dayNum(e.Date)
			if n >= monN && n <= sunN {
				spent += e.Amount
			}
		}
		weeks[i] = Week{
			Monday: mon, Friday: fri, Sunday: sun,
			Budget: cfg.ShopWeekly, Spent: spent, Left: cfg.ShopWeekly - spent,
			State: pillState(monN, sunN, todayN),
		}
	}

	// Weekends: one per Saturday in the month; range Sat..Sun.
	saturdays := timeutil.DatesOfWeekdayInMonth(in.Year, in.Month, time.Saturday)
	weekends := make([]Weekend, len(saturdays))
	for i, sat := range saturdays {
		sun := sat.AddDate(0, 0, 1)
		satN, sunN := dayNum(sat), dayNum(sun)
		var spent int64
		for _, e := range in.Expenses {
			if EnvelopeOf(e.Category, e.Date) != EnvWeekend {
				continue
			}
			n := dayNum(e.Date)
			if n == satN || n == sunN {
				spent += e.Amount
			}
		}
		weekends[i] = Weekend{
			Saturday: sat, Sunday: sun,
			Budget: cfg.WeekendBudget, Spent: spent, Left: cfg.WeekendBudget - spent,
			State: pillState(satN, sunN, todayN),
		}
	}

	// Flex and Langganan are attributed by calendar month only.
	var langgananSpent, flexSpent int64
	for _, e := range in.Expenses {
		if !inMonth(e.Date, in.Year, in.Month) {
			continue
		}
		switch EnvelopeOf(e.Category, e.Date) {
		case EnvLangganan:
			langgananSpent += e.Amount
		case EnvFleksibel:
			flexSpent += e.Amount
		}
	}

	var subsAlloc int64
	for _, s := range in.Subscriptions {
		subsAlloc += s.Alloc
	}

	var shopBudget, shopSpent int64
	shopBudget = cfg.ShopWeekly * int64(len(weeks))
	for _, w := range weeks {
		shopSpent += w.Spent
	}
	var wkndBudget, wkndSpent int64
	wkndBudget = cfg.WeekendBudget * int64(len(weekends))
	for _, w := range weekends {
		wkndSpent += w.Spent
	}

	flexBudget := cfg.Monthly - shopBudget - wkndBudget - subsAlloc
	totalSpent := shopSpent + wkndSpent + langgananSpent + flexSpent

	// Rollover (§6.6): closed sources flush their leftover into fleksibel.
	// Current/future pills and unpaid subscriptions contribute nothing.
	var rollover int64
	var rolloverItems []RolloverItem
	for _, w := range weeks {
		if w.State != StatePast {
			continue
		}
		rollover += w.Left
		rolloverItems = append(rolloverItems, RolloverItem{Type: RolloverWeek, Start: w.Monday, End: w.Sunday, Amount: w.Left})
	}
	for _, w := range weekends {
		if w.State != StatePast {
			continue
		}
		rollover += w.Left
		rolloverItems = append(rolloverItems, RolloverItem{Type: RolloverWeekend, Start: w.Saturday, End: w.Sunday, Amount: w.Left})
	}
	for _, s := range in.Subscriptions {
		st := SubscriptionStatus(s, in.Expenses, in.Year, in.Month)
		if !st.Paid {
			continue
		}
		rollover += st.Diff
		rolloverItems = append(rolloverItems, RolloverItem{Type: RolloverSubscription, Name: s.Name, Amount: st.Diff})
	}

	flexLeft := flexBudget + rollover - flexSpent
	rows := []Row{
		makeRow(EnvBelanja, shopBudget, shopSpent),
		makeRow(EnvWeekend, wkndBudget, wkndSpent),
		makeRow(EnvLangganan, subsAlloc, langgananSpent),
		{ID: EnvFleksibel, Label: EnvFleksibel.Label(), Budget: flexBudget, Spent: flexSpent, Left: flexLeft, Over: flexLeft < 0},
	}

	return MonthResult{
		Weeks:          weeks,
		Weekends:       weekends,
		SubsAlloc:      subsAlloc,
		LanggananSpent: langgananSpent,
		FlexSpent:      flexSpent,
		Rollover:       rollover,
		RolloverItems:  rolloverItems,
		ShopBudget:     shopBudget,
		ShopSpent:      shopSpent,
		WkndBudget:     wkndBudget,
		WkndSpent:      wkndSpent,
		FlexBudget:     flexBudget,
		TotalSpent:     totalSpent,
		Sisa:           cfg.Monthly - totalSpent,
		Rows:           rows,
	}
}

func makeRow(id EnvelopeID, budget, spent int64) Row {
	return Row{
		ID:     id,
		Label:  id.Label(),
		Budget: budget,
		Spent:  spent,
		Left:   budget - spent,
		Over:   spent > budget,
	}
}

// pillState ports weekPillState (spec §6.3): past when the range ends before
// today, current when today is within [start, end], else future.
func pillState(startN, endN, todayN int) State {
	if endN < todayN {
		return StatePast
	}
	if startN <= todayN {
		return StateCurrent
	}
	return StateFuture
}

func inMonth(date time.Time, year, month int) bool {
	t := date.In(timeutil.Loc)
	return t.Year() == year && int(t.Month()) == month
}

// SpentOf returns the total amount of all expenses on the given calendar date.
func SpentOf(expenses []Expense, date time.Time) int64 {
	n := dayNum(date)
	var total int64
	for _, e := range expenses {
		if dayNum(e.Date) == n {
			total += e.Amount
		}
	}
	return total
}
