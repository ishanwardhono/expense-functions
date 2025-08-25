package common

import (
	"testing"
	"time"
)

func TestNow(t *testing.T) {
	result := Now()

	// Check that the result is in Jakarta timezone
	expectedLoc, _ := time.LoadLocation("Asia/Jakarta")
	if result.Location().String() != expectedLoc.String() {
		t.Errorf("Expected timezone %s, got %s", expectedLoc.String(), result.Location().String())
	}

	// Check that the time is reasonable (within last minute)
	now := time.Now().In(expectedLoc)
	diff := now.Sub(result)
	if diff > time.Minute || diff < -time.Minute {
		t.Errorf("Time difference too large: %v", diff)
	}
}

func TestLoc(t *testing.T) {
	expectedLoc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("Failed to load expected location: %v", err)
	}

	if Loc.String() != expectedLoc.String() {
		t.Errorf("Expected location %s, got %s", expectedLoc.String(), Loc.String())
	}
}
