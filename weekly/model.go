package weekly

import "time"

type WeekData struct {
	year int
	week int
	day  int
}

type WeeklyExpense struct {
	Id          string    `firestore:"-"`
	Year        int       `firestore:"year"`
	Week        int       `firestore:"week"`
	Weekday     int64     `firestore:"weekday"`
	Weekend     int64     `firestore:"weekend"`
	CreatedTime time.Time `firestore:"created_time"`
}

type expenseResponse struct {
	Year      int              `json:"year"`
	Week      int              `json:"week"`
	DayLabel  string           `json:"day_label"`
	Remaining expenseRemaining `json:"remaining"`
}

type expenseRemaining struct {
	Weekday string   `json:"weekday"`
	Weekend string   `json:"weekend"`
	Days    []string `json:"days"`
}
