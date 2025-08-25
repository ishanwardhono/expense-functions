package monthly

import (
	"context"

	"github.com/ishanwardhono/expense-function/common"
)

func Get(ctx context.Context) (expenseResponse, error) {
	resp := expenseResponse{}

	cfg, err := loadConfig()
	if err != nil {
		return resp, err
	}

	db, err := common.ConnectDatabase(cfg.dbConfig)
	if err != nil {
		return resp, err
	}
	defer db.Close()

	monthData := getPayPeriodMonth(cfg.time)
	expenses, err := getCurrentMonthExpense(ctx, db, monthData)
	if err != nil {
		return resp, err
	}

	totalWeeks := monthData.getTotalWeeks()
	budget := monthData.getBudget(totalWeeks, cfg.maxExpense)
	remaining := calculateRemainingExpense(expenses, budget)

	return expenseResponse{
		Year:       monthData.year,
		Month:      monthData.month,
		MonthLabel: getMonthName(monthData.month),
		DateRange:  getMonthlyDateRange(monthData.year, monthData.month),
		TotalWeeks: totalWeeks,
		Budget:     common.FormatRupiah(budget),
		Remaining:  remaining,
	}, nil
}

func calculateRemainingExpense(expenses MonthlyExpenses, maxExpense int64) expenseRemaining {
	totalExpense := expenses.GetTotalExpense()
	remainingAmount := maxExpense - totalExpense

	return expenseRemaining{
		Total:   toDataLabel(remainingAmount),
		Details: expenses.ToDetailsResponse(),
	}
}
