package weekly

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func connectDatabase(cfg *config) (*sql.DB, error) {
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslrootcert=%s sslmode=verify-full",
		cfg.databaseHost, cfg.databasePort, cfg.databaseUser, cfg.databasePassword, cfg.databaseName, cfg.databaseSSLRootCert)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(time.Second * time.Duration(cfg.databaseTimeout))

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func getCurrentWeekExpense(ctx context.Context, db *sql.DB, weekData WeekData) (WeeklyExpense, error) {
	var expense WeeklyExpense
	query := `SELECT id, year, week, weekday, weekend, created_time FROM weekly_expense
			  WHERE year = $1 AND week = $2 LIMIT 1`
	err := db.QueryRowContext(ctx, query, weekData.year, weekData.week).Scan(
		&expense.Id, &expense.Year, &expense.Week, &expense.Weekday, &expense.Weekend, &expense.CreatedTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return insertCurrentWeekExpense(ctx, db, weekData)
		}
		return WeeklyExpense{}, fmt.Errorf("failed to get current week expense: %w", err)
	}
	return expense, nil
}

func insertCurrentWeekExpense(ctx context.Context, db *sql.DB, weekData WeekData) (WeeklyExpense, error) {
	var expense WeeklyExpense
	query := `INSERT INTO weekly_expense (year, week, created_time)
			  VALUES ($1, $2, $3)
			  RETURNING id, year, week, weekday, weekend, created_time`
	err := db.QueryRowContext(ctx, query, weekData.year, weekData.week, now()).Scan(
		&expense.Id, &expense.Year, &expense.Week, &expense.Weekday, &expense.Weekend, &expense.CreatedTime)
	if err != nil {
		return WeeklyExpense{}, fmt.Errorf("failed to insert current week expense: %w", err)
	}
	return expense, nil
}

func addWeekdayExpense(ctx context.Context, db *sql.DB, year, week int, weekdayAmount int64) error {
	query := `UPDATE weekly_expense SET weekday = weekday + $1 
			  WHERE year = $2 AND week = $3`
	result, err := db.ExecContext(ctx, query, weekdayAmount, year, week)
	if err != nil {
		return fmt.Errorf("failed to add weekday expense: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no weekly expense record found for year %d, week %d", year, week)
	}

	return nil
}

func addWeekendExpense(ctx context.Context, db *sql.DB, year, week int, weekendAmount int64) error {
	query := `UPDATE weekly_expense SET weekend = weekend + $1 
			  WHERE year = $2 AND week = $3`
	result, err := db.ExecContext(ctx, query, weekendAmount, year, week)
	if err != nil {
		return fmt.Errorf("failed to add weekend expense: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no weekly expense record found for year %d, week %d", year, week)
	}

	return nil
}
