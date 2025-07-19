package weekly

import (
	"context"
	"fmt"
	"os"
	"strconv"
)

type config struct {
	maxExpense   int64
	projectID    string
	databaseName string
}

func loadConfig(ctx context.Context) (*config, error) {
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

	return &config{
		maxExpense:   maxExpense,
		projectID:    projectID,
		databaseName: databaseName,
	}, nil
}
