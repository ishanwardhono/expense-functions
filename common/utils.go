package common

type DataLabel struct {
	Label      string `json:"label"`
	LabelColor string `json:"label_color,omitempty"`
}

func ToDataLabel(remaining int64, isDone bool) DataLabel {
	labelColor := ""
	if isDone && remaining > 0 {
		labelColor = "green"
	}
	if remaining < 0 {
		labelColor = "red"
	}
	return DataLabel{
		Label:      FormatRupiah(remaining),
		LabelColor: labelColor,
	}
}
