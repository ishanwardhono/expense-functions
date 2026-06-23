package subscription

import (
	"context"

	"github.com/google/uuid"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// repository is the subset of the subscription repo the service depends on. The
// concrete *Repo satisfies it; tests use an in-memory fake.
type repository interface {
	ByID(ctx context.Context, id uuid.UUID) (Identity, error)
	Resolve(ctx context.Context, year, month int) ([]Resolved, error)
	LatestVersion(ctx context.Context, id uuid.UUID, year, month int) (Version, error)
	CreateWithVersion(ctx context.Context, name, color string, year, month int, alloc int64, dueDay int16) (Identity, error)
	UpdateIdentity(ctx context.Context, id uuid.UUID, name, color string) (Identity, error)
	UpsertVersion(ctx context.Context, id uuid.UUID, year, month int, alloc int64, dueDay int16, active bool) error
}

// Service orchestrates subscription definition CRUD with effective-dated writes.
type Service struct {
	repo repository
	now  timeutil.Clock
}

// NewService builds a subscription Service. now defaults to timeutil.Now when nil.
func NewService(repo repository, now timeutil.Clock) *Service {
	if now == nil {
		now = timeutil.Now
	}
	return &Service{repo: repo, now: now}
}

// CreateRequest is the POST /subscriptions body (spec §7.3).
type CreateRequest struct {
	Name   string `json:"name"`
	Color  string `json:"color"`
	Alloc  int64  `json:"alloc"`
	DueDay int    `json:"due_day"`
}

// UpdateRequest is the PUT /subscriptions/{id} body (spec §7.3). All fields are
// optional; name/color update the identity, alloc/due_day upsert a version.
type UpdateRequest struct {
	Name   *string `json:"name"`
	Color  *string `json:"color"`
	Alloc  *int64  `json:"alloc"`
	DueDay *int    `json:"due_day"`
}

// Resolve returns the active subscription set for (year, month).
func (s *Service) Resolve(ctx context.Context, year, month int) ([]Resolved, error) {
	return s.repo.Resolve(ctx, year, month)
}

// Create validates req and inserts an identity plus a version effective from the
// current Asia/Jakarta month.
func (s *Service) Create(ctx context.Context, req CreateRequest) (Resolved, error) {
	if req.Name == "" {
		return Resolved{}, apierr.Invalid("name is required")
	}
	if req.Alloc <= 0 {
		return Resolved{}, apierr.Invalid("alloc must be > 0")
	}
	if req.DueDay < 1 || req.DueDay > 31 {
		return Resolved{}, apierr.Invalid("due_day must be between 1 and 31")
	}
	year, month := timeutil.CurrentMonth(s.now())
	ident, err := s.repo.CreateWithVersion(ctx, req.Name, req.Color, year, month, req.Alloc, int16(req.DueDay))
	if err != nil {
		return Resolved{}, err
	}
	return Resolved{ID: ident.ID, Name: ident.Name, Color: ident.Color, Alloc: req.Alloc, DueDay: int16(req.DueDay)}, nil
}

// Update applies a partial change: name/color update the identity (all months),
// alloc/due_day upsert a version effective from the current month (spec §5.2).
// It returns the subscription as it stands for the current month.
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (Resolved, error) {
	ident, err := s.repo.ByID(ctx, id)
	if err != nil {
		return Resolved{}, err
	}
	year, month := timeutil.CurrentMonth(s.now())
	latest, err := s.repo.LatestVersion(ctx, id, year, month)
	if err != nil {
		return Resolved{}, err
	}

	name, color := ident.Name, ident.Color
	if req.Name != nil {
		if *req.Name == "" {
			return Resolved{}, apierr.Invalid("name is required")
		}
		name = *req.Name
	}
	if req.Color != nil {
		color = *req.Color
	}
	if req.Name != nil || req.Color != nil {
		if _, err := s.repo.UpdateIdentity(ctx, id, name, color); err != nil {
			return Resolved{}, err
		}
	}

	alloc, dueDay := latest.Alloc, int(latest.DueDay)
	if req.Alloc != nil {
		if *req.Alloc <= 0 {
			return Resolved{}, apierr.Invalid("alloc must be > 0")
		}
		alloc = *req.Alloc
	}
	if req.DueDay != nil {
		if *req.DueDay < 1 || *req.DueDay > 31 {
			return Resolved{}, apierr.Invalid("due_day must be between 1 and 31")
		}
		dueDay = *req.DueDay
	}
	if req.Alloc != nil || req.DueDay != nil {
		if err := s.repo.UpsertVersion(ctx, id, year, month, alloc, int16(dueDay), latest.Active); err != nil {
			return Resolved{}, err
		}
	}

	return Resolved{ID: id, Name: name, Color: color, Alloc: alloc, DueDay: int16(dueDay)}, nil
}

// Delete soft-ends a subscription: it upserts a version with active=false
// effective from the current month, carrying alloc/due_day forward so past
// months keep showing it (spec §5.2).
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.ByID(ctx, id); err != nil {
		return err
	}
	year, month := timeutil.CurrentMonth(s.now())
	latest, err := s.repo.LatestVersion(ctx, id, year, month)
	if err != nil {
		return err
	}
	return s.repo.UpsertVersion(ctx, id, year, month, latest.Alloc, latest.DueDay, false)
}
