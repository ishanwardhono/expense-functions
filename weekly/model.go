package weekly

import "time"

type WeekData struct {
	year int
	week int
	day  int
}

type WeeklyExpense struct {
	id          int       `db:"id"`
	year        int       `db:"year"`
	week        int       `db:"week"`
	weekday     int64     `db:"weekday"`
	weekend     int64     `db:"weekend"`
	createdTime time.Time `db:"created_time"`
}

type expenseResponse struct {
	year     int              `json:"year"`
	week     int              `json:"week"`
	dayLabel string           `json:"day_label"`
	remaning expenseRemaining `json:"remaining"`
}

type expenseRemaining struct {
	weekday string   `json:"weekday"`
	weekend string   `json:"weekend"`
	days    []string `json:"days"`
}
