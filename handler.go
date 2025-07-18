package expensefunction

import (
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/ishanwardhono/expense-function/weekly"
)

type handlerFunc func(r *http.Request) (interface{}, error)

func init() {
	functions.HTTP("WeeklyGet", baseHandler(weeklyGet))
}

func baseHandler(hf handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := hf(r)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, resp)
	}
}

func weeklyGet(r *http.Request) (interface{}, error) {
	return weekly.Get(r.Context())
}
