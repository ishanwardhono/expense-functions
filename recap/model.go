package recap

import (
	"github.com/ishanwardhono/expense-function/common"
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
		Amount:      common.FormatRupiah(0),
		Remaining:   common.DataLabel{Label: common.FormatRupiah(weekly.MaxExpense * 2)},
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
