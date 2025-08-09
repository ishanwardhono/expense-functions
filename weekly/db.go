package weekly

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func connectDatabase(cfg *config) (*sqlx.DB, error) {
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslrootcert=%s sslmode=verify-full",
		cfg.databaseHost, cfg.databasePort, cfg.databaseUser, cfg.databasePassword, cfg.databaseName, cfg.databaseSSLRootCert)

	db, err := sqlx.Open("postgres", connectionString)
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

func getCurrentWeekExpense(ctx context.Context, db *sqlx.DB, weekData WeekData) (Expenses, error) {
	var expenses Expenses
	query := `SELECT id, year, week, day, amount, type, note, created_time FROM expense
			  WHERE year = $1 AND week = $2`

	err := db.SelectContext(ctx, &expenses, query, weekData.year, weekData.week)
	if err != nil {
		return nil, fmt.Errorf("failed to get current week expenses: %w", err)
	}

	return expenses, nil
}

func addWeekdayExpense(ctx context.Context, db *sqlx.DB, year, week int, weekdayAmount int64) error {
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

func addWeekendExpense(ctx context.Context, db *sqlx.DB, year, week int, weekendAmount int64) error {
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
