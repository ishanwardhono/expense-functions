package monthly

import (
	"time"

	"github.com/google/uuid"
	"github.com/ishanwardhono/expense-function/common"
)

type MonthlyExpenses []MonthlyExpense

type MonthlyExpense struct {
	Id          string    `db:"id"`
	Year        int       `db:"year"`
	Month       int       `db:"month"`
	Amount      int64     `db:"amount"`
	Type        string    `db:"type"`
	Note        string    `db:"note"`
	CreatedTime time.Time `db:"created_time"`
}

func (e MonthlyExpenses) GetTotalExpense() int64 {
	var total int64
	for _, expense := range e {
		total += expense.Amount
	}
	return total
}

func (e MonthlyExpenses) ToDetailsResponse() (details []expenseDetail) {
	for _, expense := range e {
		details = append(details, expenseDetail{
			Amount: common.FormatRupiah(expense.Amount),
			Type:   expense.Type,
			Note:   expense.Note,
			Time:   expense.CreatedTime.Format("2006-01-02 15:04:05"),
		})
	}
	return
}

type expenseResponse struct {
	Year       int              `json:"year"`
	Month      int              `json:"month"`
	MonthLabel string           `json:"month_label"`
	DateRange  string           `json:"date_range"`
	TotalWeeks int              `json:"total_weeks"`
	Budget     string           `json:"budget"`
	Remaining  expenseRemaining `json:"remaining"`
}

type expenseRemaining struct {
	Total   common.DataLabel `json:"total"`
	Details []expenseDetail  `json:"details"`
}

type expenseDetail struct {
	Amount string `json:"amount"`
	Type   string `json:"type"`
	Note   string `json:"note"`
	Time   string `json:"time"`
}

type AddRequest struct {
	Amount int64   `json:"amount"`
	Date   *string `json:"date"`
	Type   string  `json:"type"`
	Note   string  `json:"note"`
}

func (r *AddRequest) ToMonthlyExpense(monthData monthData, t time.Time) MonthlyExpense {
	return MonthlyExpense{
		Id:          uuid.New().String(),
		Year:        monthData.year,
		Month:       monthData.month,
		Amount:      r.Amount,
		Type:        r.Type,
		Note:        r.Note,
		CreatedTime: t,
	}
}

type RecapResp struct {
	Year       int
	Month      int
	MonthLabel string
	Amount     int64
	Remaining  int64
	TotalWeeks int
	StartYear  int
	EndYear    int
	StartWeek  int
	EndWeek    int
}
