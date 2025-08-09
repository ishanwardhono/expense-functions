package weekly

import (
	"context"
)

func Get(ctx context.Context) (expenseResponse, error) {
	resp := expenseResponse{}

	cfg, err := loadConfig()
	if err != nil {
		return resp, err
	}

	db, err := connectDatabase(cfg)
	if err != nil {
		return resp, err
	}
	defer db.Close()

	weekData := getWeekData(cfg.time)
	expenses, err := getCurrentWeekExpense(ctx, db, weekData)
	if err != nil {
		return resp, err
	}

	remaining := calculateRemainingExpense(weekData.day, expenses, cfg.maxExpense)
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
		Weekday:  toDataLabel(weekdayRemaining, day >= 5),
		Saturday: toDataLabel(saturdayRemaining, day >= 6),
		Sunday:   toDataLabel(sundayRemaining, false),
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
			strDay = formatRupiah(remainingPerDay)
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
		r.Days.Sabtu = formatRupiah(saturdayRemaining)
	}
	r.Days.Minggu = formatRupiah(sundayRemaining)
}
