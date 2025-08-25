package common

import (
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DatabaseConfig struct {
	Host        string
	Port        string
	User        string
	Password    string
	Name        string
	SSLRootCert string
	Timeout     int
}

func LoadDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:        os.Getenv("DB_HOST"),
		Port:        os.Getenv("DB_PORT"),
		User:        os.Getenv("DB_USER"),
		Password:    os.Getenv("DB_PASSWORD"),
		Name:        os.Getenv("DB_NAME"),
		SSLRootCert: os.Getenv("DB_SSL_ROOT_CERT"),
	}
}

func ConnectDatabase(cfg *DatabaseConfig) (*sqlx.DB, error) {
	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslrootcert=%s sslmode=verify-full",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLRootCert)

	db, err := sqlx.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
