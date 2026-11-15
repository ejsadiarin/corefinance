package ruleentry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/auth"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/materialize"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

// Rule-created entries for review (#8): what the worker created, filterable
// by rule, by date and by when it was inserted, only ever the caller's own.

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func pgDay(s string) pgtype.Date { return pgtype.Date{Time: day(s), Valid: true} }

type fixture struct {
	pool *pgxpool.Pool
	q    *db.Queries
	h    *Handler
}

func newFixture(t *testing.T) fixture {
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	return fixture{pool: pool, q: q, h: NewHandler(q)}
}

func (f fixture) expenseRule(t *testing.T, user uuid.UUID, cadence, start string) uuid.UUID {
	t.Helper()
	r, err := f.q.CreateRecurringExpenseRule(context.Background(), db.CreateRecurringExpenseRuleParams{
		UserID: user, Description: "rent", Amount: decimal.NewFromInt(100), Currency: "PHP",
		RecurringType: cadence, StartDate: pgDay(start), Priority: "need",
	})
	require.NoError(t, err)
	return r.ID
}

func (f fixture) incomeRule(t *testing.T, user uuid.UUID, cadence, start string) uuid.UUID {
	t.Helper()
	r, err := f.q.CreateRecurringIncomeRule(context.Background(), db.CreateRecurringIncomeRuleParams{
		UserID: user, Description: "salary", Amount: decimal.NewFromInt(500), Currency: "PHP",
		RecurringType: cadence, StartDate: pgDay(start),
	})
	require.NoError(t, err)
	return r.ID
}

// pass runs the worker as of the given date.
func (f fixture) pass(t *testing.T, today string) {
	t.Helper()
	_, err := materialize.MaterializeDue(context.Background(), f.pool, f.q, materialize.FixedZones{}, day(today))
	require.NoError(t, err)
}

func (f fixture) list(t *testing.T, user uuid.UUID, params url.Values) []Entry {
	t.Helper()
	rec := f.get(user, params)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var out []Entry
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

func (f fixture) get(user uuid.UUID, params url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/budget/rule-entries?"+params.Encode(), nil)
	req = req.WithContext(auth.WithUserID(context.Background(), user))
	rec := httptest.NewRecorder()
	f.h.List(rec, req)
	return rec
}

// keys renders entries as "kind date" in date order, for comparison.
func keys(entries []Entry) []string {
	out := []string{}
	for _, e := range entries {
		out = append(out, string(e.Kind)+" "+e.Date)
	}
	sort.Strings(out)
	return out
}

func TestRuleEntries_FilterByRule_Integration(t *testing.T) {
	f := newFixture(t)
	user := uuid.New()
	rent := f.expenseRule(t, user, "monthly", "2026-09-01")
	f.expenseRule(t, user, "monthly", "2026-09-15")
	salary := f.incomeRule(t, user, "monthly", "2026-09-05")
	f.pass(t, "2026-10-20")

	got := f.list(t, user, url.Values{"rule_id": {rent.String()}})
	assert.Equal(t, []string{"expense 2026-09-01", "expense 2026-10-01"}, keys(got))
	for _, e := range got {
		assert.Equal(t, rent.String(), e.RuleID)
	}

	got = f.list(t, user, url.Values{"rule_id": {salary.String()}})
	assert.Equal(t, []string{"income 2026-09-05", "income 2026-10-05"}, keys(got))

	got = f.list(t, user, nil)
	assert.Len(t, got, 6, "no filter: every rule's entries, expenses and incomes")
}

func TestRuleEntries_DateRange_Integration(t *testing.T) {
	f := newFixture(t)
	user := uuid.New()
	f.expenseRule(t, user, "weekly", "2026-09-01")
	f.incomeRule(t, user, "weekly", "2026-09-03")
	f.pass(t, "2026-09-30")

	got := f.list(t, user, url.Values{"start_date": {"2026-09-08"}, "end_date": {"2026-09-17"}})
	assert.Equal(t, []string{"expense 2026-09-08", "expense 2026-09-15", "income 2026-09-10", "income 2026-09-17"}, keys(got))
}

func TestRuleEntries_CreatedSinceIncludesBackfilledOlderDates_Integration(t *testing.T) {
	f := newFixture(t)
	user := uuid.New()
	f.expenseRule(t, user, "monthly", "2026-09-01")
	f.pass(t, "2026-10-20")

	var since time.Time
	require.NoError(t, f.pool.QueryRow(context.Background(), "SELECT clock_timestamp()").Scan(&since))

	// A rule added later with a start in the past: the worker back-fills
	// entries dated before `since`, but inserted after it.
	late := f.incomeRule(t, user, "monthly", "2026-08-10")
	f.pass(t, "2026-10-20")

	got := f.list(t, user, url.Values{"created_since": {since.Format(time.RFC3339Nano)}})
	assert.Equal(t, []string{"income 2026-08-10", "income 2026-09-10", "income 2026-10-10"}, keys(got))
	for _, e := range got {
		assert.Equal(t, late.String(), e.RuleID)
	}
}

func TestRuleEntries_OnlyOwnEntries_Integration(t *testing.T) {
	f := newFixture(t)
	me, other := uuid.New(), uuid.New()
	f.expenseRule(t, me, "monthly", "2026-09-01")
	theirs := f.expenseRule(t, other, "monthly", "2026-09-01")
	f.incomeRule(t, other, "monthly", "2026-09-01")
	f.pass(t, "2026-09-20")

	got := f.list(t, me, nil)
	assert.Equal(t, []string{"expense 2026-09-01"}, keys(got))

	got = f.list(t, me, url.Values{"rule_id": {theirs.String()}})
	assert.Empty(t, got, "another user's rule id shows nothing")
}

func TestRuleEntries_RejectsBadFilters_Integration(t *testing.T) {
	f := newFixture(t)
	for _, params := range []url.Values{
		{"rule_id": {"nope"}},
		{"start_date": {"2026-13-01"}},
		{"end_date": {"yesterday"}},
		{"created_since": {"2026-10-01"}},
		{"start_date": {"2026-09-10"}, "end_date": {"2026-09-01"}},
	} {
		assert.Equal(t, http.StatusBadRequest, f.get(uuid.New(), params).Code, params.Encode())
	}
}
