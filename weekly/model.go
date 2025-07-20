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
	Weekday dataLabel `json:"weekday"`
	Weekend dataLabel `json:"weekend"`
	Days    struct {
		Senin  string `json:"Senin"`
		Selasa string `json:"Selasa"`
		Rabu   string `json:"Rabu"`
		Kamis  string `json:"Kamis"`
		Jumat  string `json:"Jumat"`
		Sabtu  string `json:"Sabtu"`
		Minggu string `json:"Minggu"`
	} `json:"days"`
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
		Label:      formatRupiah(remaining),
		LabelColor: labelColor,
	}
}
