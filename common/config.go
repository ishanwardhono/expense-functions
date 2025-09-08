package common

import (
	"fmt"
	"os"
	"time"
)

type config struct {
	Time     time.Time
	DbConfig *DatabaseConfig
}

func LoadConfig() (*config, error) {
	t, err := LoadTime()
	if err != nil {
		return nil, err
	}

	return &config{
		Time:     t,
		DbConfig: LoadDatabaseConfig(),
	}, nil
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
