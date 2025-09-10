package weekly

import (
	"context"

	"github.com/ishanwardhono/expense-function/common"
	"github.com/jmoiron/sqlx"
)

func Get(ctx context.Context) (expenseResponse, error) {
	resp := expenseResponse{}

	cfg, err := common.LoadConfig()
	if err != nil {
		return resp, err
	}

	db, err := common.ConnectDatabase(cfg.DbConfig)
	if err != nil {
		return resp, err
	}
	defer db.Close()

	weekData := getWeekData(cfg.Time)
	expenses, err := getCurrentWeekExpense(ctx, db, weekData.year, weekData.week)
	if err != nil {
		return resp, err
	}

	remaining := calculateRemainingExpense(weekData.day, expenses, MaxExpense)
	return expenseResponse{
		Year:      weekData.year,
		Week:      weekData.week,
		DayLabel:  mapDayLabel[weekData.day],
		DateRange: getDateRange(weekData.year, weekData.week),
		Remaining: remaining,
	}, nil
}

func calculateRemainingExpense(day int, expenses Expenses, maxExpense int64) expenseRemaining {
	weekdayExpense, saturdayExpense, sundayExpense := expenses.GetDayExpenses()
	weekdayRemaining := maxExpense - weekdayExpense
	saturdayRemaining := (maxExpense / 2) - saturdayExpense
	sundayRemaining := (maxExpense / 2) - sundayExpense

	response := expenseRemaining{
		Weekday:  common.ToDataLabel(weekdayRemaining, day >= 5),
		Saturday: common.ToDataLabel(saturdayRemaining, day >= 6),
		Sunday:   common.ToDataLabel(sundayRemaining, false),
		Details:  expenses.ToDetailsResponse(),
	}

	if day < 5 {
		response.weekdayExpense(day, weekdayRemaining)
	}

	response.weekendExpense(day, saturdayRemaining, sundayRemaining)
	return response
}

func (r *expenseRemaining) weekdayExpense(day int, weekdayRemaining int64) {
	remainingDay := 5 - day
	remainingPerDay := weekdayRemaining / int64(remainingDay)
	days := make([]string, 5)
	for i := day; i < 5; i++ {
		strDay := GaAdaJajanLabel
		if remainingPerDay > 0 {
			strDay = common.FormatRupiah(remainingPerDay)
		}
		days[i] = strDay
	}

	r.Days.Senin = days[0]
	r.Days.Selasa = days[1]
	r.Days.Rabu = days[2]
	r.Days.Kamis = days[3]
	r.Days.Jumat = days[4]
}

func (r *expenseRemaining) weekendExpense(day int, saturdayRemaining, sundayRemaining int64) {
	if day <= 5 {
		r.Days.Sabtu = common.FormatRupiah(saturdayRemaining)
	}
	r.Days.Minggu = common.FormatRupiah(sundayRemaining)
}

func Recapitulation(ctx context.Context, db *sqlx.DB, startYear, startWeek, endYear, endWeek int) ([]RecapResp, error) {
	weeklyRecaps, err := getSumWeekExpense(ctx, db, startYear, startWeek, endYear, endWeek)
	if err != nil {
		return nil, err
	}

	return weeklyRecaps.ToRecapResponse(), nil
}
