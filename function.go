// Package expensefunction registers the single routed Cloud Function "Expense".
//
// All v2 endpoints are served by one HTTP function through an internal
// method+path router (spec §4.2): shared middleware (CORS, panic recovery,
// JSON, typed-error→status mapping) plus the §7 domain routes.
package expensefunction

import (
	"net/http"
	"sync"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/jmoiron/sqlx"

	"github.com/ishanwardhono/expense-function/internal/budget"
	"github.com/ishanwardhono/expense-function/internal/expense"
	"github.com/ishanwardhono/expense-function/internal/month"
	"github.com/ishanwardhono/expense-function/internal/platform/config"
	"github.com/ishanwardhono/expense-function/internal/platform/database"
	"github.com/ishanwardhono/expense-function/internal/platform/httpx"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
	"github.com/ishanwardhono/expense-function/internal/subscription"
)

func init() {
	functions.HTTP("Expense", httpx.Middleware(newRouter().ServeHTTP))
}

// handlers bundles the wired HTTP handlers for the domain routes.
type handlers struct {
	month   *month.Handler
	expense *expense.Handler
	sub     *subscription.Handler
	budget  *budget.Handler
}

// dependency provider: the DB connection is opened lazily on the first request
// (the serverless cold-start pattern) so init() never fails on a bad
// environment; a connect error surfaces as a 500 on the request that needed it.
var (
	once     sync.Once
	cached   *handlers
	cacheErr error
)

func getHandlers() (*handlers, error) {
	once.Do(func() {
		db, err := database.Connect(config.LoadDatabase())
		if err != nil {
			cacheErr = err
			return
		}
		cached = buildHandlers(db, timeutil.Now)
	})
	return cached, cacheErr
}

// buildHandlers wires repos → services → handlers. Exposed-for-testing seam:
// callers can pass a test DB and a pinned clock.
func buildHandlers(db *sqlx.DB, now timeutil.Clock) *handlers {
	expenseRepo := expense.NewRepo(db)
	subRepo := subscription.NewRepo(db)
	budgetRepo := budget.NewRepo(db)

	expenseSvc := expense.NewService(expenseRepo, subRepo)
	subSvc := subscription.NewService(subRepo, now)
	budgetSvc := budget.NewService(budgetRepo, now)
	monthSvc := month.NewService(budgetRepo, subRepo, expenseRepo, now)

	return &handlers{
		month:   month.NewHandler(monthSvc, now),
		expense: expense.NewHandler(expenseSvc),
		sub:     subscription.NewHandler(subSvc, now),
		budget:  budget.NewHandler(budgetSvc, now),
	}
}

// newRouter builds the application router. Each route resolves the lazily-built
// handlers, so a DB error becomes a 500 rather than a startup crash.
func newRouter() *httpx.Router {
	r := httpx.NewRouter()

	r.Handle(http.MethodGet, "/month", lazy(func(h *handlers) httpx.HandlerFunc { return h.month.Get }))

	r.Handle(http.MethodPost, "/expenses", lazy(func(h *handlers) httpx.HandlerFunc { return h.expense.Create }))
	r.Handle(http.MethodPut, "/expenses/{id}", lazy(func(h *handlers) httpx.HandlerFunc { return h.expense.Update }))
	r.Handle(http.MethodDelete, "/expenses/{id}", lazy(func(h *handlers) httpx.HandlerFunc { return h.expense.Delete }))

	r.Handle(http.MethodGet, "/subscriptions", lazy(func(h *handlers) httpx.HandlerFunc { return h.sub.List }))
	r.Handle(http.MethodPost, "/subscriptions", lazy(func(h *handlers) httpx.HandlerFunc { return h.sub.Create }))
	r.Handle(http.MethodPut, "/subscriptions/{id}", lazy(func(h *handlers) httpx.HandlerFunc { return h.sub.Update }))
	r.Handle(http.MethodDelete, "/subscriptions/{id}", lazy(func(h *handlers) httpx.HandlerFunc { return h.sub.Delete }))

	r.Handle(http.MethodGet, "/budget", lazy(func(h *handlers) httpx.HandlerFunc { return h.budget.Get }))
	r.Handle(http.MethodPut, "/budget", lazy(func(h *handlers) httpx.HandlerFunc { return h.budget.Put }))

	return r
}

// lazy adapts a handler selector into a route handler that resolves the
// lazily-built dependencies on each request.
func lazy(pick func(*handlers) httpx.HandlerFunc) httpx.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		h, err := getHandlers()
		if err != nil {
			return err
		}
		return pick(h)(w, r)
	}
}
