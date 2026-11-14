package ruleentry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/auth"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// The review feed (#13) pages by a cursor that never passes an entry whose
// transaction hasn't committed yet, so a slow worker pass can't be skipped.

func (f fixture) feed(t *testing.T, user uuid.UUID, params url.Values) Feed {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/budget/rule-entries/feed?"+params.Encode(), nil)
	req = req.WithContext(auth.WithUserID(context.Background(), user))
	rec := httptest.NewRecorder()
	f.h.Feed(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var out Feed
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

// insertEntry inserts one rule entry through q (a pool or an open transaction).
func insertEntry(t *testing.T, q *db.Queries, user, rule uuid.UUID, date string) {
	t.Helper()
	n, err := q.CreateExpenseInstance(context.Background(), db.CreateExpenseInstanceParams{
		UserID: user, Amount: decimal.NewFromInt(1), Currency: "PHP", Description: "x",
		ExpenseDate: pgDay(date), Priority: "need", SourceRuleID: pgtype.UUID{Bytes: rule, Valid: true},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
}

func TestFeed_WaitsForUncommittedPass_Integration(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	user := uuid.New()
	rule := f.expenseRule(t, user, "monthly", "2026-09-01")

	first := f.feed(t, user, nil)
	assert.Empty(t, first.Entries)

	// A slow pass starts first and is still running...
	slow, err := f.pool.Begin(ctx)
	require.NoError(t, err)
	defer slow.Rollback(ctx)
	insertEntry(t, f.q.WithTx(slow), user, rule, "2026-09-01")
	// ...while a later one commits.
	insertEntry(t, f.q, user, rule, "2026-10-01")

	mid := f.feed(t, user, url.Values{"after": {first.Next}})
	assert.Empty(t, mid.Entries, "nothing past a transaction that is still running")

	require.NoError(t, slow.Commit(ctx))
	last := f.feed(t, user, url.Values{"after": {mid.Next}})
	assert.Equal(t, []string{"expense 2026-09-01", "expense 2026-10-01"}, keys(last.Entries),
		"both show up once the slow pass commits; neither is lost")

	again := f.feed(t, user, url.Values{"after": {last.Next}})
	assert.Empty(t, again.Entries, "and they are not repeated")
}

func TestFeed_PagesWithoutRepeatsOrGaps_Integration(t *testing.T) {
	f := newFixture(t)
	user := uuid.New()
	f.expenseRule(t, user, "daily", "2026-09-01")
	f.pass(t, "2026-09-10") // 10 entries in one transaction

	var seen []Entry
	cursor := ""
	for i := 0; i < 10; i++ {
		page := f.feed(t, user, url.Values{"after": {cursor}, "page_size": {"3"}})
		if len(page.Entries) == 0 {
			break
		}
		seen = append(seen, page.Entries...)
		cursor = page.Next
		if i == 0 {
			f.incomeRule(t, user, "daily", "2026-09-01")
			f.pass(t, "2026-09-02") // new rows arrive mid-paging
		}
	}
	ids := map[string]bool{}
	for _, e := range seen {
		assert.False(t, ids[e.ID], "entry %s repeated", e.ID)
		ids[e.ID] = true
	}
	assert.Len(t, seen, 12, "10 expenses plus the 2 incomes that arrived while paging")
}

func TestFeed_RejectsBadCursor_Integration(t *testing.T) {
	f := newFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/api/budget/rule-entries/feed?after=nope", nil)
	req = req.WithContext(auth.WithUserID(context.Background(), uuid.New()))
	rec := httptest.NewRecorder()
	f.h.Feed(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
