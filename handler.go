package expensefunction

import (
	"encoding/json"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/ishanwardhono/expense-function/weekly"
)

func init() {
	functions.HTTP("WeeklyGet", baseHandler(weeklyGet))
	functions.HTTP("WeeklyAdd", baseHandler(weeklyAdd))
}

func weeklyGet(r *http.Request) (interface{}, error) {
	return weekly.Get(r.Context())
}

func weeklyAdd(r *http.Request) (interface{}, error) {
	var req weekly.AddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return nil, weekly.Add(r.Context(), req)
}
