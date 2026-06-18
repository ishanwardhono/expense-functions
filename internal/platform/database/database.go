// Package database opens the CockroachDB connection used by the function.
//
// The serverless (Cloud Functions) pattern uses a single connection
// (SetMaxOpenConns(1)) and sslmode=verify-full (requires the CA cert).
package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver

	"github.com/ishanwardhono/expense-function/internal/platform/config"
)

// Connect opens and verifies a CockroachDB connection.
func Connect(cfg config.Database) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslrootcert=%s sslmode=verify-full",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLRootCert,
	)

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// Single connection: the serverless function pattern.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}
