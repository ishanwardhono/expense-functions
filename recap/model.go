package recap

import (
	"fmt"

	"github.com/ishanwardhono/expense-function/common"
	"github.com/ishanwardhono/expense-function/monthly"
	"github.com/ishanwardhono/expense-function/weekly"
)

const (
	MinYear  = 2025
	MinMonth = 8
)

type GetRequest struct {
	Month int `json:"month"`
	Year  int `json:"year"`
}

type RecapData struct {
	Description string           `json:"description"`
	Amount      string           `json:"amount"`
	Remaining   common.DataLabel `json:"remaining"`
}

func EmptyRecapData(desc string) RecapData {
	return RecapData{
		Description: desc,
		Amount:      "-",
		Remaining:   common.DataLabel{Label: "-"},
	}
}

type RecapResponse struct {
	DateLabel string           `json:"date_label"`
	Expense   string           `json:"expense"`
	Remaining common.DataLabel `json:"remaining"`
	Details   []RecapData      `json:"details"`
	PrevMonth *PrevMonth       `json:"prev_month,omitempty"`
}

type PrevMonth struct {
	Month int
	Year  int
}

func newPrevMonth(month, year int) *PrevMonth {
	month--
	if month == 0 {
		month = 12
		year--
	}

	if year < MinYear || (year == MinYear && month < MinMonth) {
		return nil
	}

	return &PrevMonth{
		Month: month,
		Year:  year,
	}
}

func toRecapResponse(monthRecap monthly.RecapResp, weekRecap []weekly.RecapResp) RecapResponse {
	expense := int64(0)
	remaining := int64(0)
	recapData := make([]RecapData, 0)

	for i := 0; i < monthRecap.TotalWeeks; i++ {
		if i >= len(weekRecap) {
			recapData = append(recapData, EmptyRecapData(fmt.Sprintf("Minggu ke-%d", i+1)))
			continue
		}

		expense += weekRecap[i].Amount
		remaining += weekRecap[i].Remaining

		recapData = append(recapData, RecapData{
			Description: fmt.Sprintf("Minggu ke-%d", i+1),
			Amount:      common.FormatRupiah(weekRecap[i].Amount),
			Remaining:   common.ToDataLabel(weekRecap[i].Remaining, true),
		})
	}

	expense += monthRecap.Amount
	remaining += monthRecap.Remaining
	recapData = append(recapData, RecapData{
		Description: "Bulanan",
		Amount:      common.FormatRupiah(monthRecap.Amount),
		Remaining:   common.ToDataLabel(monthRecap.Remaining, true),
	})

	return RecapResponse{
		DateLabel: fmt.Sprintf("%s %d", monthRecap.MonthLabel, monthRecap.Year),
		Expense:   common.FormatRupiah(expense),
		Remaining: common.ToDataLabel(remaining, true),
		Details:   recapData,
		PrevMonth: newPrevMonth(monthRecap.Month, monthRecap.Year),
	}
}
