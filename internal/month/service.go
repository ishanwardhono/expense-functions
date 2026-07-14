package month

import (
	"context"
	"fmt"
	"time"

	"github.com/ishanwardhono/expense-function/internal/budget"
	"github.com/ishanwardhono/expense-function/internal/envelope"
	"github.com/ishanwardhono/expense-function/internal/expense"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
	"github.com/ishanwardhono/expense-function/internal/subscription"
)

// budgetResolver resolves the effective budget config for a month (spec §5.1).
type budgetResolver interface {
	Resolve(ctx context.Context, year, month int) (budget.Config, error)
}

// subResolver resolves the active subscription set for a month (spec §5.1).
type subResolver interface {
	Resolve(ctx context.Context, year, month int) ([]subscription.Resolved, error)
}

// expenseLister loads expenses over the wide boundary window (spec §6.2).
type expenseLister interface {
	ForMonth(ctx context.Context, year, month int) ([]expense.Expense, error)
}

// Service resolves the month context and assembles the dashboard payload.
type Service struct {
	budget   budgetResolver
	subs     subResolver
	expenses expenseLister
	now      timeutil.Clock
}

// NewService builds a month Service. now defaults to timeutil.Now when nil.
func NewService(b budgetResolver, s subResolver, e expenseLister, now timeutil.Clock) *Service {
	if now == nil {
		now = timeutil.Now
	}
	return &Service{budget: b, subs: s, expenses: e, now: now}
}

// Dashboard resolves the effective context for (year, month), runs the engine,
// and assembles the §7.1 payload.
func (s *Service) Dashboard(ctx context.Context, year, month int) (Dashboard, error) {
	cfg, err := s.budget.Resolve(ctx, year, month)
	if err != nil {
		return Dashboard{}, err
	}
	resolvedSubs, err := s.subs.Resolve(ctx, year, month)
	if err != nil {
		return Dashboard{}, err
	}
	exps, err := s.expenses.ForMonth(ctx, year, month)
	if err != nil {
		return Dashboard{}, err
	}

	today := s.now()

	engSubs := make([]envelope.Subscription, 0, len(resolvedSubs))
	for _, r := range resolvedSubs {
		engSubs = append(engSubs, envelope.Subscription{
			ID: r.ID.String(), Name: r.Name, Color: r.Color, Alloc: r.Alloc, DueDay: int(r.DueDay),
		})
	}
	engExps := make([]envelope.Expense, 0, len(exps))
	for _, e := range exps {
		subID := ""
		if e.SubscriptionID != nil {
			subID = e.SubscriptionID.String()
		}
		engExps = append(engExps, envelope.Expense{
			Date: e.OccurredDate, Amount: e.Amount, Category: envelope.Category(e.Category), SubscriptionID: subID,
		})
	}

	result := envelope.ComputeMonth(envelope.MonthInput{
		Year:  year,
		Month: month,
		Today: today,
		Config: envelope.Config{
			Monthly: cfg.Monthly, ShopWeekly: cfg.ShopWeekly, WeekendBudget: cfg.WeekendBudget,
		},
		Subscriptions: engSubs,
		Expenses:      engExps,
	})

	curY, curM := timeutil.CurrentMonth(today)

	dash := Dashboard{
		Period: Period{
			Year: year, Month: month,
			Label:     fmt.Sprintf("%s %d", timeutil.MonthName(month), year),
			IsCurrent: year == curY && month == curM,
		},
		Stats: Stats{Spent: result.TotalSpent, Budget: cfg.Monthly, Remaining: result.Sisa},
		Flex: Flex{
			Budget:        result.FlexBudget,
			Rollover:      result.Rollover,
			Spent:         result.FlexSpent,
			Left:          result.FlexBudget + result.Rollover - result.FlexSpent,
			RolloverItems: rolloverItems(result.RolloverItems),
		},
		Days: map[string][]expense.Response{},
	}

	for _, row := range result.Rows {
		dash.Envelopes = append(dash.Envelopes, EnvelopeRow{
			ID: string(row.ID), Label: row.Label, Budget: row.Budget, Spent: row.Spent, Left: row.Left, Over: row.Over,
		})
	}
	for _, w := range result.Weeks {
		dash.BelanjaWeeks = append(dash.BelanjaWeeks, WeekDTO{
			Range:  rangeLabel(w.Monday, w.Sunday),
			Monday: ymd(w.Monday), Friday: ymd(w.Friday), Sunday: ymd(w.Sunday),
			Budget: w.Budget, Spent: w.Spent, Left: w.Left, State: string(w.State),
		})
	}
	for _, w := range result.Weekends {
		dash.Weekends = append(dash.Weekends, WeekendDTO{
			Range:    rangeLabel(w.Saturday, w.Sunday),
			Saturday: ymd(w.Saturday), Sunday: ymd(w.Sunday),
			Budget: w.Budget, Spent: w.Spent, Left: w.Left, State: string(w.State),
		})
	}

	// Calendar grid: one cell per day of the calendar month.
	last := timeutil.LastOfMonth(year, month).Day()
	for d := 1; d <= last; d++ {
		date := timeutil.Date(year, month, d)
		dash.Calendar = append(dash.Calendar, CalendarDay{
			Date:      ymd(date),
			Dow:       isoDow(date),
			IsWeekend: isWeekend(date),
			IsToday:   timeutil.SameDate(date, today),
			Spent:     envelope.SpentOf(engExps, date),
		})
	}

	// Day list: in-month expenses grouped by calendar date, preserving repo order.
	for _, e := range exps {
		ed := e.OccurredDate.In(timeutil.Loc)
		if ed.Year() != year || int(ed.Month()) != month {
			continue
		}
		key := ymd(e.OccurredDate)
		dash.Days[key] = append(dash.Days[key], expense.ToResponse(e))
	}

	// Subscriptions with derived payment status.
	for _, sub := range engSubs {
		status := envelope.SubscriptionStatus(sub, engExps, year, month)
		dto := SubscriptionDTO{
			ID: sub.ID, Name: sub.Name, Color: sub.Color, Alloc: sub.Alloc, DueDay: int16(sub.DueDay),
			Status: string(status.Status),
		}
		if status.Paid {
			dto.Paid = &PaidDTO{Date: ymd(status.PaidDate), Amount: status.PaidAmount}
		}
		dash.Subscriptions = append(dash.Subscriptions, dto)
	}

	return dash, nil
}

// rolloverItems groups the engine's per-source rollover items into one summed
// DTO per type (week, weekend, subscription — §7.1), skipping types with no
// closed source. It always returns a non-nil slice so the payload serializes
// as [] rather than null.
func rolloverItems(items []envelope.RolloverItem) []RolloverItemDTO {
	sums := map[envelope.RolloverType]int64{}
	counts := map[envelope.RolloverType]int{}
	for _, it := range items {
		sums[it.Type] += it.Amount
		counts[it.Type]++
	}
	dtos := make([]RolloverItemDTO, 0, 3)
	for _, t := range []envelope.RolloverType{envelope.RolloverWeek, envelope.RolloverWeekend, envelope.RolloverSubscription} {
		if counts[t] == 0 {
			continue
		}
		dtos = append(dtos, RolloverItemDTO{Type: string(t), Amount: sums[t]})
	}
	return dtos
}

// ymd formats a date as YYYY-MM-DD in Asia/Jakarta.
func ymd(t time.Time) string {
	return t.In(timeutil.Loc).Format("2006-01-02")
}

// isoDow returns the ISO weekday (Monday=1 .. Sunday=7).
func isoDow(t time.Time) int {
	wd := int(t.In(timeutil.Loc).Weekday()) // Sunday=0 .. Saturday=6
	if wd == 0 {
		return 7
	}
	return wd
}

// isWeekend reports whether the date is Saturday or Sunday (Asia/Jakarta).
func isWeekend(t time.Time) bool {
	wd := t.In(timeutil.Loc).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// rangeLabel renders a date range like "1–7 Jun" (same month) or "29 Jun–5 Jul"
// (spanning two months), using Indonesian month abbreviations.
func rangeLabel(start, end time.Time) string {
	s := start.In(timeutil.Loc)
	e := end.In(timeutil.Loc)
	if s.Month() == e.Month() {
		return fmt.Sprintf("%d–%d %s", s.Day(), e.Day(), timeutil.MonthAbbr(int(s.Month())))
	}
	return fmt.Sprintf("%d %s–%d %s", s.Day(), timeutil.MonthAbbr(int(s.Month())), e.Day(), timeutil.MonthAbbr(int(e.Month())))
}
