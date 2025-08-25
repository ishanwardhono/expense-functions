package weekly

import (
	"time"

	"github.com/google/uuid"
	"github.com/ishanwardhono/expense-function/common"
)

type WeekData struct {
	year int
	week int
	day  int
}

type Expenses []Expense

type Expense struct {
	Id          string    `db:"id"`
	Year        int       `db:"year"`
	Week        int       `db:"week"`
	Day         int       `db:"day"`
	Amount      int64     `db:"amount"`
	Type        string    `db:"type"`
	Note        string    `db:"note"`
	CreatedTime time.Time `db:"created_time"`
}

func (e Expenses) GetDayExpenses() (weekday, saturday, sunday int64) {
	for _, expense := range e {
		switch {
		case expense.Day < 5:
			weekday += expense.Amount
		case expense.Day == 5:
			saturday += expense.Amount
		case expense.Day == 6:
			sunday += expense.Amount
		}
	}
	return
}

func (e Expenses) ToDetailsResponse() (details []expenseDetail) {
	for _, expense := range e {
		details = append(details, expenseDetail{
			Day:    expense.Day,
			Amount: common.FormatRupiah(expense.Amount),
			Type:   expense.Type,
			Note:   expense.Note,
			Time:   expense.CreatedTime.Format("2006-01-02 15:04:05"),
		})
	}
	return
}

type expenseResponse struct {
	Year      int              `json:"year"`
	Week      int              `json:"week"`
	DayLabel  string           `json:"day_label"`
	DateRange string           `json:"date_range"`
	Remaining expenseRemaining `json:"remaining"`
}

type expenseRemaining struct {
	Weekday  dataLabel `json:"weekday"`
	Saturday dataLabel `json:"saturday"`
	Sunday   dataLabel `json:"sunday"`
	Days     struct {
		Senin  string `json:"Senin"`
		Selasa string `json:"Selasa"`
		Rabu   string `json:"Rabu"`
		Kamis  string `json:"Kamis"`
		Jumat  string `json:"Jumat"`
		Sabtu  string `json:"Sabtu"`
		Minggu string `json:"Minggu"`
	} `json:"days"`
	Details []expenseDetail `json:"details"`
}

type expenseDetail struct {
	Day    int    `json:"day"`
	Amount string `json:"amount"`
	Type   string `json:"type"`
	Note   string `json:"note"`
	Time   string `json:"time"`
}

type dataLabel struct {
	Label      string `json:"label"`
	LabelColor string `json:"label_color,omitempty"`
}

func toDataLabel(remaining int64, isDone bool) dataLabel {
	labelColor := ""
	if isDone {
		labelColor = "green"
	}
	if remaining < 0 {
		labelColor = "red"
	}
	return dataLabel{
		Label:      common.FormatRupiah(remaining),
		LabelColor: labelColor,
	}
}

type AddRequest struct {
	Amount int64   `json:"amount"`
	Date   *string `json:"date"`
	Type   string  `json:"type"`
	Note   string  `json:"note"`
}

func (r *AddRequest) ToExpense(weekData WeekData, t time.Time) Expense {
	return Expense{
		Id:          uuid.New().String(),
		Year:        weekData.year,
		Week:        weekData.week,
		Day:         weekData.day,
		Amount:      r.Amount,
		Type:        r.Type,
		Note:        r.Note,
		CreatedTime: t,
	}
}
