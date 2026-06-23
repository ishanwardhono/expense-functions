package expense

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ishanwardhono/expense-function/internal/envelope"
	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

// repository is the subset of the expense repo the service depends on. The
// concrete *Repo satisfies it; tests use an in-memory fake.
type repository interface {
	ByID(ctx context.Context, id uuid.UUID) (Expense, error)
	Create(ctx context.Context, e Expense) (Expense, error)
	Update(ctx context.Context, e Expense) (Expense, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsForSubscriptionMonth(ctx context.Context, subscriptionID uuid.UUID, year, month int, excludeID *uuid.UUID) (bool, error)
}

// subChecker verifies a referenced subscription exists (spec §7.5). The
// subscription repo's Exists method satisfies it.
type subChecker interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

// Service orchestrates expense CRUD with validation, the Langganan ⇔
// subscription_id rule, and the once-per-month payment rule (spec §7.2/§7.5).
type Service struct {
	repo repository
	subs subChecker
}

// NewService builds an expense Service.
func NewService(repo repository, subs subChecker) *Service {
	return &Service{repo: repo, subs: subs}
}

// WriteRequest is the POST/PUT /expenses body (spec §7.2).
type WriteRequest struct {
	Date           string  `json:"date"` // YYYY-MM-DD
	Time           *string `json:"time"` // HH:MM, optional
	Amount         int64   `json:"amount"`
	Category       string  `json:"category"`
	SubscriptionID *string `json:"subscription_id"` // required iff category == Langganan
	Note           string  `json:"note"`
}

var validCategories = map[envelope.Category]bool{
	envelope.CatMakan:     true,
	envelope.CatBelanja:   true,
	envelope.CatJajan:     true,
	envelope.CatCash:      true,
	envelope.CatLainnya:   true,
	envelope.CatLangganan: true,
}

// Create validates req and inserts a new expense.
func (s *Service) Create(ctx context.Context, req WriteRequest) (Expense, error) {
	e, err := s.validateAndBuild(ctx, req, nil)
	if err != nil {
		return Expense{}, err
	}
	return s.repo.Create(ctx, e)
}

// Update validates req and replaces the expense with id. The once-per-month
// check excludes the row itself so re-saving an unchanged Langganan payment is
// allowed (spec §7.2).
func (s *Service) Update(ctx context.Context, id uuid.UUID, req WriteRequest) (Expense, error) {
	if _, err := s.repo.ByID(ctx, id); err != nil {
		return Expense{}, err
	}
	e, err := s.validateAndBuild(ctx, req, &id)
	if err != nil {
		return Expense{}, err
	}
	e.ID = id
	return s.repo.Update(ctx, e)
}

// Delete removes the expense with id.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// validateAndBuild performs §7.5 validation and assembles the DB model.
// excludeID is the row being edited (nil on create), excluded from the
// once-per-month pre-check.
func (s *Service) validateAndBuild(ctx context.Context, req WriteRequest, excludeID *uuid.UUID) (Expense, error) {
	if req.Amount <= 0 {
		return Expense{}, apierr.Invalid("amount must be > 0")
	}
	cat := envelope.Category(req.Category)
	if !validCategories[cat] {
		return Expense{}, apierr.Invalid("invalid category %q", req.Category)
	}
	date, err := timeutil.ParseDate(req.Date)
	if err != nil {
		return Expense{}, apierr.Invalid("invalid date %q, expected YYYY-MM-DD", req.Date)
	}
	var occurredTime *time.Time
	if req.Time != nil && *req.Time != "" {
		occurredTime, err = parseTimeOfDay(*req.Time)
		if err != nil {
			return Expense{}, err
		}
	}

	var subID *uuid.UUID
	if cat == envelope.CatLangganan {
		if req.SubscriptionID == nil || *req.SubscriptionID == "" {
			return Expense{}, apierr.Invalid("subscription_id is required for Langganan expenses")
		}
		parsed, err := uuid.Parse(*req.SubscriptionID)
		if err != nil {
			return Expense{}, apierr.Invalid("invalid subscription_id %q", *req.SubscriptionID)
		}
		exists, err := s.subs.Exists(ctx, parsed)
		if err != nil {
			return Expense{}, err
		}
		if !exists {
			return Expense{}, apierr.Invalid("subscription %s not found", parsed)
		}
		// Once-per-month pre-check on the calendar month of the date.
		year, month := date.Year(), int(date.Month())
		dup, err := s.repo.ExistsForSubscriptionMonth(ctx, parsed, year, month, excludeID)
		if err != nil {
			return Expense{}, err
		}
		if dup {
			return Expense{}, apierr.Conflict("subscription already paid this month")
		}
		subID = &parsed
	} else if req.SubscriptionID != nil && *req.SubscriptionID != "" {
		return Expense{}, apierr.Invalid("subscription_id is only valid for Langganan expenses")
	}

	return Expense{
		OccurredDate:   date,
		OccurredTime:   occurredTime,
		Amount:         req.Amount,
		Category:       string(cat),
		SubscriptionID: subID,
		Note:           req.Note,
	}, nil
}

// parseTimeOfDay parses "HH:MM" into the clock-face-hours-in-UTC representation
// the DB stores as SQL TIME (matches Expense.OccurredAt — model.go).
func parseTimeOfDay(s string) (*time.Time, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return nil, apierr.Invalid("invalid time %q, expected HH:MM", s)
	}
	tt := time.Date(1, 1, 1, t.Hour(), t.Minute(), 0, 0, time.UTC)
	return &tt, nil
}
