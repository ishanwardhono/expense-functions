package weekly

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func getCurrentWeekExpense(ctx context.Context, db *sqlx.DB, year, week int) (Expenses, error) {
	var expenses Expenses
	query := `SELECT id, year, week, day, amount, type, note, created_time FROM expense
			  WHERE year = $1 AND week = $2
			  ORDER BY created_time ASC
			`

	err := db.SelectContext(ctx, &expenses, query, year, week)
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

func getSumWeekExpense(ctx context.Context, db *sqlx.DB, startYear, startWeek, endYear, endWeek int) (WeeklySummaries, error) {
	var weeklySummaries WeeklySummaries
	query := `SELECT year, week, SUM(amount) as amount FROM expense
			  WHERE year >= $1 AND week >= $2 AND year <= $3 AND week <= $4
			  GROUP BY year, week
			  ORDER BY year, week
			`

	err := db.SelectContext(ctx, &weeklySummaries, query, startYear, startWeek, endYear, endWeek)
	if err != nil {
		return nil, fmt.Errorf("failed to get weekly expense summary: %w", err)
	}

	return weeklySummaries, nil
}
