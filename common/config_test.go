package common

import (
	"os"
	"testing"
	"time"
)

func TestLoadMaxExpense(t *testing.T) {
	// Save original value to restore later
	originalValue := os.Getenv("MAX_EXPENSE")
	defer os.Setenv("MAX_EXPENSE", originalValue)

	tests := []struct {
		name        string
		envValue    string
		expected    int64
		expectError bool
	}{
		{"Valid value", "100000", 100000, false},
		{"Zero value", "0", 0, false},
		{"Invalid value", "not_a_number", 0, true},
		{"Empty value", "", 0, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Setenv("MAX_EXPENSE", test.envValue)

			result, err := LoadMaxExpense()

			if test.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != test.expected {
					t.Errorf("Expected %d, got %d", test.expected, result)
				}
			}
		})
	}
}

func TestLoadTime(t *testing.T) {
	// Save original value to restore later
	originalValue := os.Getenv("TIME")
	defer os.Setenv("TIME", originalValue)

	tests := []struct {
		name        string
		envValue    string
		expectError bool
	}{
		{"No TIME env var", "", false},
		{"Valid RFC3339 time", "2024-08-24T10:30:00Z", false},
		{"Invalid time format", "invalid_time", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			os.Setenv("TIME", test.envValue)

			result, err := LoadTime()

			if test.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if test.envValue != "" {
					expectedTime, _ := time.Parse(time.RFC3339, test.envValue)
					if !result.Equal(expectedTime) {
						t.Errorf("Expected %v, got %v", expectedTime, result)
					}
				}
			}
		})
	}
}
