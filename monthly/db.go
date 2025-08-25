package monthly

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func getCurrentMonthExpense(ctx context.Context, db *sqlx.DB, monthData monthData) (MonthlyExpenses, error) {
	var expenses MonthlyExpenses
	query := `SELECT id, year, month, amount, type, note, created_time FROM monthly_expense
			  WHERE year = $1 AND month = $2
			  ORDER BY created_time ASC
			`

	err := db.SelectContext(ctx, &expenses, query, monthData.year, monthData.month)
	if err != nil {
		return nil, fmt.Errorf("failed to get current month expenses: %w", err)
	}

	return expenses, nil
}

func addMonthlyExpense(ctx context.Context, db *sqlx.DB, expense MonthlyExpense) error {
	query := `INSERT INTO monthly_expense (id, year, month, amount, type, note, created_time) 
			  VALUES (:id, :year, :month, :amount, :type, :note, :created_time)
			`
	_, err := db.NamedExecContext(ctx, query, expense)
	if err != nil {
		return fmt.Errorf("failed to add monthly expense: %w", err)
	}
	return nil
}
