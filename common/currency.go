package common

import "fmt"

func FormatRupiah(amount int64) string {
	isNegative := amount < 0
	if isNegative {
		amount = -amount
	}

	str := fmt.Sprintf("%d", amount)
	n := len(str)
	if n <= 3 {
		if isNegative {
			return "- Rp " + str
		}
		return "Rp " + str
	}

	result := ""
	for i, digit := range str {
		if i > 0 && (n-i)%3 == 0 {
			result += "."
		}
		result += string(digit)
	}

	if isNegative {
		return "- Rp " + result
	}
	return "Rp " + result
}
