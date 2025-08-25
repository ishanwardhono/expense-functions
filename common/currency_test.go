package common

import "testing"

func TestFormatRupiah(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		expected string
	}{
		{"Zero", 0, "Rp 0"},
		{"Small amount", 500, "Rp 500"},
		{"Thousands", 1000, "Rp 1.000"},
		{"Ten thousands", 15000, "Rp 15.000"},
		{"Hundreds of thousands", 250000, "Rp 250.000"},
		{"Millions", 1500000, "Rp 1.500.000"},
		{"Negative small", -500, "- Rp 500"},
		{"Negative thousands", -15000, "- Rp 15.000"},
		{"Large amount", 1234567890, "Rp 1.234.567.890"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := FormatRupiah(test.amount)
			if result != test.expected {
				t.Errorf("FormatRupiah(%d) = %s; want %s", test.amount, result, test.expected)
			}
		})
	}
}
