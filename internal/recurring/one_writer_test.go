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

	"github.com/ejsadiarin/corefinance/internal/auth"
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
	ctx = auth.WithTimezone(ctx, "Asia/Manila") // the token's tz claim

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

// A token without tz (#18) dates a rule save in the owner's last-seen zone,
// not UTC: the next due time stays midnight in that zone after a create and
// an update. Kiritimati (UTC+14) makes the difference visible at any hour.
func TestSavingRuleWithoutTokenZoneUsesLastSeenZone_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	s := NewServiceWithPool(q, pool)
	user := uuid.New()
	kiritimati, err := time.LoadLocation("Pacific/Kiritimati")
	require.NoError(t, err)
	require.NoError(t, materialize.ObserveTimezone(ctx, pool, user, kiritimati, time.Now()))

	today := time.Now().In(kiritimati)
	y, m, d := today.AddDate(0, 0, 1).Date()
	tomorrowMidnight := time.Date(y, m, d, 0, 0, 0, 0, kiritimati)
	wake := func() time.Time {
		t.Helper()
		w, err := materialize.NextWake(ctx, q, time.Now())
		require.NoError(t, err)
		return w
	}

	req := CreateExpenseRuleRequest{
		Amount: 10, Currency: "PHP", Description: "coffee", RecurringType: "daily",
		StartDate: today.Format(time.DateOnly), Priority: "want",
	}
	rule, err := s.CreateExpenseRule(ctx, user, req) // ctx has no tz
	require.NoError(t, err)
	assert.Equal(t, []string{today.Format(time.DateOnly)}, ruleEntryDates(t, ctx, q, user, rule.ID))
	assert.True(t, tomorrowMidnight.Equal(wake()), "after create: %s", wake())

	_, err = s.UpdateExpenseRule(ctx, rule.ID, user, UpdateExpenseRuleRequest{
		Amount: 12, Currency: req.Currency, Description: req.Description,
		RecurringType: req.RecurringType, StartDate: req.StartDate, Priority: req.Priority,
	})
	require.NoError(t, err)
	assert.True(t, tomorrowMidnight.Equal(wake()), "after update: %s", wake())
}

func TestSavingRuleDuringWorkerPassMakesNoDuplicates_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	s := NewServiceWithPool(q, pool)
	user := uuid.New()
	start := time.Now().UTC().AddDate(0, 0, -30).Format(time.DateOnly)

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
			_, err := materialize.MaterializeDue(ctx, pool, q, materialize.FixedZones{}, time.Now())
			assert.NoError(t, err)
		}
	}()
	wg.Wait()
	require.NoError(t, createErr)
	_, err := materialize.MaterializeDue(ctx, pool, q, materialize.FixedZones{}, time.Now())
	require.NoError(t, err)

	dates := ruleEntryDates(t, ctx, q, user, rule.ID)
	assert.Len(t, dates, 31, "one entry per day, no duplicates")
}

func pgUUID(id uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: id, Valid: true} }

func sortStrings(s []string) []string { sort.Strings(s); return s }
