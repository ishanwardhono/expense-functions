package weekly

import (
	"fmt"
	"time"
)

const GaAdaJajanLabel = "Ga ada jajan"

var mapDayLabel = map[int]string{
	0: "Senin",
	1: "Selasa",
	2: "Rabu",
	3: "Kamis",
	4: "Jumat",
	5: "Sabtu",
	6: "Minggu",
}

func now() time.Time {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return time.Now().In(loc)
}

func formatRupiah(amount int64) string {
	isNegative := amount < 0
	if isNegative {
		amount = -amount
	}

	str := fmt.Sprintf("%d", amount)
	n := len(str)
	if n <= 3 {
		if isNegative {
			return "-Rp " + str
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
