//go:build integration

package expense

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/ishanwardhono/expense-function/internal/platform/apierr"
	"github.com/ishanwardhono/expense-function/internal/platform/timeutil"
)

func connectTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	host := os.Getenv("DB_HOST")
	if host == "" {
		t.Skip("DB_HOST not set; skipping integration test")
	}
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslrootcert=%s sslmode=verify-full",
		host, os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSL_ROOT_CERT"),
	)
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// createTestSub inserts a bare subscription identity row (no version needed for
// FK references) and registers cleanup. Returns the new subscription ID.
func createTestSub(t *testing.T, db *sqlx.DB, name string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	err := db.GetContext(ctx, &id,
		`INSERT INTO amplop.subscription (name, color) VALUES ($1, '') RETURNING id`, name)
	if err != nil {
		t.Fatalf("createTestSub: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM amplop.subscription WHERE id = $1`, id) //nolint:errcheck
	})
	return id
}

// deleteExpenses removes a set of expense rows by ID. Called in t.Cleanup.
func deleteExpenses(ctx context.Context, db *sqlx.DB, ids ...uuid.UUID) {
	for _, id := range ids {
		db.ExecContext(ctx, `DELETE FROM amplop.expense WHERE id = $1`, id) //nolint:errcheck
	}
}

func TestForMonth_WideBoundaryWindow(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	// Jun 29, 2026 (Monday) — belongs to the week whose Friday is Jul 3, 2026.
	// It should appear in July's ForMonth window but not in May's.
	jun29 := Expense{
		OccurredDate: timeutil.Date(2026, 6, 29),
		Amount:       10_000,
		Category:     "Belanja",
		Note:         "boundary test jun29",
	}
	created1, err := repo.Create(ctx, jun29)
	if err != nil {
		t.Fatalf("Create jun29: %v", err)
	}
	t.Cleanup(func() { deleteExpenses(ctx, db, created1.ID) })

	// Jul 1, 2026 — falls inside June's +7d window (window extends to Jul 7).
	jul1 := Expense{
		OccurredDate: timeutil.Date(2026, 7, 1),
		Amount:       20_000,
		Category:     "Makan",
		Note:         "boundary test jul1",
	}
	created2, err := repo.Create(ctx, jul1)
	if err != nil {
		t.Fatalf("Create jul1: %v", err)
	}
	t.Cleanup(func() { deleteExpenses(ctx, db, created2.ID) })

	containsID := func(expenses []Expense, id uuid.UUID) bool {
		for _, e := range expenses {
			if e.ID == id {
				return true
			}
		}
		return false
	}

	// Jun 29 should appear in July's window [Jun 24 … Aug 7].
	julyExpenses, err := repo.ForMonth(ctx, 2026, 7)
	if err != nil {
		t.Fatalf("ForMonth July: %v", err)
	}
	if !containsID(julyExpenses, created1.ID) {
		t.Error("Jun 29 expense should be included in July's wide window")
	}

	// Jun 29 must NOT appear in May's window [Apr 24 … Jun 7].
	mayExpenses, err := repo.ForMonth(ctx, 2026, 5)
	if err != nil {
		t.Fatalf("ForMonth May: %v", err)
	}
	if containsID(mayExpenses, created1.ID) {
		t.Error("Jun 29 expense must not be included in May's window")
	}

	// Jul 1 should appear in June's window [May 25 … Jul 7].
	juneExpenses, err := repo.ForMonth(ctx, 2026, 6)
	if err != nil {
		t.Fatalf("ForMonth June: %v", err)
	}
	if !containsID(juneExpenses, created2.ID) {
		t.Error("Jul 1 expense should be included in June's wide window")
	}
}

func TestCreate_And_ByID(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	// Create with no time (nil OccurredTime).
	e := Expense{
		OccurredDate: timeutil.Date(2026, 6, 15),
		Amount:       45_000,
		Category:     "Makan",
		Note:         "nasi padang",
	}
	created, err := repo.Create(ctx, e)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { deleteExpenses(ctx, db, created.ID) })

	if created.ID == (uuid.UUID{}) {
		t.Error("ID should be non-zero after Create")
	}
	if created.OccurredYear != 2026 {
		t.Errorf("OccurredYear: got %d, want 2026", created.OccurredYear)
	}
	if created.OccurredMonth != 6 {
		t.Errorf("OccurredMonth: got %d, want 6", created.OccurredMonth)
	}
	at := created.OccurredAt()
	if at.Hour() != 0 || at.Minute() != 0 {
		t.Errorf("OccurredAt without time: expected midnight Jakarta, got %v", at)
	}
	if at.Location() != timeutil.Loc {
		t.Errorf("OccurredAt location: got %v, want %v", at.Location(), timeutil.Loc)
	}

	// ByID round-trip.
	fetched, err := repo.ByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if fetched.Amount != 45_000 {
		t.Errorf("Amount: got %d, want 45000", fetched.Amount)
	}
	if fetched.Note != "nasi padang" {
		t.Errorf("Note: got %q, want %q", fetched.Note, "nasi padang")
	}

	// Create with a non-nil time and verify OccurredAt preserves H:M:S.
	// lib/pq stores SQL TIME as clock-face hours in UTC on 0001-01-01.
	hm := time.Date(1, 1, 1, 14, 30, 0, 0, time.UTC)
	e2 := Expense{
		OccurredDate: timeutil.Date(2026, 6, 16),
		OccurredTime: &hm,
		Amount:       18_000,
		Category:     "Jajan",
		Note:         "es teh",
	}
	created2, err := repo.Create(ctx, e2)
	if err != nil {
		t.Fatalf("Create with time: %v", err)
	}
	t.Cleanup(func() { deleteExpenses(ctx, db, created2.ID) })

	at2 := created2.OccurredAt()
	if at2.Hour() != 14 || at2.Minute() != 30 {
		t.Errorf("OccurredAt with time: got %v, want 14:30 Jakarta", at2)
	}
}

func TestByID_NotFound(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	_, err := repo.ByID(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindNotFound {
		t.Errorf("expected NotFound, got %v", err)
	}
}

func TestCreate_LanggananUniqueConstraint(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	subID := createTestSub(t, db, "Test Sub Unique 2026")

	// First Langganan expense for the sub in June 2026.
	e1 := Expense{
		OccurredDate:   timeutil.Date(2026, 6, 5),
		Amount:         186_000,
		Category:       "Langganan",
		SubscriptionID: &subID,
		Note:           "first payment",
	}
	created1, err := repo.Create(ctx, e1)
	if err != nil {
		t.Fatalf("Create first Langganan: %v", err)
	}
	t.Cleanup(func() { deleteExpenses(ctx, db, created1.ID) })

	// Second Langganan for the same sub in the same month → DB backstop fires.
	e2 := e1
	e2.Note = "duplicate payment"
	_, err = repo.Create(ctx, e2)
	if err == nil {
		t.Fatal("expected Conflict, got nil")
	}
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindConflict {
		t.Errorf("expected Conflict, got %v", err)
	}

	// A Langganan expense in a different month succeeds.
	e3 := Expense{
		OccurredDate:   timeutil.Date(2026, 7, 5),
		Amount:         186_000,
		Category:       "Langganan",
		SubscriptionID: &subID,
		Note:           "july payment",
	}
	created3, err := repo.Create(ctx, e3)
	if err != nil {
		t.Fatalf("Create Langganan different month: %v", err)
	}
	t.Cleanup(func() { deleteExpenses(ctx, db, created3.ID) })
}

func TestExistsForSubscriptionMonth(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	subID := createTestSub(t, db, "Test Sub Exists 2026")

	e := Expense{
		OccurredDate:   timeutil.Date(2026, 6, 5),
		Amount:         186_000,
		Category:       "Langganan",
		SubscriptionID: &subID,
		Note:           "exists check",
	}
	created, err := repo.Create(ctx, e)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { deleteExpenses(ctx, db, created.ID) })

	// Exists without exclude → true.
	exists, err := repo.ExistsForSubscriptionMonth(ctx, subID, 2026, 6, nil)
	if err != nil {
		t.Fatalf("ExistsForSubscriptionMonth: %v", err)
	}
	if !exists {
		t.Error("expected exists=true, got false")
	}

	// Exists excluding the row itself → false (update-self case).
	exists, err = repo.ExistsForSubscriptionMonth(ctx, subID, 2026, 6, &created.ID)
	if err != nil {
		t.Fatalf("ExistsForSubscriptionMonth with exclude: %v", err)
	}
	if exists {
		t.Error("expected exists=false when excluding own row, got true")
	}

	// Different month → false.
	exists, err = repo.ExistsForSubscriptionMonth(ctx, subID, 2026, 7, nil)
	if err != nil {
		t.Fatalf("ExistsForSubscriptionMonth different month: %v", err)
	}
	if exists {
		t.Error("expected exists=false for different month, got true")
	}
}

func TestUpdate_And_Delete(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	e := Expense{
		OccurredDate: timeutil.Date(2026, 6, 10),
		Amount:       30_000,
		Category:     "Lainnya",
		Note:         "original",
	}
	created, err := repo.Create(ctx, e)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Update amount and note.
	created.Amount = 35_000
	created.Note = "updated"
	updated, err := repo.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Amount != 35_000 {
		t.Errorf("Amount after Update: got %d, want 35000", updated.Amount)
	}
	if updated.Note != "updated" {
		t.Errorf("Note after Update: got %q, want %q", updated.Note, "updated")
	}

	// Delete.
	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// ByID after delete → NotFound.
	_, err = repo.ByID(ctx, created.ID)
	if err == nil {
		t.Fatal("expected NotFound after Delete, got nil")
	}
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindNotFound {
		t.Errorf("expected NotFound, got %v", err)
	}

	// Delete non-existent → NotFound.
	err = repo.Delete(ctx, created.ID)
	if err == nil {
		t.Fatal("expected NotFound on second Delete, got nil")
	}
	if !errors.As(err, &ae) || ae.Kind != apierr.KindNotFound {
		t.Errorf("expected NotFound on second Delete, got %v", err)
	}
}

func TestUpdate_LanggananMove_Conflict(t *testing.T) {
	db := connectTestDB(t)
	repo := NewRepo(db)
	ctx := context.Background()

	subID := createTestSub(t, db, "Test Sub Move 2026")

	// Payment in June.
	e1 := Expense{
		OccurredDate:   timeutil.Date(2026, 6, 5),
		Amount:         186_000,
		Category:       "Langganan",
		SubscriptionID: &subID,
		Note:           "june payment",
	}
	c1, err := repo.Create(ctx, e1)
	if err != nil {
		t.Fatalf("Create June: %v", err)
	}
	t.Cleanup(func() { deleteExpenses(ctx, db, c1.ID) })

	// Payment in July.
	e2 := Expense{
		OccurredDate:   timeutil.Date(2026, 7, 5),
		Amount:         186_000,
		Category:       "Langganan",
		SubscriptionID: &subID,
		Note:           "july payment",
	}
	c2, err := repo.Create(ctx, e2)
	if err != nil {
		t.Fatalf("Create July: %v", err)
	}
	t.Cleanup(func() { deleteExpenses(ctx, db, c2.ID) })

	// Move the July payment into June → conflicts with the existing June payment.
	c2.OccurredDate = timeutil.Date(2026, 6, 20)
	_, err = repo.Update(ctx, c2)
	if err == nil {
		t.Fatal("expected Conflict when moving payment into occupied month, got nil")
	}
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Kind != apierr.KindConflict {
		t.Errorf("expected Conflict, got %v", err)
	}
}
