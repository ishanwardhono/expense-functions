package envelope

import (
	"testing"
	"time"

	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

func d(y, m, day int) time.Time { return timeutil.Date(y, m, day) }

// ---- EnvelopeOf ----------------------------------------------------------

func TestEnvelopeOf(t *testing.T) {
	weekday := d(2026, 6, 1) // Monday
	saturday := d(2026, 6, 6)
	sunday := d(2026, 6, 7)

	cases := []struct {
		name string
		cat  Category
		date time.Time
		want EnvelopeID
	}{
		{"Langganan any day", CatLangganan, weekday, EnvLangganan},
		{"Langganan weekend", CatLangganan, saturday, EnvLangganan},
		{"Belanja weekday", CatBelanja, weekday, EnvBelanja},
		{"Belanja weekend", CatBelanja, saturday, EnvBelanja},
		{"Cash weekday", CatCash, weekday, EnvBelanja},
		{"Cash weekend", CatCash, sunday, EnvBelanja},
		{"Makan weekday", CatMakan, weekday, EnvBelanja},
		{"Makan saturday", CatMakan, saturday, EnvWeekend},
		{"Makan sunday", CatMakan, sunday, EnvWeekend},
		{"Jajan weekday", CatJajan, weekday, EnvBelanja},
		{"Jajan weekend", CatJajan, saturday, EnvWeekend},
		{"Lainnya weekday", CatLainnya, weekday, EnvFleksibel},
		{"Lainnya saturday", CatLainnya, saturday, EnvFleksibel},
		{"Lainnya sunday", CatLainnya, sunday, EnvFleksibel},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := EnvelopeOf(c.cat, c.date); got != c.want {
				t.Fatalf("EnvelopeOf(%s, %s) = %s, want %s", c.cat, c.date.Format("2006-01-02"), got, c.want)
			}
		})
	}
}

// ---- June 2026 seed reproduction (mirrors the prototype) -----------------

// seedConfig and seedExpenses replicate the prototype's June-2026 seed so the
// Go engine reproduces the JS computeAmplop outputs exactly.
func seedConfig() Config {
	return Config{Monthly: 5_000_000, ShopWeekly: 600_000, WeekendBudget: 200_000}
}

func seedSubs() []Subscription {
	return []Subscription{
		{ID: "s1", Name: "Netflix", Alloc: 187_000, DueDay: 5},
		{ID: "s2", Name: "Spotify", Alloc: 55_000, DueDay: 10},
		{ID: "s3", Name: "YouTube Premium", Alloc: 59_000, DueDay: 18},
		{ID: "s4", Name: "iCloud+", Alloc: 29_000, DueDay: 28},
	}
}

func seedExpenses() []Expense {
	e := func(day int, amount int64, cat Category) Expense {
		return Expense{Date: d(2026, 6, day), Amount: amount, Category: cat}
	}
	exps := []Expense{
		e(1, 18_000, CatMakan), e(1, 12_000, CatJajan), e(1, 15_000, CatMakan),
		e(2, 22_000, CatMakan), e(2, 35_000, CatJajan),
		e(3, 30_000, CatMakan), e(3, 8_000, CatLainnya),
		e(4, 25_000, CatMakan), e(4, 40_000, CatBelanja),
		e(5, 20_000, CatMakan), e(5, 10_000, CatJajan), e(5, 20_000, CatCash),
		e(6, 145_000, CatBelanja), e(6, 65_000, CatMakan),
		e(7, 85_000, CatMakan), e(7, 28_000, CatJajan), e(7, 160_000, CatBelanja),
		e(8, 19_000, CatMakan), e(8, 9_000, CatJajan),
		e(9, 26_000, CatMakan), e(9, 30_000, CatLainnya),
		e(10, 24_000, CatMakan), e(10, 14_000, CatJajan),
		e(11, 32_000, CatMakan), e(11, 50_000, CatCash),
		e(12, 16_000, CatMakan),
	}
	// Subscription payments are ordinary Langganan expenses (v2 model).
	exps = append(exps,
		Expense{Date: d(2026, 6, 5), Amount: 186_000, Category: CatLangganan, SubscriptionID: "s1"},
		Expense{Date: d(2026, 6, 10), Amount: 65_000, Category: CatLangganan, SubscriptionID: "s2"},
	)
	return exps
}

func TestComputeMonth_June2026Seed(t *testing.T) {
	in := MonthInput{
		Year: 2026, Month: 6,
		Today:         d(2026, 6, 16),
		Config:        seedConfig(),
		Subscriptions: seedSubs(),
		Expenses:      seedExpenses(),
	}
	got := ComputeMonth(in)

	if len(got.Weeks) != 4 {
		t.Fatalf("len(Weeks) = %d, want 4", len(got.Weeks))
	}
	if len(got.Weekends) != 4 {
		t.Fatalf("len(Weekends) = %d, want 4", len(got.Weekends))
	}

	wantWeekSpent := []int64{552_000, 190_000, 0, 0}
	wantWeekLeft := []int64{48_000, 410_000, 600_000, 600_000}
	wantWeekState := []State{StatePast, StatePast, StateCurrent, StateFuture}
	for i, w := range got.Weeks {
		if w.Budget != 600_000 {
			t.Errorf("week %d budget = %d, want 600000", i, w.Budget)
		}
		if w.Spent != wantWeekSpent[i] {
			t.Errorf("week %d spent = %d, want %d", i, w.Spent, wantWeekSpent[i])
		}
		if w.Left != wantWeekLeft[i] {
			t.Errorf("week %d left = %d, want %d", i, w.Left, wantWeekLeft[i])
		}
		if w.State != wantWeekState[i] {
			t.Errorf("week %d state = %s, want %s", i, w.State, wantWeekState[i])
		}
	}

	wantWkndSpent := []int64{178_000, 0, 0, 0}
	wantWkndLeft := []int64{22_000, 200_000, 200_000, 200_000}
	wantWkndState := []State{StatePast, StatePast, StateFuture, StateFuture}
	for i, w := range got.Weekends {
		if w.Budget != 200_000 {
			t.Errorf("weekend %d budget = %d, want 200000", i, w.Budget)
		}
		if w.Spent != wantWkndSpent[i] {
			t.Errorf("weekend %d spent = %d, want %d", i, w.Spent, wantWkndSpent[i])
		}
		if w.Left != wantWkndLeft[i] {
			t.Errorf("weekend %d left = %d, want %d", i, w.Left, wantWkndLeft[i])
		}
		if w.State != wantWkndState[i] {
			t.Errorf("weekend %d state = %s, want %s", i, w.State, wantWkndState[i])
		}
	}

	checks := []struct {
		name string
		got  int64
		want int64
	}{
		{"ShopBudget", got.ShopBudget, 2_400_000},
		{"ShopSpent", got.ShopSpent, 742_000},
		{"WkndBudget", got.WkndBudget, 800_000},
		{"WkndSpent", got.WkndSpent, 178_000},
		{"SubsAlloc", got.SubsAlloc, 330_000},
		{"LanggananSpent", got.LanggananSpent, 251_000},
		{"FlexSpent", got.FlexSpent, 38_000},
		{"FlexBudget", got.FlexBudget, 1_470_000},
		{"TotalSpent", got.TotalSpent, 1_209_000},
		{"Sisa", got.Sisa, 3_791_000},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}

	// rows: belanja, weekend, langganan, fleksibel. Fleksibel's row carries the
	// EFFECTIVE budget (flexBudget 1470000 + rollover 671000) so the card's
	// progress bar agrees with the over flag (§6.6); left = budget − spent.
	wantRows := []Row{
		{ID: EnvBelanja, Budget: 2_400_000, Spent: 742_000, Left: 1_658_000, Over: false},
		{ID: EnvWeekend, Budget: 800_000, Spent: 178_000, Left: 622_000, Over: false},
		{ID: EnvLangganan, Budget: 330_000, Spent: 251_000, Left: 79_000, Over: false},
		{ID: EnvFleksibel, Budget: 2_141_000, Spent: 38_000, Left: 2_103_000, Over: false},
	}
	if len(got.Rows) != 4 {
		t.Fatalf("len(Rows) = %d, want 4", len(got.Rows))
	}
	for i, r := range got.Rows {
		w := wantRows[i]
		if r.ID != w.ID || r.Budget != w.Budget || r.Spent != w.Spent || r.Left != w.Left || r.Over != w.Over {
			t.Errorf("row %d = %+v, want id=%s budget=%d spent=%d left=%d over=%v",
				i, r, w.ID, w.Budget, w.Spent, w.Left, w.Over)
		}
	}
}

// ---- Month boundary attribution (§6.2) -----------------------------------

func TestComputeMonth_WeekOwnedByFridaysMonth(t *testing.T) {
	// A belanja expense on Mon Jun 29 2026 belongs to the week of Fri Jul 3
	// (owned by July). Viewing June, it must NOT count in any June week.
	exp := []Expense{{Date: d(2026, 6, 29), Amount: 100_000, Category: CatBelanja}}

	june := ComputeMonth(MonthInput{Year: 2026, Month: 6, Today: d(2026, 6, 16), Config: seedConfig(), Expenses: exp})
	if june.ShopSpent != 0 {
		t.Errorf("June ShopSpent = %d, want 0 (Jun 29 belongs to July's week)", june.ShopSpent)
	}
	if len(june.Weeks) != 4 {
		t.Fatalf("June weeks = %d, want 4", len(june.Weeks))
	}

	// Viewing July, the same expense counts in July's first week (Fri Jul 3,
	// range Mon Jun 29 – Sun Jul 5).
	july := ComputeMonth(MonthInput{Year: 2026, Month: 7, Today: d(2026, 7, 16), Config: seedConfig(), Expenses: exp})
	if july.ShopSpent != 100_000 {
		t.Errorf("July ShopSpent = %d, want 100000 (Jun 29 belongs to July's first week)", july.ShopSpent)
	}
	if july.Weeks[0].Spent != 100_000 {
		t.Errorf("July first week spent = %d, want 100000", july.Weeks[0].Spent)
	}
}

func TestComputeMonth_WeekendOwnedBySaturdaysMonth(t *testing.T) {
	// A weekend-envelope expense on Sun Jun 28 2026 belongs to the weekend of
	// Sat Jun 27 (owned by June). Viewing July, it must not count.
	exp := []Expense{{Date: d(2026, 6, 28), Amount: 90_000, Category: CatMakan}} // Sunday → weekend

	july := ComputeMonth(MonthInput{Year: 2026, Month: 7, Today: d(2026, 7, 16), Config: seedConfig(), Expenses: exp})
	if july.WkndSpent != 0 {
		t.Errorf("July WkndSpent = %d, want 0 (Jun 28 belongs to June's weekend)", july.WkndSpent)
	}

	june := ComputeMonth(MonthInput{Year: 2026, Month: 6, Today: d(2026, 6, 16), Config: seedConfig(), Expenses: exp})
	if june.WkndSpent != 90_000 {
		t.Errorf("June WkndSpent = %d, want 90000", june.WkndSpent)
	}
}

func TestComputeMonth_FlexAndLanggananByCalendarMonth(t *testing.T) {
	// Flex (Lainnya weekday) and Langganan are attributed by calendar month
	// only, regardless of week boundaries. Jun 29 is in June's calendar.
	exp := []Expense{
		{Date: d(2026, 6, 29), Amount: 12_000, Category: CatLainnya},                         // weekday flex
		{Date: d(2026, 6, 30), Amount: 50_000, Category: CatLangganan, SubscriptionID: "s1"}, // langganan
	}
	june := ComputeMonth(MonthInput{Year: 2026, Month: 6, Today: d(2026, 6, 16), Config: seedConfig(), Expenses: exp})
	if june.FlexSpent != 12_000 {
		t.Errorf("June FlexSpent = %d, want 12000", june.FlexSpent)
	}
	if june.LanggananSpent != 50_000 {
		t.Errorf("June LanggananSpent = %d, want 50000", june.LanggananSpent)
	}
}

// ---- Negative / edge cases ------------------------------------------------

func TestComputeMonth_NegativeWeekLeftAndOverflowRow(t *testing.T) {
	// Overspend a week: belanja 700000 in week of Fri Jun 5 (budget 600000).
	exp := []Expense{{Date: d(2026, 6, 3), Amount: 700_000, Category: CatBelanja}}
	got := ComputeMonth(MonthInput{Year: 2026, Month: 6, Today: d(2026, 6, 16), Config: seedConfig(), Expenses: exp})
	if got.Weeks[0].Left != -100_000 {
		t.Errorf("week0 left = %d, want -100000", got.Weeks[0].Left)
	}
	if got.Rows[0].Over {
		t.Errorf("belanja row over = true, but total shop budget (2.4M) > spent (700k)")
	}
}

func TestComputeMonth_NegativeFlexBudget(t *testing.T) {
	// monthly too small to cover shop+weekend budgets → flexBudget negative.
	cfg := Config{Monthly: 1_000_000, ShopWeekly: 600_000, WeekendBudget: 200_000}
	got := ComputeMonth(MonthInput{Year: 2026, Month: 6, Today: d(2026, 6, 16), Config: cfg})
	// shopBudget = 600000*4 = 2.4M; wkndBudget = 200000*4 = 0.8M; flex = 1M - 2.4M - 0.8M = -2.2M
	if got.FlexBudget != -2_200_000 {
		t.Errorf("FlexBudget = %d, want -2200000", got.FlexBudget)
	}
}

func TestComputeMonth_EmptyMonth(t *testing.T) {
	got := ComputeMonth(MonthInput{Year: 2026, Month: 6, Today: d(2026, 6, 16), Config: seedConfig()})
	if got.TotalSpent != 0 {
		t.Errorf("TotalSpent = %d, want 0", got.TotalSpent)
	}
	if got.Sisa != 5_000_000 {
		t.Errorf("Sisa = %d, want 5000000", got.Sisa)
	}
}

// ---- Rollover into Fleksibel (§6.6) ---------------------------------------

func TestComputeMonth_RolloverJune2026Seed(t *testing.T) {
	// Mid-month (today Jun 16): closed sources are weeks 1–2, weekends 1–2, and
	// the two paid subscriptions (Netflix +1000, Spotify 55000−65000 = −10000).
	got := ComputeMonth(MonthInput{
		Year: 2026, Month: 6,
		Today:         d(2026, 6, 16),
		Config:        seedConfig(),
		Subscriptions: seedSubs(),
		Expenses:      seedExpenses(),
	})

	// 48000 + 410000 (weeks) + 22000 + 200000 (weekends) + 1000 − 10000 (subs).
	if got.Rollover != 671_000 {
		t.Errorf("Rollover = %d, want 671000", got.Rollover)
	}

	want := []RolloverItem{
		{Type: RolloverWeek, Start: d(2026, 6, 1), End: d(2026, 6, 7), Amount: 48_000},
		{Type: RolloverWeek, Start: d(2026, 6, 8), End: d(2026, 6, 14), Amount: 410_000},
		{Type: RolloverWeekend, Start: d(2026, 6, 6), End: d(2026, 6, 7), Amount: 22_000},
		{Type: RolloverWeekend, Start: d(2026, 6, 13), End: d(2026, 6, 14), Amount: 200_000},
		{Type: RolloverSubscription, Name: "Netflix", Amount: 1_000},
		{Type: RolloverSubscription, Name: "Spotify", Amount: -10_000},
	}
	if len(got.RolloverItems) != len(want) {
		t.Fatalf("len(RolloverItems) = %d, want %d (unpaid subs must be absent)", len(got.RolloverItems), len(want))
	}
	for i, it := range got.RolloverItems {
		w := want[i]
		if it.Type != w.Type || it.Amount != w.Amount || it.Name != w.Name {
			t.Errorf("item %d = %+v, want type=%s name=%q amount=%d", i, it, w.Type, w.Name, w.Amount)
		}
		if w.Type != RolloverSubscription && (!timeutil.SameDate(it.Start, w.Start) || !timeutil.SameDate(it.End, w.End)) {
			t.Errorf("item %d range = %s..%s, want %s..%s",
				i, it.Start.Format("2006-01-02"), it.End.Format("2006-01-02"),
				w.Start.Format("2006-01-02"), w.End.Format("2006-01-02"))
		}
	}

	// Rollover must not change sisa (already asserted in the seed test) and the
	// planned flex budget stays untouched.
	if got.FlexBudget != 1_470_000 {
		t.Errorf("FlexBudget = %d, want 1470000 (rollover must not change the plan)", got.FlexBudget)
	}
}

func TestComputeMonth_RolloverMonthStart(t *testing.T) {
	// On the month's first day nothing is closed: no past pills, no paid subs.
	got := ComputeMonth(MonthInput{Year: 2026, Month: 6, Today: d(2026, 6, 1), Config: seedConfig()})
	if got.Rollover != 0 {
		t.Errorf("Rollover = %d, want 0", got.Rollover)
	}
	if len(got.RolloverItems) != 0 {
		t.Errorf("len(RolloverItems) = %d, want 0", len(got.RolloverItems))
	}
	// flexBudget = 5M − 2.4M − 0.8M (no subs) = 1.8M; left is untouched.
	if got.Rows[3].Left != 1_800_000 {
		t.Errorf("fleksibel left = %d, want 1800000", got.Rows[3].Left)
	}
}

func TestComputeMonth_RolloverPastMonthView(t *testing.T) {
	// Viewing June from July: every pill is past and payments are final, so the
	// rollover is the month's full week+weekend+subscription leftover.
	got := ComputeMonth(MonthInput{
		Year: 2026, Month: 6,
		Today:         d(2026, 7, 20),
		Config:        seedConfig(),
		Subscriptions: seedSubs(),
		Expenses:      seedExpenses(),
	})
	// weeks 1658000 + weekends 622000 + subs (1000 − 10000).
	if got.Rollover != 2_271_000 {
		t.Errorf("Rollover = %d, want 2271000", got.Rollover)
	}
	if len(got.RolloverItems) != 10 { // 4 weeks + 4 weekends + 2 paid subs
		t.Errorf("len(RolloverItems) = %d, want 10", len(got.RolloverItems))
	}
	if got.Rows[3].Left != 3_703_000 { // 1470000 + 2271000 − 38000
		t.Errorf("fleksibel left = %d, want 3703000", got.Rows[3].Left)
	}
}

func TestComputeMonth_NegativeRolloverPushesFlexOver(t *testing.T) {
	// A heavily overspent past week drags fleksibel below zero even though
	// nothing was spent from fleksibel itself.
	exp := []Expense{{Date: d(2026, 6, 3), Amount: 4_000_000, Category: CatBelanja}}
	got := ComputeMonth(MonthInput{Year: 2026, Month: 6, Today: d(2026, 6, 16), Config: seedConfig(), Expenses: exp})
	// week1 −3.4M + week2 600K + weekends 2×200K = −2.4M.
	if got.Rollover != -2_400_000 {
		t.Errorf("Rollover = %d, want -2400000", got.Rollover)
	}
	flex := got.Rows[3]
	if flex.Budget != -600_000 { // effective: 1800000 − 2400000
		t.Errorf("fleksibel effective budget = %d, want -600000", flex.Budget)
	}
	if flex.Left != -600_000 { // −600000 − 0 spent
		t.Errorf("fleksibel left = %d, want -600000", flex.Left)
	}
	if !flex.Over {
		t.Error("fleksibel over = false, want true (negative rolled-up left)")
	}
}

func TestComputeMonth_RolloverIncludesZeroAmountItems(t *testing.T) {
	// A past week closed exactly on budget still appears in the breakdown.
	exp := []Expense{{Date: d(2026, 6, 3), Amount: 600_000, Category: CatBelanja}}
	got := ComputeMonth(MonthInput{Year: 2026, Month: 6, Today: d(2026, 6, 16), Config: seedConfig(), Expenses: exp})
	if len(got.RolloverItems) != 4 { // weeks 1–2 + weekends 1–2
		t.Fatalf("len(RolloverItems) = %d, want 4", len(got.RolloverItems))
	}
	if got.RolloverItems[0].Amount != 0 {
		t.Errorf("week1 item amount = %d, want 0 (zero items are part of the audit trail)", got.RolloverItems[0].Amount)
	}
}

func TestComputeMonth_RolloverFollowsWeekOwnership(t *testing.T) {
	// A boundary week's leftover rolls in the month that OWNS the week (its
	// Friday), same as its spent. Jun 29 belongs to July's first week.
	exp := []Expense{{Date: d(2026, 6, 29), Amount: 100_000, Category: CatBelanja}}

	july := ComputeMonth(MonthInput{Year: 2026, Month: 7, Today: d(2026, 8, 5), Config: seedConfig(), Expenses: exp})
	first := july.RolloverItems[0]
	if first.Type != RolloverWeek || !timeutil.SameDate(first.Start, d(2026, 6, 29)) || first.Amount != 500_000 {
		t.Errorf("July first rollover item = %+v, want week starting 2026-06-29 with amount 500000", first)
	}

	june := ComputeMonth(MonthInput{Year: 2026, Month: 6, Today: d(2026, 8, 5), Config: seedConfig(), Expenses: exp})
	// June's rollover is untouched by the Jun-29 expense: 4 full weeks + 4 full
	// weekends = 2.4M + 0.8M.
	if june.Rollover != 3_200_000 {
		t.Errorf("June Rollover = %d, want 3200000 (Jun-29 belongs to July's week)", june.Rollover)
	}
}

// ---- Per-subscription status (§6.4) --------------------------------------

func TestSubscriptionStatus(t *testing.T) {
	exps := seedExpenses()
	subs := seedSubs()

	// Netflix (s1) paid 186000 on Jun 5; alloc 187000 → diff +1000, paid.
	netflix := SubscriptionStatus(subs[0], exps, 2026, 6)
	if !netflix.Paid {
		t.Fatalf("Netflix should be paid")
	}
	if netflix.PaidAmount != 186_000 {
		t.Errorf("Netflix paid amount = %d, want 186000", netflix.PaidAmount)
	}
	if !timeutil.SameDate(netflix.PaidDate, d(2026, 6, 5)) {
		t.Errorf("Netflix paid date = %v, want 2026-06-05", netflix.PaidDate)
	}
	if netflix.Diff != 1_000 {
		t.Errorf("Netflix diff = %d, want 1000", netflix.Diff)
	}
	if netflix.Status != StatusPaid {
		t.Errorf("Netflix status = %s, want paid", netflix.Status)
	}

	// YouTube (s3) unpaid.
	youtube := SubscriptionStatus(subs[2], exps, 2026, 6)
	if youtube.Paid {
		t.Fatalf("YouTube should be unpaid")
	}
	if youtube.Status != StatusUnpaid {
		t.Errorf("YouTube status = %s, want unpaid", youtube.Status)
	}
}
