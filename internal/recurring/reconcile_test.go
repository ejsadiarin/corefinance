package recurring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/auth"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/mock"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

// Reconciling a rule's entries after a schedule change (#15, ADR-0001):
// entries the new schedule no longer produces are reported, and skipped if
// the user never edited them, unless the caller keeps them. Ending a rule in
// the past (#5) is one case of it.

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func ptr[T any](v T) *T { return &v }

// rel is the date n days from today, in UTC: the zone of a request whose
// token carries no tz. Saving a rule fills it up to
// today (#14), so the #5 scenario (entries on the 3rd, 10th and 17th, end set
// to the 8th) is laid out relative to today: entries 14, 7 and 0 days ago,
// end moved to 10 days ago.
func rel(n int) string { return time.Now().UTC().AddDate(0, 0, n).Format(time.DateOnly) }

func entryDates(entries []StaleEntry) []string {
	out := []string{}
	for _, e := range entries {
		out = append(out, e.Date)
	}
	return out
}

func entryStatuses(entries []StaleEntry) []string {
	out := []string{}
	for _, e := range entries {
		out = append(out, e.Status)
	}
	return out
}

// weeklyExpenseRule seeds a weekly rule with entries on Aug 3, 10 and 17.
func weeklyExpenseRule(t *testing.T, ctx context.Context, pool *pgxpool.Pool, s *Service, user uuid.UUID) (db.Rule, UpdateExpenseRuleRequest) {
	t.Helper()
	req := CreateExpenseRuleRequest{
		Amount: 50, Currency: "PHP", Description: "groceries", RecurringType: "weekly",
		StartDate: rel(-14), Priority: "need",
	}
	rule, err := s.CreateExpenseRule(ctx, user, req)
	require.NoError(t, err)
	return rule, UpdateExpenseRuleRequest{
		Amount: req.Amount, Currency: req.Currency, Description: req.Description,
		RecurringType: req.RecurringType, StartDate: req.StartDate, Priority: req.Priority,
	}
}

func expenseStatus(t *testing.T, ctx context.Context, q *db.Queries, user uuid.UUID, id string) string {
	t.Helper()
	e, err := q.GetExpense(ctx, db.GetExpenseParams{ID: uuid.MustParse(id), UserID: user})
	require.NoError(t, err)
	return e.Status
}

func TestEndExpenseRuleInPast_SkipsLaterEntriesByDefault_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	s := NewServiceWithPool(q, pool)
	user := uuid.New()
	rule, upd := weeklyExpenseRule(t, ctx, pool, s, user)

	upd.EndDate = ptr(rel(-10))
	got, err := s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)

	assert.Equal(t, []string{rel(-7), rel(0)}, entryDates(got.StaleEntries))
	assert.Equal(t, []string{"skipped", "skipped"}, entryStatuses(got.StaleEntries))
	for _, e := range got.StaleEntries {
		assert.Equal(t, "skipped", expenseStatus(t, ctx, q, user, e.ID))
	}
}

func TestEndExpenseRuleInPast_KeepLeavesEntriesPosted_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	s := NewServiceWithPool(q, pool)
	user := uuid.New()
	rule, upd := weeklyExpenseRule(t, ctx, pool, s, user)

	upd.EndDate = ptr(rel(-10))
	upd.StaleEntries = StaleKeep
	got, err := s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)

	assert.Equal(t, []string{rel(-7), rel(0)}, entryDates(got.StaleEntries))
	assert.Equal(t, []string{"posted", "posted"}, entryStatuses(got.StaleEntries))
	for _, e := range got.StaleEntries {
		assert.Equal(t, "posted", expenseStatus(t, ctx, q, user, e.ID))
	}

	// A later edit that leaves the end date where it is doesn't revisit the
	// entries the user chose to keep.
	kept := got.StaleEntries
	upd.StaleEntries = ""
	upd.Amount = 60
	got, err = s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)
	assert.Empty(t, got.StaleEntries)
	for _, e := range kept {
		assert.Equal(t, "posted", expenseStatus(t, ctx, q, user, e.ID))
	}
}

func TestEndExpenseRuleInFuture_ReportsNothing_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	s := NewServiceWithPool(db.New(pool), pool)
	user := uuid.New()
	rule, upd := weeklyExpenseRule(t, ctx, pool, s, user)

	upd.EndDate = ptr("2026-12-31")
	got, err := s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)
	assert.Empty(t, got.StaleEntries)
}

func TestEndIncomeRuleInPast_SkipsLaterEntriesByDefault_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	s := NewServiceWithPool(q, pool)
	user := uuid.New()
	req := CreateIncomeRuleRequest{
		Amount: 500, Currency: "PHP", Description: "allowance", RecurringType: "weekly", StartDate: rel(-14),
	}
	rule, err := s.CreateIncomeRule(ctx, user, req)
	require.NoError(t, err)

	got, err := s.UpdateIncomeRule(ctx, rule.ID, user, UpdateIncomeRuleRequest{
		Amount: req.Amount, Currency: req.Currency, Description: req.Description,
		RecurringType: req.RecurringType, StartDate: req.StartDate, EndDate: ptr(rel(-10)),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{rel(-7), rel(0)}, entryDates(got.StaleEntries))
	assert.Equal(t, []string{"skipped", "skipped"}, entryStatuses(got.StaleEntries))
}

// The handler refuses a choice other than skip or keep, before any write.
func TestUpdateRuleHandler_RejectsUnknownStaleEntries(t *testing.T) {
	h := NewHandler(NewService(&mock.MockQuerier{}))
	for path, update := range map[string]http.HandlerFunc{
		"/api/budget/recurring-expenses/": h.UpdateExpenseRule,
		"/api/budget/recurring-incomes/":  h.UpdateIncomeRule,
	} {
		id := uuid.New()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id.String())
		req := httptest.NewRequest(http.MethodPut, path+id.String(), strings.NewReader(`{"stale_entries":"delete"}`))
		req = req.WithContext(context.WithValue(auth.WithUserID(context.Background(), uuid.New()), chi.RouteCtxKey, rctx))
		rec := httptest.NewRecorder()
		update(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code, path)
	}
}

func expenseRuleWith(t *testing.T, ctx context.Context, s *Service, user uuid.UUID, cadence, start string) (db.Rule, UpdateExpenseRuleRequest) {
	t.Helper()
	req := CreateExpenseRuleRequest{
		Amount: 50, Currency: "PHP", Description: "groceries", RecurringType: cadence, StartDate: start, Priority: "need",
	}
	rule, err := s.CreateExpenseRule(ctx, user, req)
	require.NoError(t, err)
	return rule, UpdateExpenseRuleRequest{
		Amount: req.Amount, Currency: req.Currency, Description: req.Description,
		RecurringType: req.RecurringType, StartDate: req.StartDate, Priority: req.Priority,
	}
}

func newReconcileService(t *testing.T) (*Service, *db.Queries) {
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	return NewServiceWithPool(q, pool), q
}

// Weekly to monthly: the weekly entries the monthly schedule doesn't produce
// are skipped and reported; the ones it does produce stay.
func TestReconcile_CadenceChangeSkipsStaleEntries_Integration(t *testing.T) {
	ctx := context.Background()
	s, q := newReconcileService(t)
	user := uuid.New()
	rule, upd := expenseRuleWith(t, ctx, s, user, "weekly", rel(-14))

	upd.RecurringType = "monthly"
	got, err := s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)
	assert.Equal(t, []string{rel(-7), rel(0)}, entryDates(got.StaleEntries))
	assert.Equal(t, []string{"skipped", "skipped"}, entryStatuses(got.StaleEntries))
	for _, e := range got.StaleEntries {
		assert.False(t, e.NeedsReview)
		assert.Equal(t, "skipped", expenseStatus(t, ctx, q, user, e.ID))
	}
}

func TestReconcile_LaterStartSkipsEntriesBeforeIt_Integration(t *testing.T) {
	ctx := context.Background()
	s, _ := newReconcileService(t)
	user := uuid.New()
	rule, upd := expenseRuleWith(t, ctx, s, user, "daily", rel(-4))

	upd.StartDate = rel(-1)
	got, err := s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)
	assert.Equal(t, []string{rel(-4), rel(-3), rel(-2)}, entryDates(got.StaleEntries))
	assert.Equal(t, []string{"skipped", "skipped", "skipped"}, entryStatuses(got.StaleEntries))
}

// An entry the user edited is never skipped by a schedule change: it is kept
// and reported for review.
func TestReconcile_EditedEntryIsReportedNotSkipped_Integration(t *testing.T) {
	ctx := context.Background()
	s, q := newReconcileService(t)
	user := uuid.New()
	rule, upd := expenseRuleWith(t, ctx, s, user, "weekly", rel(-14))

	edited := entryOn(t, ctx, q, user, rule.ID, rel(-7))
	_, err := q.UpdateExpense(ctx, db.UpdateExpenseParams{
		ID: edited.ID, UserID: user, Amount: edited.Amount, Currency: edited.Currency,
		ExchangeRate: edited.ExchangeRate, AmountPhp: edited.AmountPhp, RateDate: edited.RateDate,
		Description: "groceries (big week)", ExpenseDate: edited.ExpenseDate,
		Priority: edited.Priority, Status: edited.Status, SourceRuleID: edited.SourceRuleID,
	})
	require.NoError(t, err)

	upd.RecurringType = "monthly"
	got, err := s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)
	require.Len(t, got.StaleEntries, 2)
	byDate := map[string]StaleEntry{}
	for _, e := range got.StaleEntries {
		byDate[e.Date] = e
	}
	assert.True(t, byDate[rel(-7)].NeedsReview)
	assert.Equal(t, "posted", byDate[rel(-7)].Status)
	assert.Equal(t, "posted", expenseStatus(t, ctx, q, user, edited.ID.String()))
	assert.False(t, byDate[rel(0)].NeedsReview)
	assert.Equal(t, "skipped", byDate[rel(0)].Status)
}

func TestReconcile_RepeatingTheUpdateDoesNothing_Integration(t *testing.T) {
	ctx := context.Background()
	s, _ := newReconcileService(t)
	user := uuid.New()
	rule, upd := expenseRuleWith(t, ctx, s, user, "weekly", rel(-14))

	upd.RecurringType = "monthly"
	_, err := s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)
	got, err := s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)
	assert.Empty(t, got.StaleEntries)
}

// End, clear, end again: entries kept the first time are the user's decision,
// so the second end reports them for review instead of skipping them.
func TestReconcile_KeptEntriesSurviveEndingAgain_Integration(t *testing.T) {
	ctx := context.Background()
	s, q := newReconcileService(t)
	user := uuid.New()
	rule, upd := expenseRuleWith(t, ctx, s, user, "weekly", rel(-14))

	upd.EndDate = ptr(rel(-10))
	upd.StaleEntries = StaleKeep
	_, err := s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)

	upd.EndDate, upd.StaleEntries = nil, ""
	_, err = s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)

	upd.EndDate = ptr(rel(-10))
	got, err := s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)
	require.Len(t, got.StaleEntries, 2)
	for _, e := range got.StaleEntries {
		assert.True(t, e.NeedsReview, e.Date)
		assert.Equal(t, "posted", expenseStatus(t, ctx, q, user, e.ID))
	}
}

func entryOn(t *testing.T, ctx context.Context, q *db.Queries, user, rule uuid.UUID, date string) db.GetExpenseRow {
	t.Helper()
	rows, err := q.ListRuleEntries(ctx, db.ListRuleEntriesParams{
		UserID: user, RuleID: pgtype.UUID{Bytes: rule, Valid: true},
		StartDate: pgtype.Date{Time: day(date), Valid: true}, EndDate: pgtype.Date{Time: day(date), Valid: true}, PageLimit: 10,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	e, err := q.GetExpense(ctx, db.GetExpenseParams{ID: rows[0].ID, UserID: user})
	require.NoError(t, err)
	return e
}

// A schedule change adds no entries before today (#15): reopening a daily
// rule that ended 5 days ago adds today's entry, not the days in between.
func TestReconcile_ReopeningEndedRuleAddsOnlyToday_Integration(t *testing.T) {
	ctx := context.Background()
	s, q := newReconcileService(t)
	user := uuid.New()
	req := CreateExpenseRuleRequest{
		Amount: 5, Currency: "PHP", Description: "bus", RecurringType: "daily",
		StartDate: rel(-7), EndDate: ptr(rel(-5)), Priority: "need",
	}
	rule, err := s.CreateExpenseRule(ctx, user, req)
	require.NoError(t, err)
	require.Equal(t, []string{rel(-7), rel(-6), rel(-5)}, ruleEntryDates(t, ctx, q, user, rule.ID))

	_, err = s.UpdateExpenseRule(ctx, rule.ID, user, UpdateExpenseRuleRequest{
		Amount: req.Amount, Currency: req.Currency, Description: req.Description,
		RecurringType: req.RecurringType, StartDate: req.StartDate, Priority: req.Priority,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{rel(-7), rel(-6), rel(-5), rel(0)}, ruleEntryDates(t, ctx, q, user, rule.ID))
}
