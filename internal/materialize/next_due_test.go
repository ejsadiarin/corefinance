package materialize

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Next-due scheduling (#16, ADR-0001): each rule records when its next
// occurrence starts in the owner's timezone; the worker fills rules whose
// time has come and wakes for the earliest one.

func storedNextDue(t *testing.T, ctx context.Context, pool *pgxpool.Pool, ruleID uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COALESCE(to_char(next_due_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI"Z"'), 'null') FROM rules WHERE id = $1`,
		ruleID).Scan(&s))
	return s
}

func TestNextDue_ManilaTuesdayIsDueAtMondayEveningUTC_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	manila := uuid.New()
	setTimezone(t, ctx, q, manila, "Asia/Manila")
	rule := seedExpenseRule(t, ctx, q, manila, "weekly", "2026-11-10", nil) // a Tuesday

	// Monday 10:00 UTC (18:00 in Manila): nothing yet, due at Tuesday 00:00 Manila.
	_, err := MaterializeDue(ctx, pool, q, testZones, time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Empty(t, ruleExpenseDates(t, ctx, pool, rule.ID))
	assert.Equal(t, "2026-11-09T16:00Z", storedNextDue(t, ctx, pool, rule.ID))

	wake, err := NextWake(ctx, q, time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 11, 9, 16, 0, 0, 0, time.UTC), wake.UTC())

	// Thirty seconds after Tuesday starts in Manila, the entry is there.
	n, err := MaterializeDue(ctx, pool, q, testZones, time.Date(2026, 11, 9, 16, 0, 30, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"2026-11-10"}, ruleExpenseDates(t, ctx, pool, rule.ID))
	assert.Equal(t, "2026-11-16T16:00Z", storedNextDue(t, ctx, pool, rule.ID), "next Tuesday, Manila midnight")
}

func TestNextDue_WorkerDownThreeDaysFillsOnce_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	rule := seedExpenseRule(t, ctx, q, uuid.New(), "daily", "2026-09-01", nil)

	_, err := MaterializeDue(ctx, pool, q, testZones, time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	n, err := MaterializeDue(ctx, pool, q, testZones, time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	n, err = MaterializeDue(ctx, pool, q, testZones, time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, []string{"2026-09-01", "2026-09-02", "2026-09-03", "2026-09-04"}, ruleExpenseDates(t, ctx, pool, rule.ID))
}

func TestNextDue_EndedRuleIsNeverDueAgain_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	end := "2026-09-02"
	rule := seedExpenseRule(t, ctx, q, uuid.New(), "daily", "2026-09-01", &end)

	_, err := MaterializeDue(ctx, pool, q, testZones, time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, "infinity", nextDueAtRaw(t, ctx, pool, rule.ID))
	_, err = NextWake(ctx, q, time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC))
	assert.ErrorIs(t, err, ErrNothingDue)
}

// Changing the owner's timezone (at the gateway) recomputes their rules'
// next due time.
func TestNextDue_TimezoneChangeRecomputes_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	user := uuid.New()
	rule := seedExpenseRule(t, ctx, q, user, "weekly", "2026-11-10", nil)
	_, err := MaterializeDue(ctx, pool, q, testZones, time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	assert.Equal(t, "2026-11-10T00:00Z", storedNextDue(t, ctx, pool, rule.ID), "UTC until a timezone is saved")

	// The gateway now says Manila, and the user makes no request: once the
	// zone is due a re-check, a pass sees the change and recomputes.
	setTimezone(t, ctx, q, user, "Asia/Manila")
	_, err = MaterializeDue(ctx, pool, q, testZones, time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC).Add(ZoneRefreshEvery+time.Minute))
	require.NoError(t, err)
	assert.Equal(t, "2026-11-09T16:00Z", storedNextDue(t, ctx, pool, rule.ID))
}

func nextDueAtRaw(t *testing.T, ctx context.Context, pool *pgxpool.Pool, ruleID uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, pool.QueryRow(ctx, `SELECT COALESCE(next_due_at::text, 'null') FROM rules WHERE id = $1`, ruleID).Scan(&s))
	return s
}
