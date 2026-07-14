package month

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ishanwardhono/expense-function/internal/budget"
	"github.com/ishanwardhono/expense-function/internal/expense"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
	"github.com/ishanwardhono/expense-function/internal/subscription"
)

type fakeBudget struct{ cfg budget.Config }

func (f fakeBudget) Resolve(_ context.Context, _, _ int) (budget.Config, error) { return f.cfg, nil }

type fakeSubs struct{ subs []subscription.Resolved }

func (f fakeSubs) Resolve(_ context.Context, _, _ int) ([]subscription.Resolved, error) {
	return f.subs, nil
}

type fakeExpenses struct{ exps []expense.Expense }

func (f fakeExpenses) ForMonth(_ context.Context, _, _ int) ([]expense.Expense, error) {
	return f.exps, nil
}

func fixedClock(y, m, d int) timeutil.Clock {
	return func() time.Time { return timeutil.Date(y, m, d) }
}

func exp(date time.Time, amount int64, cat string, sub *uuid.UUID) expense.Expense {
	return expense.Expense{
		ID:             uuid.New(),
		OccurredDate:   date,
		Amount:         amount,
		Category:       cat,
		SubscriptionID: sub,
	}
}

func TestDashboard_Assembly(t *testing.T) {
	subID := uuid.New()
	cfg := budget.Config{Monthly: 5_000_000, ShopWeekly: 600_000, WeekendBudget: 200_000}
	subs := []subscription.Resolved{{ID: subID, Name: "Netflix", Color: "#c8403c", Alloc: 187_000, DueDay: 5}}
	exps := []expense.Expense{
		exp(timeutil.Date(2026, 6, 15), 18_000, "Makan", nil),        // weekday → belanja, June week
		exp(timeutil.Date(2026, 6, 5), 186_000, "Langganan", &subID), // langganan, paid
		exp(timeutil.Date(2026, 6, 16), 8_000, "Lainnya", nil),       // Lainnya → fleksibel
		exp(timeutil.Date(2026, 6, 29), 50_000, "Belanja", nil),      // week owned by July's Friday
	}

	svc := NewService(fakeBudget{cfg}, fakeSubs{subs}, fakeExpenses{exps}, fixedClock(2026, 6, 23))
	dash, err := svc.Dashboard(context.Background(), 2026, 6)
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}

	if dash.Period.Label != "Juni 2026" {
		t.Errorf("label: got %q, want %q", dash.Period.Label, "Juni 2026")
	}
	if !dash.Period.IsCurrent {
		t.Error("expected is_current true for the clock's month")
	}
	if dash.Stats.Budget != 5_000_000 {
		t.Errorf("stats.budget: got %d, want 5000000", dash.Stats.Budget)
	}

	// Langganan envelope spent == the single payment.
	var langg EnvelopeRow
	for _, r := range dash.Envelopes {
		if r.ID == "langganan" {
			langg = r
		}
	}
	if langg.Spent != 186_000 {
		t.Errorf("langganan spent: got %d, want 186000", langg.Spent)
	}
	if langg.Budget != 187_000 {
		t.Errorf("langganan budget (subsAlloc): got %d, want 187000", langg.Budget)
	}

	// The Jun-29 belanja belongs to July's week → excluded from June week spend.
	var shopSpent int64
	for _, w := range dash.BelanjaWeeks {
		shopSpent += w.Spent
	}
	if shopSpent != 18_000 {
		t.Errorf("June belanja-week spend: got %d, want 18000 (Jun-29 belongs to July)", shopSpent)
	}

	// ...but it is still visible in the June day list.
	if len(dash.Days["2026-06-29"]) != 1 {
		t.Errorf("expected Jun-29 expense visible in days, got %d", len(dash.Days["2026-06-29"]))
	}

	// Flex spent.
	if dash.Flex.Spent != 8_000 {
		t.Errorf("flex spent: got %d, want 8000", dash.Flex.Spent)
	}

	// Subscription payment status.
	if len(dash.Subscriptions) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(dash.Subscriptions))
	}
	s := dash.Subscriptions[0]
	if s.Status != "paid" || s.Paid == nil || s.Paid.Amount != 186_000 {
		t.Errorf("subscription status: %+v", s)
	}
	if s.Paid != nil && s.Paid.Date != "2026-06-05" {
		t.Errorf("paid date: got %q, want 2026-06-05", s.Paid.Date)
	}

	// Calendar has one cell per June day.
	if len(dash.Calendar) != 30 {
		t.Errorf("calendar length: got %d, want 30", len(dash.Calendar))
	}
}

func TestDashboard_FlexRollover(t *testing.T) {
	// Clock on Tue Jun 23 2026: weeks 1–3 and weekends 1–3 are past; the one
	// subscription is paid (alloc 187000, paid 186000 → +1000).
	subID := uuid.New()
	cfg := budget.Config{Monthly: 5_000_000, ShopWeekly: 600_000, WeekendBudget: 200_000}
	subs := []subscription.Resolved{{ID: subID, Name: "Netflix", Color: "#c8403c", Alloc: 187_000, DueDay: 5}}
	exps := []expense.Expense{
		exp(timeutil.Date(2026, 6, 15), 18_000, "Makan", nil),        // week 3 belanja
		exp(timeutil.Date(2026, 6, 5), 186_000, "Langganan", &subID), // paid sub
		exp(timeutil.Date(2026, 6, 16), 8_000, "Lainnya", nil),       // fleksibel
	}

	svc := NewService(fakeBudget{cfg}, fakeSubs{subs}, fakeExpenses{exps}, fixedClock(2026, 6, 23))
	dash, err := svc.Dashboard(context.Background(), 2026, 6)
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}

	// weeks 600000+600000+582000 + weekends 3×200000 + sub 1000.
	if dash.Flex.Rollover != 2_383_000 {
		t.Errorf("flex.rollover: got %d, want 2383000", dash.Flex.Rollover)
	}
	// flexBudget = 5M − 2.4M − 0.8M − 187000 = 1613000; left = budget + rollover − spent.
	if dash.Flex.Budget != 1_613_000 {
		t.Errorf("flex.budget: got %d, want 1613000", dash.Flex.Budget)
	}
	if dash.Flex.Left != 3_988_000 {
		t.Errorf("flex.left: got %d, want 3988000", dash.Flex.Left)
	}

	items := dash.Flex.RolloverItems
	if len(items) != 7 { // 3 weeks + 3 weekends + 1 paid sub
		t.Fatalf("len(rollover_items) = %d, want 7", len(items))
	}
	first := items[0]
	if first.Type != "week" || first.Start != "2026-06-01" || first.End != "2026-06-07" || first.Amount != 600_000 || first.Name != "" {
		t.Errorf("first item = %+v, want week 2026-06-01..2026-06-07 amount 600000 without name", first)
	}
	sub := items[6]
	if sub.Type != "subscription" || sub.Name != "Netflix" || sub.Amount != 1_000 || sub.Start != "" || sub.End != "" {
		t.Errorf("sub item = %+v, want subscription Netflix amount 1000 without dates", sub)
	}

	// The fleksibel envelope row carries the EFFECTIVE budget (planned +
	// rollover) so the card's progress bar is honest, and the rolled-up left.
	for _, r := range dash.Envelopes {
		if r.ID == "fleksibel" {
			if r.Budget != 3_996_000 { // 1613000 + 2383000
				t.Errorf("fleksibel row budget: got %d, want 3996000", r.Budget)
			}
			if r.Left != 3_988_000 {
				t.Errorf("fleksibel row left: got %d, want 3988000", r.Left)
			}
		}
	}
}

func TestDashboard_FlexRolloverEmptyIsNotNull(t *testing.T) {
	// Month start: nothing closed. rollover_items must serialize as [] (the
	// client guards on the array), never null.
	cfg := budget.Config{Monthly: 5_000_000, ShopWeekly: 600_000, WeekendBudget: 200_000}
	svc := NewService(fakeBudget{cfg}, fakeSubs{}, fakeExpenses{}, fixedClock(2026, 6, 1))
	dash, err := svc.Dashboard(context.Background(), 2026, 6)
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}
	if dash.Flex.Rollover != 0 {
		t.Errorf("flex.rollover: got %d, want 0", dash.Flex.Rollover)
	}
	if dash.Flex.RolloverItems == nil || len(dash.Flex.RolloverItems) != 0 {
		t.Errorf("rollover_items: got %#v, want empty non-nil slice", dash.Flex.RolloverItems)
	}
}

func TestDashboard_RangeLabels(t *testing.T) {
	cfg := budget.Config{Monthly: 5_000_000, ShopWeekly: 600_000, WeekendBudget: 200_000}
	svc := NewService(fakeBudget{cfg}, fakeSubs{}, fakeExpenses{}, fixedClock(2026, 6, 23))
	dash, err := svc.Dashboard(context.Background(), 2026, 6)
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}
	// First June week is Mon Jun 1 .. Sun Jun 7 → "1–7 Jun".
	if len(dash.BelanjaWeeks) == 0 || dash.BelanjaWeeks[0].Range != "1–7 Jun" {
		t.Errorf("first week range: got %q, want %q", weekRangeOrEmpty(dash), "1–7 Jun")
	}
}

func weekRangeOrEmpty(d Dashboard) string {
	if len(d.BelanjaWeeks) == 0 {
		return ""
	}
	return d.BelanjaWeeks[0].Range
}
