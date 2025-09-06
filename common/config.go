package common

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type config struct {
	MaxExpense        int64
	MaxMonthlyExpense int64
	Time              time.Time
	DbConfig          *DatabaseConfig
}

func LoadConfig() (*config, error) {
	maxExpense, err := LoadMaxExpense()
	if err != nil {
		return nil, err
	}

	maxMonthlyExpense, err := LoadMaxMonthlyExpense()
	if err != nil {
		return nil, err
	}

	t, err := LoadTime()
	if err != nil {
		return nil, err
	}

	return &config{
		MaxExpense:        maxExpense,
		MaxMonthlyExpense: maxMonthlyExpense,
		Time:              t,
		DbConfig:          LoadDatabaseConfig(),
	}, nil
}

func LoadMaxExpense() (int64, error) {
	maxExpenseStr := os.Getenv("MAX_EXPENSE")
	maxExpense, err := strconv.ParseInt(maxExpenseStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid MAX_EXPENSE environment variable: %v", err)
	}
	return maxExpense, nil
}

func LoadMaxMonthlyExpense() (int64, error) {
	maxMonthlyExpenseStr := os.Getenv("MAX_MONTHLY_EXPENSE")
	maxMonthlyExpense, err := strconv.ParseInt(maxMonthlyExpenseStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid MAX_MONTHLY_EXPENSE environment variable: %v", err)
	}
	return maxMonthlyExpense, nil
}

func LoadTime() (time.Time, error) {
	t := Now()
	mockTimeStr := os.Getenv("TIME")
	if mockTimeStr != "" {
		var err error
		t, err = time.Parse(time.RFC3339, mockTimeStr)
		if err != nil {
			return t, fmt.Errorf("invalid TIME environment variable: %v", err)
		}
	}
	return t, nil
}
