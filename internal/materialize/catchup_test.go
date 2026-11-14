package materialize

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// Watermark catch-up (#7): the worker fills what it missed while down,
// never recreates deleted entries, never touches edited or skipped ones.

func pd(s string) pgtype.Date { return pgtype.Date{Time: d(s), Valid: true} }

func setExpenseWatermark(t *testing.T, ctx context.Context, pool *pgxpool.Pool, ruleID uuid.UUID, through string) {
	t.Helper()
	_, err := pool.Exec(ctx, `UPDATE recurring_expense_rules SET materialized_through = $2 WHERE id = $1`, ruleID, d(through))
	require.NoError(t, err)
}

func ruleExpenseDates(t *testing.T, ctx context.Context, pool *pgxpool.Pool, ruleID uuid.UUID) []string {
	t.Helper()
	rows, err := pool.Query(ctx, `SELECT expense_date FROM expenses WHERE source_rule_id = $1 ORDER BY expense_date`, ruleID)
	require.NoError(t, err)
	defer rows.Close()
	var out []string
	for rows.Next() {
		var dt pgtype.Date
		require.NoError(t, rows.Scan(&dt))
		out = append(out, dt.Time.Format("2006-01-02"))
	}
	require.NoError(t, rows.Err())
	return out
}

func TestPendingDates(t *testing.T) {
	start := d("2026-09-01")
	today := d("2026-09-10")

	t.Run("no watermark fills from start", func(t *testing.T) {
		dates, through := PendingDates(start, "daily", pgtype.Date{}, pgtype.Date{}, today, 100)
		assert.Len(t, dates, 10)
		assert.True(t, through.Equal(today))
	})
	t.Run("watermark skips filled dates", func(t *testing.T) {
		dates, through := PendingDates(start, "daily", pd("2026-09-07"), pgtype.Date{}, today, 100)
		assert.Equal(t, []time.Time{d("2026-09-08"), d("2026-09-09"), d("2026-09-10")}, dates)
		assert.True(t, through.Equal(today))
	})
	t.Run("bounded by end", func(t *testing.T) {
		dates, through := PendingDates(start, "daily", pd("2026-09-03"), pd("2026-09-05"), today, 100)
		assert.Equal(t, []time.Time{d("2026-09-04"), d("2026-09-05")}, dates)
		assert.True(t, through.Equal(d("2026-09-05")))
	})
	t.Run("limit stops at the last returned date", func(t *testing.T) {
		dates, through := PendingDates(start, "daily", pgtype.Date{}, pgtype.Date{}, today, 4)
		assert.Len(t, dates, 4)
		assert.True(t, through.Equal(d("2026-09-04")))
	})
	t.Run("no due date still moves to the bound", func(t *testing.T) {
		dates, through := PendingDates(start, "monthly", pd("2026-09-01"), pgtype.Date{}, today, 100)
		assert.Empty(t, dates)
		assert.True(t, through.Equal(today))
	})
}

// Worker down Mon-Thu, weekly-Tuesday rule: Thursday's restart creates the
// Tuesday entry once; a second pass inserts nothing.
func TestCatchUp_WeeklyTuesdayMissedWhileDown_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	rule := seedExpenseRule(t, ctx, q, uuid.New(), "weekly", "2026-11-03", nil) // a Tuesday

	_, err := MaterializeDue(ctx, pool, q, d("2026-11-09")) // Monday
	require.NoError(t, err)
	require.Equal(t, []string{"2026-11-03"}, ruleExpenseDates(t, ctx, pool, rule.ID))

	n, err := MaterializeDue(ctx, pool, q, d("2026-11-12")) // Thursday
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"2026-11-03", "2026-11-10"}, ruleExpenseDates(t, ctx, pool, rule.ID))

	n, err = MaterializeDue(ctx, pool, q, d("2026-11-12"))
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

// Monthly rule from 2026-02-23 with only its February entry (the production
// case in #4): the next pass fills March through the current month.
func TestCatchUp_MonthlyGapFromWatermark_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	rule := seedExpenseRule(t, ctx, q, uuid.New(), "monthly", "2026-02-23", nil)

	_, err := MaterializeDue(ctx, pool, q, d("2026-02-23"))
	require.NoError(t, err)

	n, err := MaterializeDue(ctx, pool, q, d("2026-11-14"))
	require.NoError(t, err)
	assert.Equal(t, 8, n)
	assert.Equal(t, []string{
		"2026-02-23", "2026-03-23", "2026-04-23", "2026-05-23", "2026-06-23",
		"2026-07-23", "2026-08-23", "2026-09-23", "2026-10-23",
	}, ruleExpenseDates(t, ctx, pool, rule.ID))
}

// A rule entry the user deleted is not recreated; one they edited or skipped
// is left as it is.
func TestCatchUp_DeletedEditedSkippedEntriesStay_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	rule := seedExpenseRule(t, ctx, q, uuid.New(), "daily", "2026-09-01", nil)

	_, err := MaterializeDue(ctx, pool, q, d("2026-09-05"))
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `DELETE FROM expenses WHERE source_rule_id = $1 AND expense_date = '2026-09-03'`, rule.ID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE expenses SET amount = 300 WHERE source_rule_id = $1 AND expense_date = '2026-09-04'`, rule.ID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE expenses SET status = 'skipped' WHERE source_rule_id = $1 AND expense_date = '2026-09-05'`, rule.ID)
	require.NoError(t, err)

	for _, day := range []string{"2026-09-05", "2026-09-06", "2026-09-06"} {
		_, err := MaterializeDue(ctx, pool, q, d(day))
		require.NoError(t, err)
	}

	assert.Equal(t, []string{"2026-09-01", "2026-09-02", "2026-09-04", "2026-09-05", "2026-09-06"},
		ruleExpenseDates(t, ctx, pool, rule.ID))
	var amount float64
	var status string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT amount::float8 FROM expenses WHERE source_rule_id = $1 AND expense_date = '2026-09-04'`, rule.ID).Scan(&amount))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT status FROM expenses WHERE source_rule_id = $1 AND expense_date = '2026-09-05'`, rule.ID).Scan(&status))
	assert.Equal(t, 300.0, amount)
	assert.Equal(t, "skipped", status)
}

// A rule that ended while the worker was down gets its entries up to its end
// date and none after.
func TestCatchUp_RuleEndedWhileDown_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	end := "2026-09-05"
	rule := seedExpenseRule(t, ctx, q, uuid.New(), "daily", "2026-09-01", &end)
	setExpenseWatermark(t, ctx, pool, rule.ID, "2026-09-02")

	n, err := MaterializeDue(ctx, pool, q, d("2026-09-10"))
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []string{"2026-09-03", "2026-09-04", "2026-09-05"}, ruleExpenseDates(t, ctx, pool, rule.ID))

	n, err = MaterializeDue(ctx, pool, q, d("2026-09-11"))
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

// A paused rule gets nothing; when resumed it fills from the resume day, not
// the stretch it was paused.
func TestCatchUp_PausedRuleSkipsPausedStretch_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	rule := seedExpenseRule(t, ctx, q, uuid.New(), "daily", "2026-09-01", nil)

	_, err := MaterializeDue(ctx, pool, q, d("2026-09-03"))
	require.NoError(t, err)

	update := func(active bool, today time.Time) {
		_, err := q.UpdateRecurringExpenseRule(ctx, db.UpdateRecurringExpenseRuleParams{
			ID: rule.ID, UserID: rule.UserID, Description: rule.Description, Amount: rule.Amount,
			Currency: rule.Currency, RecurringType: rule.RecurringType, StartDate: rule.StartDate,
			EndDate: rule.EndDate, Priority: rule.Priority,
			IsActive:      pgtype.Bool{Bool: active, Valid: true},
			ResumeThrough: ResumeThrough(today),
		})
		require.NoError(t, err)
	}

	update(false, d("2026-09-03"))
	n, err := MaterializeDue(ctx, pool, q, d("2026-09-08"))
	require.NoError(t, err)
	assert.Equal(t, 0, n, "paused rule gets nothing")

	update(true, d("2026-09-10"))
	_, err = MaterializeDue(ctx, pool, q, d("2026-09-11"))
	require.NoError(t, err)
	assert.Equal(t, []string{"2026-09-01", "2026-09-02", "2026-09-03", "2026-09-10", "2026-09-11"},
		ruleExpenseDates(t, ctx, pool, rule.ID))
}

// A long outage fills over several passes, MaxDatesPerRulePass at a time.
func TestCatchUp_LongOutageFillsInBatches_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	rule := seedExpenseRule(t, ctx, q, uuid.New(), "daily", "2025-01-01", nil)
	today := d("2026-09-10")
	total := len(DueDates(d("2025-01-01"), "daily", nil, today))
	require.Greater(t, total, MaxDatesPerRulePass)

	n, err := MaterializeDue(ctx, pool, q, today)
	require.NoError(t, err)
	assert.Equal(t, MaxDatesPerRulePass, n)

	n, err = MaterializeDue(ctx, pool, q, today)
	require.NoError(t, err)
	assert.Equal(t, total-MaxDatesPerRulePass, n)

	n, err = MaterializeDue(ctx, pool, q, today)
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, total, countRuleExpenses(t, ctx, pool, rule.ID))
}
