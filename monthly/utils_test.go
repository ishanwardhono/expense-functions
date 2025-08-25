package monthly

import (
	"testing"
)

func TestGetMonthName(t *testing.T) {
	tests := []struct {
		month    int
		expected string
	}{
		{1, "January"},
		{2, "February"},
		{3, "March"},
		{4, "April"},
		{5, "May"},
		{6, "June"},
		{7, "July"},
		{8, "August"},
		{9, "September"},
		{10, "October"},
		{11, "November"},
		{12, "December"},
		{13, "Unknown"},
		{0, "Unknown"},
	}

	for _, test := range tests {
		result := getMonthName(test.month)
		if result != test.expected {
			t.Errorf("For month %d, expected %s, got %s", test.month, test.expected, result)
		}
	}
}

func TestMonthlyExpenses_GetTotalExpense(t *testing.T) {
	expenses := MonthlyExpenses{
		{Amount: 50000},
		{Amount: 25000},
		{Amount: 15000},
	}

	total := expenses.GetTotalExpense()
	expected := int64(90000)

	if total != expected {
		t.Errorf("Expected total %d, got %d", expected, total)
	}
}

func TestToDataLabel(t *testing.T) {
	tests := []struct {
		remaining     int64
		expectedColor string
	}{
		{50000, "green"},
		{0, ""},
		{-10000, "red"},
	}

	for _, test := range tests {
		label := toDataLabel(test.remaining)
		if label.LabelColor != test.expectedColor {
			t.Errorf("For remaining %d, expected color %s, got %s", test.remaining, test.expectedColor, label.LabelColor)
		}
	}
}
