package weekly

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type config struct {
	maxExpense   int64
	projectID    string
	databaseName string
	time         time.Time
}

func loadConfig() (*config, error) {
	maxExpenseStr := os.Getenv("MAX_EXPENSE")
	maxExpense, err := strconv.ParseInt(maxExpenseStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_EXPENSE environment variable: %v", err)
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT environment variable is required")
	}

	databaseName := os.Getenv("FIRESTORE_DATABASE")
	if databaseName == "" {
		return nil, fmt.Errorf("FIRESTORE_DATABASE environment variable is required")
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return nil, fmt.Errorf("failed to load Asia/Jakarta location: %v", err)
	}
	t := time.Now().In(loc)
	mockTimeStr := os.Getenv("TIME")
	if mockTimeStr != "" {
		t, err = time.Parse(time.RFC3339, mockTimeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid TIME environment variable: %v", err)
		}
	}

	return &config{
		maxExpense:   maxExpense,
		projectID:    projectID,
		databaseName: databaseName,
		time:         t,
	}, nil
}
