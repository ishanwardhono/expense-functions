package weekly

import (
	"testing"
	"time"
)

func TestGetDateRange(t *testing.T) {
	tests := []struct {
		name     string
		dateStr  string
		expected string
	}{
		// Original test cases
		{
			name:     "2025-07-13 (Sunday of week 28)",
			dateStr:  "2025-07-13T07:00:00Z",
			expected: "7 - 13 Jul 2025",
		},
		{
			name:     "2025-07-14 (Monday of week 29)",
			dateStr:  "2025-07-14T07:00:00Z",
			expected: "14 - 20 Jul 2025",
		},
		{
			name:     "2025-07-20 (Sunday of week 29)",
			dateStr:  "2025-07-20T07:00:00Z",
			expected: "14 - 20 Jul 2025",
		},
		{
			name:     "2025-07-21 (Monday of week 30)",
			dateStr:  "2025-07-21T07:00:00Z",
			expected: "21 - 27 Jul 2025",
		},
		{
			name:     "2025-01-01 (Wednesday of week 1)",
			dateStr:  "2025-01-01T07:00:00Z",
			expected: "30 Dec 2024 - 5 Jan 2025",
		},
		{
			name:     "2025-11-28 (Friday of week 48)",
			dateStr:  "2025-11-28T07:00:00Z",
			expected: "24 - 30 Nov 2025",
		},
		{
			name:     "2025-10-28 (Tuesday of week 44)",
			dateStr:  "2025-10-28T07:00:00Z",
			expected: "27 Oct - 2 Nov 2025",
		},
		{
			name:     "2025-12-27 (Saturday of week 52)",
			dateStr:  "2025-12-27T07:00:00Z",
			expected: "22 - 28 Dec 2025",
		},
		{
			name:     "2025-12-28 (Sunday of week 52)",
			dateStr:  "2025-12-28T07:00:00Z",
			expected: "22 - 28 Dec 2025",
		},
		{
			name:     "2025-12-29 (Monday of week 1 2026)",
			dateStr:  "2025-12-29T07:00:00Z",
			expected: "29 Dec 2025 - 4 Jan 2026",
		},
		{
			name:     "2025-12-30 (Tuesday of week 1 2026)",
			dateStr:  "2025-12-30T07:00:00Z",
			expected: "29 Dec 2025 - 4 Jan 2026",
		},
		{
			name:     "2026-01-01 (Thursday of week 1)",
			dateStr:  "2026-01-01T07:00:00Z",
			expected: "29 Dec 2025 - 4 Jan 2026",
		},
		{
			name:     "2026-01-06 (Tuesday of week 2)",
			dateStr:  "2026-01-06T07:00:00Z",
			expected: "5 - 11 Jan 2026",
		},

		// Additional edge cases
		{
			name:     "2024-01-01 (Monday, week 1)",
			dateStr:  "2024-01-01T07:00:00Z",
			expected: "1 - 7 Jan 2024",
		},
		{
			name:     "2024-12-30 (Monday, week 1 2025)",
			dateStr:  "2024-12-30T07:00:00Z",
			expected: "30 Dec 2024 - 5 Jan 2025",
		},
		{
			name:     "2024-12-31 (Tuesday, week 1 2025)",
			dateStr:  "2024-12-31T07:00:00Z",
			expected: "30 Dec 2024 - 5 Jan 2025",
		},
		{
			name:     "2023-01-02 (Monday, week 1)",
			dateStr:  "2023-01-02T07:00:00Z",
			expected: "2 - 8 Jan 2023",
		},
		{
			name:     "2023-01-01 (Sunday, week 52 2022)",
			dateStr:  "2023-01-01T07:00:00Z",
			expected: "26 Dec 2022 - 1 Jan 2023",
		},
		{
			name:     "2027-01-01 (Friday, week 53 2026)",
			dateStr:  "2027-01-01T07:00:00Z",
			expected: "28 Dec 2026 - 3 Jan 2027",
		},
		{
			name:     "2027-01-04 (Monday, week 1)",
			dateStr:  "2027-01-04T07:00:00Z",
			expected: "4 - 10 Jan 2027",
		},
		{
			name:     "Leap year Feb 29, 2024",
			dateStr:  "2024-02-29T07:00:00Z",
			expected: "26 Feb - 3 Mar 2024",
		},
		{
			name:     "End of February non-leap year",
			dateStr:  "2025-02-28T07:00:00Z",
			expected: "24 Feb - 2 Mar 2025",
		},
		{
			name:     "Cross month March-April",
			dateStr:  "2025-03-31T07:00:00Z",
			expected: "31 Mar - 6 Apr 2025",
		},
		{
			name:     "Cross month April-May",
			dateStr:  "2025-04-28T07:00:00Z",
			expected: "28 Apr - 4 May 2025",
		},
		{
			name:     "Mid year June",
			dateStr:  "2025-06-15T07:00:00Z",
			expected: "9 - 15 Jun 2025",
		},
		{
			name:     "Cross month August-September",
			dateStr:  "2025-08-31T07:00:00Z",
			expected: "25 - 31 Aug 2025",
		},
		{
			name:     "Cross month September-October",
			dateStr:  "2025-09-29T07:00:00Z",
			expected: "29 Sep - 5 Oct 2025",
		},
		{
			name:     "Week 53 in 2020 (leap year with 53 weeks)",
			dateStr:  "2020-12-28T07:00:00Z",
			expected: "28 Dec 2020 - 3 Jan 2021",
		},
		{
			name:     "Last week of 2021 (week 52)",
			dateStr:  "2021-12-27T07:00:00Z",
			expected: "27 Dec 2021 - 2 Jan 2022",
		},
		{
			name:     "First Monday of 2022",
			dateStr:  "2022-01-03T07:00:00Z",
			expected: "3 - 9 Jan 2022",
		},
		{
			name:     "Last day of 2022 (Saturday, week 52)",
			dateStr:  "2022-12-31T07:00:00Z",
			expected: "26 Dec 2022 - 1 Jan 2023",
		},
		{
			name:     "New Year's Day 2023 (Sunday, week 52 2022)",
			dateStr:  "2023-01-01T07:00:00Z",
			expected: "26 Dec 2022 - 1 Jan 2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the test date
			testDate, err := time.Parse(time.RFC3339, tt.dateStr)
			if err != nil {
				t.Fatalf("Failed to parse test date: %v", err)
			}

			// Get week data from the test date
			weekData := getWeekData(testDate)

			// Get the date range
			result := getDateRange(weekData.year, weekData.week)

			if result != tt.expected {
				t.Errorf("getDateRange(%d, %d) = %q, want %q",
					weekData.year, weekData.week, result, tt.expected)
				t.Logf("Test date: %s, ISO Week: %d-%d",
					testDate.Format("2006-01-02"), weekData.year, weekData.week)
			}
		})
	}
}

func TestGetWeekData(t *testing.T) {
	tests := []struct {
		name         string
		dateStr      string
		expectedYear int
		expectedWeek int
		expectedDay  int
	}{
		{
			name:         "Monday",
			dateStr:      "2025-07-14T07:00:00Z",
			expectedYear: 2025,
			expectedWeek: 29,
			expectedDay:  0, // Monday = 0 in our system
		},
		{
			name:         "Sunday",
			dateStr:      "2025-07-20T07:00:00Z",
			expectedYear: 2025,
			expectedWeek: 29,
			expectedDay:  6, // Sunday = 6 in our system
		},
		{
			name:         "Year boundary - Dec 29 2025 is week 1 of 2026",
			dateStr:      "2025-12-29T07:00:00Z",
			expectedYear: 2026,
			expectedWeek: 1,
			expectedDay:  0, // Monday = 0
		},
		{
			name:         "Year boundary - Jan 1 2023 is week 52 of 2022",
			dateStr:      "2023-01-01T07:00:00Z",
			expectedYear: 2022,
			expectedWeek: 52,
			expectedDay:  6, // Sunday = 6
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDate, err := time.Parse(time.RFC3339, tt.dateStr)
			if err != nil {
				t.Fatalf("Failed to parse test date: %v", err)
			}

			weekData := getWeekData(testDate)

			if weekData.year != tt.expectedYear {
				t.Errorf("getWeekData().year = %d, want %d", weekData.year, tt.expectedYear)
			}
			if weekData.week != tt.expectedWeek {
				t.Errorf("getWeekData().week = %d, want %d", weekData.week, tt.expectedWeek)
			}
			if weekData.day != tt.expectedDay {
				t.Errorf("getWeekData().day = %d, want %d", weekData.day, tt.expectedDay)
			}
		})
	}
}

func TestFormatRupiah(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		expected string
	}{
		{
			name:     "Zero",
			amount:   0,
			expected: "Rp 0",
		},
		{
			name:     "Small amount",
			amount:   500,
			expected: "Rp 500",
		},
		{
			name:     "Thousands",
			amount:   1500,
			expected: "Rp 1.500",
		},
		{
			name:     "Millions",
			amount:   1500000,
			expected: "Rp 1.500.000",
		},
		{
			name:     "Negative small",
			amount:   -500,
			expected: "- Rp 500",
		},
		{
			name:     "Negative thousands",
			amount:   -1500,
			expected: "- Rp 1.500",
		},
		{
			name:     "Large amount",
			amount:   123456789,
			expected: "Rp 123.456.789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRupiah(tt.amount)
			if result != tt.expected {
				t.Errorf("formatRupiah(%d) = %q, want %q", tt.amount, result, tt.expected)
			}
		})
	}
}
