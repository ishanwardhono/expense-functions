package weekly

import (
	"context"
	"time"
)

func Get(ctx context.Context) (expenseResponse, error) {
	resp := expenseResponse{}

	cfg, err := loadConfig(ctx)
	if err != nil {
		return resp, err
	}

	db, err := connectDatabase(ctx, cfg)
	if err != nil {
		return resp, err
	}
	defer db.Close()

	weekData := getCurrentWeekData()
	weeklyExpense, err := getCurrentWeekExpense(ctx, db, weekData)
	if err != nil {
		return resp, err
	}

	remaining := calculateRemainingExpense(weekData.day, weeklyExpense, cfg.maxExpense)
	return expenseResponse{
		Year:      weekData.year,
		Week:      weekData.week,
		DayLabel:  mapDayLabel[weekData.day],
		Remaining: remaining,
	}, nil
}

func getCurrentWeekData() WeekData {
	now := time.Now()
	year, week := now.ISOWeek()
	day := int(now.Weekday())
	return WeekData{year: year, week: week, day: day}
}

func calculateRemainingExpense(day int, expense WeeklyExpense, maxExpense int64) expenseRemaining {
	weekdayRemaining := maxExpense - expense.Weekday
	weekendRemaining := maxExpense - expense.Weekend

	response := expenseRemaining{
		Weekday: formatRupiah(weekdayRemaining),
		Weekend: formatRupiah(weekendRemaining),
		Days:    make([]string, 7),
	}

	// If today is a weekday (Monday to Friday)
	if day >= 1 && day <= 5 {
		response.weekdayExpense(day, weekdayRemaining, weekendRemaining)
		return response
	}

	// If today is a weekend (Saturday or Sunday)
	response.weekendExpense(day, weekendRemaining)
	return response
}

func (r *expenseRemaining) weekdayExpense(day int, weekdayRemaining, weekendRemaining int64) {
	weekdayRemainingDay := 6 - day
	weekdayRemainingPerDay := weekdayRemaining / int64(weekdayRemainingDay)
	for i := day; i <= 5; i++ {
		strDay := "Ga ada jajan"
		if weekdayRemainingPerDay > 0 {
			strDay = formatRupiah(weekdayRemainingPerDay)
		}
		r.Days[i] = strDay
	}
	r.weekendExpense(day, weekendRemaining)
}

func (r *expenseRemaining) weekendExpense(day int, weekendRemaining int64) {
	weekendRemainingPerDay := weekendRemaining
	if day != 0 {
		weekendRemainingPerDay = weekendRemaining / 2
		r.Days[6] = formatRupiah(weekendRemainingPerDay)
	}
	r.Days[0] = formatRupiah(weekendRemainingPerDay)
}
