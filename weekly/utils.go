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

func getWeekData(t time.Time) WeekData {
	year, week := t.ISOWeek()
	day := (int(t.Weekday()) - 1 + 7) % 7
	return WeekData{year: year, week: week, day: day}
}

func getDateRange(year, week int) string {
	// Calculate the Monday of the given ISO week
	// ISO week 1 is the first week that has at least 4 days in the new year
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.UTC)

	// Find the Monday of week 1 of the given year
	// Calculate days to subtract to get to Monday
	daysFromMonday := int(jan4.Weekday()-time.Monday+7) % 7
	mondayWeek1 := jan4.AddDate(0, 0, -daysFromMonday)

	// Calculate the Monday of the target week
	monday := mondayWeek1.AddDate(0, 0, 7*(week-1))

	// Get Sunday (6 days after Monday)
	sunday := monday.AddDate(0, 0, 6)

	mondayDay := monday.Day()
	sundayDay := sunday.Day()
	mondayMonth := monday.Format("Jan")
	sundayMonth := sunday.Format("Jan")
	mondayYear := monday.Year()
	sundayYear := sunday.Year()

	// Same year and same month
	if mondayYear == sundayYear && mondayMonth == sundayMonth {
		return fmt.Sprintf("%d - %d %s %d", mondayDay, sundayDay, mondayMonth, mondayYear)
	}

	// Same year, different month
	if mondayYear == sundayYear {
		return fmt.Sprintf("%d %s - %d %s %d", mondayDay, mondayMonth, sundayDay, sundayMonth, sundayYear)
	}

	// Different year
	return fmt.Sprintf("%d %s %d - %d %s %d", mondayDay, mondayMonth, mondayYear, sundayDay, sundayMonth, sundayYear)
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
