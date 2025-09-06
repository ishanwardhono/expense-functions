package weekly

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func getCurrentWeekExpense(ctx context.Context, db *sqlx.DB, weekData WeekData) (Expenses, error) {
	var expenses Expenses
	query := `SELECT id, year, week, day, amount, type, note, created_time FROM expense
			  WHERE year = $1 AND week = $2
			  ORDER BY created_time ASC
			`

	err := db.SelectContext(ctx, &expenses, query, weekData.year, weekData.week)
	if err != nil {
		return nil, fmt.Errorf("failed to get current week expenses: %w", err)
	}

	return expenses, nil
}

func addExpense(ctx context.Context, db *sqlx.DB, expense Expense) error {
	query := `INSERT INTO expense (id, year, week, day, amount, type, note, created_time) 
			  VALUES (:id, :year, :week, :day, :amount, :type, :note, :created_time)
			`
	_, err := db.NamedExecContext(ctx, query, expense)
	if err != nil {
		return fmt.Errorf("failed to add expense: %w", err)
	}
	return nil
}
