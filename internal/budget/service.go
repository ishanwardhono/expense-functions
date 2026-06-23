package budget

import (
	"context"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// repository is the subset of the budget repo the service depends on. The
// concrete *Repo satisfies it; tests use an in-memory fake.
type repository interface {
	Resolve(ctx context.Context, year, month int) (Config, error)
	Upsert(ctx context.Context, year, month int, monthly, shopWeekly, weekendBudget int64) error
}

// Service orchestrates budget config reads and effective-dated writes.
type Service struct {
	repo repository
	now  timeutil.Clock
}

// NewService builds a budget Service. now defaults to timeutil.Now when nil.
func NewService(repo repository, now timeutil.Clock) *Service {
	if now == nil {
		now = timeutil.Now
	}
	return &Service{repo: repo, now: now}
}

// UpdateRequest is the PUT /budget body (spec §7.4).
type UpdateRequest struct {
	Monthly       int64 `json:"monthly"`
	ShopWeekly    int64 `json:"shop_weekly"`
	WeekendBudget int64 `json:"weekend_budget"`
}

// Resolve returns the effective budget config for (year, month).
func (s *Service) Resolve(ctx context.Context, year, month int) (Config, error) {
	return s.repo.Resolve(ctx, year, month)
}

// Update validates req and upserts a budget config version effective from the
// current Asia/Jakarta month (spec §5.2). It returns the now-effective config.
func (s *Service) Update(ctx context.Context, req UpdateRequest) (Config, error) {
	if req.Monthly < 0 || req.ShopWeekly < 0 || req.WeekendBudget < 0 {
		return Config{}, apierr.Invalid("budget values must be >= 0")
	}
	year, month := timeutil.CurrentMonth(s.now())
	if err := s.repo.Upsert(ctx, year, month, req.Monthly, req.ShopWeekly, req.WeekendBudget); err != nil {
		return Config{}, err
	}
	return s.repo.Resolve(ctx, year, month)
}
