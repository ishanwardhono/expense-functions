package expensefunction

import (
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/ishanwardhono/expense-function/weekly"
)

func init() {
	functions.HTTP("WeeklyGet", baseHandler(weeklyGet))
}

func weeklyGet(r *http.Request) (interface{}, error) {
	return weekly.Get(r.Context())
}
