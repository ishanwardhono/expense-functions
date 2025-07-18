package expensefunction

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("HelloGet", HelloGet)
}

// HelloGet is an HTTP Cloud Function.
func HelloGet(w http.ResponseWriter, r *http.Request) {
	maxExpenseStr := os.Getenv("MAX_EXPENSE")
	maxExpense, err := strconv.ParseInt(maxExpenseStr, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid MAX_EXPENSE environment variable: %v", err), http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, "Weekly expense limit is: Rp", maxExpense)
}
