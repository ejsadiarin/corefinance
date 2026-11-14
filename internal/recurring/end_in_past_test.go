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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/auth"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/mock"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

// Ending a rule in the past (#5): entries the rule created after the new end
// date are reported, and skipped unless the caller keeps them.

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func ptr[T any](v T) *T { return &v }

// rel is the server's date n days from today. Saving a rule fills it up to
// today (#14), so the #5 scenario (entries on the 3rd, 10th and 17th, end set
// to the 8th) is laid out relative to today: entries 14, 7 and 0 days ago,
// end moved to 10 days ago.
func rel(n int) string { return time.Now().AddDate(0, 0, n).Format(time.DateOnly) }

func entryDates(entries []RuleEntry) []string {
	out := []string{}
	for _, e := range entries {
		out = append(out, e.Date)
	}
	return out
}

func entryStatuses(entries []RuleEntry) []string {
	out := []string{}
	for _, e := range entries {
		out = append(out, e.Status)
	}
	return out
}

// weeklyExpenseRule seeds a weekly rule with entries on Aug 3, 10 and 17.
func weeklyExpenseRule(t *testing.T, ctx context.Context, pool *pgxpool.Pool, s *Service, user uuid.UUID) (db.RecurringExpenseRule, UpdateExpenseRuleRequest) {
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

	assert.Equal(t, []string{rel(-7), rel(0)}, entryDates(got.EntriesAfterEnd))
	assert.Equal(t, []string{"skipped", "skipped"}, entryStatuses(got.EntriesAfterEnd))
	for _, e := range got.EntriesAfterEnd {
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
	upd.EntriesAfterEnd = EntriesAfterEndKeep
	got, err := s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)

	assert.Equal(t, []string{rel(-7), rel(0)}, entryDates(got.EntriesAfterEnd))
	assert.Equal(t, []string{"posted", "posted"}, entryStatuses(got.EntriesAfterEnd))
	for _, e := range got.EntriesAfterEnd {
		assert.Equal(t, "posted", expenseStatus(t, ctx, q, user, e.ID))
	}

	// A later edit that leaves the end date where it is doesn't revisit the
	// entries the user chose to keep.
	kept := got.EntriesAfterEnd
	upd.EntriesAfterEnd = ""
	upd.Amount = 60
	got, err = s.UpdateExpenseRule(ctx, rule.ID, user, upd)
	require.NoError(t, err)
	assert.Empty(t, got.EntriesAfterEnd)
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
	assert.Empty(t, got.EntriesAfterEnd)
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
	assert.Equal(t, []string{rel(-7), rel(0)}, entryDates(got.EntriesAfterEnd))
	assert.Equal(t, []string{"skipped", "skipped"}, entryStatuses(got.EntriesAfterEnd))
}

// The handler refuses a choice other than skip or keep, before any write.
func TestUpdateRuleHandler_RejectsUnknownEntriesAfterEnd(t *testing.T) {
	h := NewHandler(NewService(&mock.MockQuerier{}))
	for path, update := range map[string]http.HandlerFunc{
		"/api/budget/recurring-expenses/": h.UpdateExpenseRule,
		"/api/budget/recurring-incomes/":  h.UpdateIncomeRule,
	} {
		id := uuid.New()
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", id.String())
		req := httptest.NewRequest(http.MethodPut, path+id.String(), strings.NewReader(`{"entries_after_end":"delete"}`))
		req = req.WithContext(context.WithValue(auth.WithUserID(context.Background(), uuid.New()), chi.RouteCtxKey, rctx))
		rec := httptest.NewRecorder()
		update(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code, path)
	}
}
