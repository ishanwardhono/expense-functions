// Package expensefunction registers the single routed Cloud Function "Expense".
//
// All v2 endpoints are served by one HTTP function through an internal
// method+path router (spec §4.2). Domain routes are wired in Phase 3; for now
// the router only carries shared middleware (CORS, panic recovery, JSON, and
// the typed-error→status mapping) and a 404 fallback.
package expensefunction

import (
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"

	"github.com/ishanwardhono/expense-function/internal/platform/httpx"
)

func init() {
	r := newRouter()
	functions.HTTP("Expense", httpx.Middleware(r.ServeHTTP))
}

// newRouter builds the application router. Routes are added here as handlers
// land (Phase 3).
func newRouter() *httpx.Router {
	r := httpx.NewRouter()
	// TODO(Phase 3): r.Handle(http.MethodGet, "/month", monthHandler) etc.
	return r
}
