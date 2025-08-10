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
