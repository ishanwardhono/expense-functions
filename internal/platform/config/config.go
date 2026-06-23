// Package config loads runtime configuration from the environment.
package config

import (
	"os"
	"time"

	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// Database holds the CockroachDB connection settings.
type Database struct {
	Host        string
	Port        string
	User        string
	Password    string
	Name        string
	SSLMode     string
	SSLRootCert string
}

// Config aggregates the resolved "now" and the database settings.
type Config struct {
	Now time.Time
	DB  Database
}

// Load reads the environment and resolves "now" (honoring the TIME override).
func Load() (*Config, error) {
	now, err := timeutil.LoadTime()
	if err != nil {
		return nil, err
	}
	return &Config{
		Now: now,
		DB:  LoadDatabase(),
	}, nil
}

// LoadDatabase reads the DB_* environment variables. DB_SSL_MODE defaults to
// "verify-full" (the production CockroachDB pattern); local dev against an
// insecure node sets it to "disable".
func LoadDatabase() Database {
	sslMode := os.Getenv("DB_SSL_MODE")
	if sslMode == "" {
		sslMode = "verify-full"
	}
	return Database{
		Host:        os.Getenv("DB_HOST"),
		Port:        os.Getenv("DB_PORT"),
		User:        os.Getenv("DB_USER"),
		Password:    os.Getenv("DB_PASSWORD"),
		Name:        os.Getenv("DB_NAME"),
		SSLMode:     sslMode,
		SSLRootCert: os.Getenv("DB_SSL_ROOT_CERT"),
	}
}
