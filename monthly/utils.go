package monthly

import (
	"fmt"
	"time"

	"github.com/ishanwardhono/expense-function/common"
)

const (
	MaxExpense = 9000000
)

var monthNames = map[int]string{
	1:  "January",
	2:  "February",
	3:  "March",
	4:  "April",
	5:  "May",
	6:  "June",
	7:  "July",
	8:  "August",
	9:  "September",
	10: "October",
	11: "November",
	12: "December",
}

func getMonthName(month int) string {
	if name, exists := monthNames[month]; exists {
		return name
	}
	return "Unknown"
}

// Legacy functions for payroll period calculation (keeping for compatibility)
type monthData struct {
	year      int
	month     int
	startYear int
	endYear   int
	startWeek int
	endWeek   int
}

// getTotalWeeks calculates the total number of weeks in the pay period
func (m monthData) getTotalWeeks() int {
	// If within the same year
	if m.startYear == m.endYear {
		return m.endWeek - m.startWeek + 1
	}

	// If spanning across years, calculate weeks from start year + weeks from end year
	weeksInStartYear := 52 - m.startWeek + 1 // Remaining weeks in start year
	if isLeapYear(m.startYear) && m.startWeek <= 53 {
		weeksInStartYear = 53 - m.startWeek + 1
	}

	return weeksInStartYear + m.endWeek
}

func (m monthData) getBudget(totalWeeks int, maxExpense int64) int64 {
	weeklyBudget := int64(totalWeeks) * 1000000
	return maxExpense - weeklyBudget
}

// isLeapYear checks if a year is a leap year (may have 53 weeks)
func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func getPayPeriodMonth(t time.Time) monthData {
	year, calendarMonth, day := t.Year(), int(t.Month()), t.Day()
	if day >= 25 {
		// Day 25 onwards belongs to next month's pay period
		nextMonth := calendarMonth + 1
		nextYear := year
		if nextMonth > 12 {
			nextMonth = 1 // Wrap to January
			nextYear++
		}

		periodWeek := monthData{month: nextMonth, year: nextYear}
		periodWeek.startYear, periodWeek.startWeek = getFirstMonday(time.Date(year, time.Month(calendarMonth), 25, 0, 0, 0, 0, common.Loc)).ISOWeek()
		periodWeek.endYear, periodWeek.endWeek = getLastMondayBefore(time.Date(nextYear, time.Month(nextMonth), 24, 0, 0, 0, 0, common.Loc)).ISOWeek()
		return periodWeek
	}

	// Before day 25 belongs to current month's pay period
	prevMonth := calendarMonth - 1
	prevYear := year
	if prevMonth < 1 {
		prevMonth = 12 // Wrap to December
		prevYear--
	}

	periodWeek := monthData{month: calendarMonth, year: year}
	periodWeek.startYear, periodWeek.startWeek = getFirstMonday(time.Date(prevYear, time.Month(prevMonth), 25, 0, 0, 0, 0, common.Loc)).ISOWeek()
	periodWeek.endYear, periodWeek.endWeek = getLastMondayBefore(time.Date(year, time.Month(calendarMonth), 24, 0, 0, 0, 0, common.Loc)).ISOWeek()
	return periodWeek
}

// getFirstMonday returns the first Monday on or after the given date
func getFirstMonday(date time.Time) time.Time {
	weekday := date.Weekday()
	if weekday == time.Monday {
		return date
	}

	// Calculate days to add to get to next Monday
	daysToAdd := (7 - int(weekday) + int(time.Monday)) % 7
	if daysToAdd == 0 {
		daysToAdd = 7
	}

	return date.AddDate(0, 0, daysToAdd)
}

// getLastMondayBefore returns the last Monday before the given date
func getLastMondayBefore(date time.Time) time.Time {
	weekday := date.Weekday()

	// Calculate days to subtract to get to previous Monday
	daysToSubtract := int(weekday) - int(time.Monday)
	if daysToSubtract < 0 {
		daysToSubtract += 7
	}

	return date.AddDate(0, 0, -daysToSubtract)
}

func getDateRange(year, month int) string {
	// Calculate start date (25th of previous month)
	var startMonth, startYear int
	if month == 1 {
		startMonth = 12
		startYear = year - 1
	} else {
		startMonth = month - 1
		startYear = year
	}
	startDate := time.Date(startYear, time.Month(startMonth), 25, 0, 0, 0, 0, time.UTC)

	// Calculate end date (24th of current month)
	endDate := time.Date(year, time.Month(month), 24, 0, 0, 0, 0, time.UTC)

	startDay := startDate.Day()
	endDay := endDate.Day()
	startMonthName := startDate.Format("Jan")
	endMonthName := endDate.Format("Jan")
	startYearVal := startDate.Year()
	endYearVal := endDate.Year()

	// Same year and same month (shouldn't happen with our logic, but for safety)
	if startYearVal == endYearVal && startMonthName == endMonthName {
		return fmt.Sprintf("%d - %d %s %d", startDay, endDay, startMonthName, startYearVal)
	}

	// Same year, different month
	if startYearVal == endYearVal {
		return fmt.Sprintf("%d %s - %d %s %d", startDay, startMonthName, endDay, endMonthName, endYearVal)
	}

	// Different year
	return fmt.Sprintf("%d %s %d - %d %s %d", startDay, startMonthName, startYearVal, endDay, endMonthName, endYearVal)
}
