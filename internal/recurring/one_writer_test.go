package recurring

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/materialize"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

// One writer (#14): saving a rule runs the owner's materialize pass, the same
// code the worker runs, so a rule saved through the API gets exactly the
// entries the worker would have created.

func ruleEntryDates(t *testing.T, ctx context.Context, q *db.Queries, user, rule uuid.UUID) []string {
	t.Helper()
	rows, err := q.ListRuleEntries(ctx, db.ListRuleEntriesParams{
		UserID: user, RuleID: pgUUID(rule), PageLimit: 1000,
	})
	require.NoError(t, err)
	out := []string{}
	for _, r := range rows {
		out = append(out, r.Date.Time.Format(time.DateOnly))
	}
	return sortStrings(out)
}

func TestSavingRuleFillsMissedDaysInOwnersTimezone_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	s := NewServiceWithPool(q, pool)
	user := uuid.New()
	_, err := q.PutUserSettings(ctx, db.PutUserSettingsParams{UserID: user, Timezone: "Asia/Manila"})
	require.NoError(t, err)

	manila, err := time.LoadLocation("Asia/Manila")
	require.NoError(t, err)
	today := time.Now().In(manila)
	start := today.AddDate(0, 0, -6).Format(time.DateOnly)

	rule, err := s.CreateExpenseRule(ctx, user, CreateExpenseRuleRequest{
		Amount: 10, Currency: "PHP", Description: "coffee", RecurringType: "daily", StartDate: start, Priority: "want",
	})
	require.NoError(t, err)

	want := []string{}
	for i := -6; i <= 0; i++ {
		want = append(want, today.AddDate(0, 0, i).Format(time.DateOnly))
	}
	assert.Equal(t, want, ruleEntryDates(t, ctx, q, user, rule.ID))

	// Income rules too, through the same writer.
	inc, err := s.CreateIncomeRule(ctx, user, CreateIncomeRuleRequest{
		Amount: 100, Currency: "PHP", Description: "tips", RecurringType: "daily", StartDate: start,
	})
	require.NoError(t, err)
	assert.Equal(t, want, ruleEntryDates(t, ctx, q, user, inc.ID))
}

func TestSavingRuleDuringWorkerPassMakesNoDuplicates_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	s := NewServiceWithPool(q, pool)
	user := uuid.New()
	start := time.Now().AddDate(0, 0, -30).Format(time.DateOnly)

	var wg sync.WaitGroup
	var rule db.RecurringExpenseRule
	var createErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		rule, createErr = s.CreateExpenseRule(ctx, user, CreateExpenseRuleRequest{
			Amount: 10, Currency: "PHP", Description: "coffee", RecurringType: "daily", StartDate: start, Priority: "want",
		})
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			_, err := materialize.MaterializeDue(ctx, pool, q, time.Now())
			assert.NoError(t, err)
		}
	}()
	wg.Wait()
	require.NoError(t, createErr)
	_, err := materialize.MaterializeDue(ctx, pool, q, time.Now())
	require.NoError(t, err)

	dates := ruleEntryDates(t, ctx, q, user, rule.ID)
	assert.Len(t, dates, 31, "one entry per day, no duplicates")
}

func pgUUID(id uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: id, Valid: true} }

func sortStrings(s []string) []string { sort.Strings(s); return s }
