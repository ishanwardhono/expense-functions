package monthly

import (
	"context"
	"time"

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

	monthData := getPayPeriodMonth(cfg.Time)
	expenses, err := getCurrentMonthExpense(ctx, db, monthData.year, monthData.month)
	if err != nil {
		return resp, err
	}

	totalWeeks := monthData.getTotalWeeks()
	budget := monthData.getBudget(totalWeeks, MaxExpense)
	remaining := calculateRemainingExpense(expenses, budget)

	return expenseResponse{
		Year:       monthData.year,
		Month:      monthData.month,
		MonthLabel: getMonthName(monthData.month),
		DateRange:  getDateRange(monthData.year, monthData.month),
		TotalWeeks: totalWeeks,
		Budget:     common.FormatRupiah(budget),
		Remaining:  remaining,
	}, nil
}

func calculateRemainingExpense(expenses MonthlyExpenses, maxExpense int64) expenseRemaining {
	totalExpense := expenses.GetTotalExpense()
	remainingAmount := maxExpense - totalExpense

	return expenseRemaining{
		Total:   common.ToDataLabel(remainingAmount, false),
		Details: expenses.ToDetailsResponse(),
	}
}

func Recapitulation(ctx context.Context, db *sqlx.DB, t time.Time) (RecapResp, error) {
	monthData := getPayPeriodMonth(t)
	totalWeeks := monthData.getTotalWeeks()
	budget := monthData.getBudget(totalWeeks, MaxExpense)

	totalExpenseAmount, err := getSumMonthExpense(ctx, db, monthData.year, monthData.month)
	if err != nil {
		return RecapResp{}, err
	}

	return RecapResp{
		Year:       monthData.year,
		Month:      monthData.month,
		MonthLabel: getMonthName(monthData.month),
		Amount:     totalExpenseAmount,
		Remaining:  budget - totalExpenseAmount,
		TotalWeeks: totalWeeks,
		StartYear:  monthData.year,
		EndYear:    monthData.year,
		StartWeek:  monthData.startWeek,
		EndWeek:    monthData.endWeek,
	}, nil
}
