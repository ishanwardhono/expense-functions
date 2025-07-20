package weekly

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type config struct {
	maxExpense int64
	time       time.Time

	databaseHost        string
	databasePort        string
	databaseUser        string
	databasePassword    string
	databaseName        string
	databaseSSLRootCert string
	databaseTimeout     int
}

func loadConfig() (*config, error) {
	maxExpenseStr := os.Getenv("MAX_EXPENSE")
	maxExpense, err := strconv.ParseInt(maxExpenseStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_EXPENSE environment variable: %v", err)
	}

	t := now()
	mockTimeStr := os.Getenv("TIME")
	if mockTimeStr != "" {
		t, err = time.Parse(time.RFC3339, mockTimeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid TIME environment variable: %v", err)
		}
	}

	return &config{
		maxExpense:          maxExpense,
		databaseHost:        os.Getenv("DB_HOST"),
		databasePort:        os.Getenv("DB_PORT"),
		databaseUser:        os.Getenv("DB_USER"),
		databasePassword:    os.Getenv("DB_PASSWORD"),
		databaseName:        os.Getenv("DB_NAME"),
		databaseSSLRootCert: os.Getenv("DB_SSL_ROOT_CERT"),
		time:                t,
	}, nil
}
