package weekly

import (
	"fmt"
	"os"
	"strconv"
)

type config struct {
	databaseHost     string
	databasePort     string
	databaseUser     string
	databasePassword string
	databaseName     string
	databaseTimeout  int

	maxExpense int64
}

func loadConfig() (*config, error) {
	maxExpenseStr := os.Getenv("MAX_EXPENSE")
	maxExpense, err := strconv.ParseInt(maxExpenseStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_EXPENSE environment variable: %v", err)
	}

	dbTimeoutStr := os.Getenv("DB_TIMEOUT")
	dbTimeout, err := strconv.Atoi(dbTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid DB_TIMEOUT environment variable: %v", err)
	}

	return &config{
		databaseHost:     os.Getenv("DB_HOST"),
		databasePort:     os.Getenv("DB_PORT"),
		databaseUser:     os.Getenv("DB_USER"),
		databasePassword: os.Getenv("DB_PASSWORD"),
		databaseName:     os.Getenv("DB_NAME"),
		databaseTimeout:  dbTimeout,
		maxExpense:       maxExpense,
	}, nil
}
