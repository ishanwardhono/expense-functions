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
	weeklyExpense, err := getCurrentWeekExpense(ctx, db, weekData)
	if err != nil {
		return resp, err
	}

	remaining := calculateRemainingExpense(weekData.day, weeklyExpense, cfg.maxExpense)
	return expenseResponse{
		Year:      weekData.year,
		Week:      weekData.week,
		DayLabel:  mapDayLabel[weekData.day],
		DateRange: getDateRange(weekData.year, weekData.week),
		Remaining: remaining,
	}, nil
}

func calculateRemainingExpense(day int, expense WeeklyExpense, maxExpense int64) expenseRemaining {
	weekdayRemaining := maxExpense - expense.Weekday
	weekendRemaining := maxExpense - expense.Weekend

	response := expenseRemaining{
		Weekday: toDataLabel(weekdayRemaining, day >= 5),
		Weekend: toDataLabel(weekendRemaining, false),
	}

	// If today is a weekday (Monday to Friday)
	if day < 5 {
		response.weekdayExpense(day, weekdayRemaining, weekendRemaining)
		return response
	}

	// If today is a weekend (Saturday or Sunday)
	response.weekendExpense(day, weekendRemaining)
	return response
}

func (r *expenseRemaining) weekdayExpense(day int, weekdayRemaining, weekendRemaining int64) {
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
	r.weekendExpense(day, weekendRemaining)
}

func (r *expenseRemaining) weekendExpense(day int, weekendRemaining int64) {
	strDay := GaAdaJajanLabel
	if weekendRemaining > 0 {
		if day <= 5 {
			weekendRemaining /= 2
		}
		strDay = formatRupiah(weekendRemaining)
	}
	if day <= 5 {
		r.Days.Sabtu = strDay
	}
	r.Days.Minggu = strDay
}
