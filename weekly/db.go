package weekly

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func connectDatabase(cfg *config) (*sql.DB, error) {
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.databaseHost, cfg.databasePort, cfg.databaseUser, cfg.databasePassword, cfg.databaseName)

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
		&expense.id, &expense.year, &expense.week, &expense.weekday, &expense.weekend, &expense.createdTime)
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
	query := `INSERT INTO weekly_expense (year, week)
			  VALUES ($1, $2)
			  RETURNING id, year, week, weekday, weekend, created_time`
	err := db.QueryRowContext(ctx, query, weekData.year, weekData.week).Scan(
		&expense.id, &expense.year, &expense.week, &expense.weekday, &expense.weekend, &expense.createdTime)
	if err != nil {
		return WeeklyExpense{}, fmt.Errorf("failed to insert current week expense: %w", err)
	}
	return expense, nil
}
