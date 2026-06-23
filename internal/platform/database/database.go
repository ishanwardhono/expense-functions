// Package database opens the CockroachDB connection used by the function.
//
// The serverless (Cloud Functions) pattern uses a single connection
// (SetMaxOpenConns(1)). Production uses sslmode=verify-full (requires the CA
// cert); local dev against an insecure node uses sslmode=disable (DB_SSL_MODE).
package database

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver

	"github.com/ishanwardhono/expense-function/internal/platform/config"
)

// Connect opens and verifies a CockroachDB connection.
func Connect(cfg config.Database) (*sqlx.DB, error) {
	// Build the DSN from non-empty parts only. An empty unquoted value (e.g.
	// password= for a local insecure root) makes lib/pq mis-tokenize and drop
	// the following field (dbname), so optional empty fields are omitted.
	parts := []string{
		"host=" + cfg.Host,
		"port=" + cfg.Port,
		"user=" + cfg.User,
		"dbname=" + cfg.Name,
		"sslmode=" + cfg.SSLMode,
	}
	if cfg.Password != "" {
		parts = append(parts, "password="+cfg.Password)
	}
	// sslrootcert is only meaningful for the verifying SSL modes; omit it for
	// sslmode=disable so a local insecure node needs no CA cert.
	if cfg.SSLMode != "disable" {
		parts = append(parts, "sslrootcert="+cfg.SSLRootCert)
	}
	dsn := strings.Join(parts, " ")

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
