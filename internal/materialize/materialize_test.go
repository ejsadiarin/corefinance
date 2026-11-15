package materialize

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func newMaterializeTest(t *testing.T) (*pgxpool.Pool, *db.Queries) {
	t.Helper()
	pool := testutil.SetupTestDB(t)
	return pool, db.New(pool)
}

func seedExpenseRule(t *testing.T, ctx context.Context, q *db.Queries, userID uuid.UUID, cadence, start string, end *string) db.Rule {
	t.Helper()
	rule, err := q.CreateRecurringExpenseRule(ctx, db.CreateRecurringExpenseRuleParams{
		UserID:        userID,
		Description:   "rent",
		Amount:        decimal.NewFromFloat(100),
		Currency:      "PHP",
		RecurringType: cadence,
		StartDate:     pgtype.Date{Time: d(start), Valid: true},
		EndDate:       endDatePtr(end),
		Priority:      "need",
	})
	require.NoError(t, err)
	return rule
}

func seedIncomeRule(t *testing.T, ctx context.Context, q *db.Queries, userID uuid.UUID, cadence, start string) db.Rule {
	t.Helper()
	rule, err := q.CreateRecurringIncomeRule(ctx, db.CreateRecurringIncomeRuleParams{
		UserID:        userID,
		Amount:        decimal.NewFromFloat(500),
		Currency:      "PHP",
		Description:   "salary",
		RecurringType: cadence,
		StartDate:     pgtype.Date{Time: d(start), Valid: true},
	})
	require.NoError(t, err)
	return rule
}

func endDatePtr(end *string) pgtype.Date {
	if end == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: d(*end), Valid: true}
}

func countRuleExpenses(t *testing.T, ctx context.Context, pool *pgxpool.Pool, ruleID uuid.UUID) int {
	t.Helper()
	var n int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM expenses WHERE source_rule_id = $1`, ruleID).Scan(&n)
	require.NoError(t, err)
	return n
}

// Daily rule starting today: one pass creates exactly today's instance with
// one-time/NULL-start/NULL-end/source-set/posted shape.
func TestMaterializeDue_DailyCreatesTodaysInstance_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")

	rule := seedExpenseRule(t, ctx, q, userID, "daily", "2026-09-10", nil)

	n, err := MaterializeDue(ctx, pool, q, testZones, today)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	var got db.Expense
	err = pool.QueryRow(ctx,
		`SELECT id, user_id, category_id, amount, currency, description, notes, expense_date, priority, status, is_debt, source_rule_id, created_at, updated_at
		 FROM expenses WHERE source_rule_id = $1`, rule.ID).
		Scan(&got.ID, &got.UserID, &got.CategoryID, &got.Amount, &got.Currency,
			&got.Description, &got.Notes, &got.ExpenseDate,
			&got.Priority, &got.Status, &got.IsDebt,
			&got.SourceRuleID, &got.CreatedAt, &got.UpdatedAt)
	require.NoError(t, err)
	assert.True(t, got.SourceRuleID.Valid)
	assert.Equal(t, rule.ID, uuid.UUID(got.SourceRuleID.Bytes))
	assert.Equal(t, "posted", got.Status)
	assert.True(t, got.ExpenseDate.Time.Equal(today))
}

// Double run for the same day inserts nothing the second time.
func TestMaterializeDue_DoubleRunIdempotent_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")

	rule := seedExpenseRule(t, ctx, q, userID, "daily", "2026-09-10", nil)
	incomeRule := seedIncomeRule(t, ctx, q, userID, "daily", "2026-09-10")

	n1, err := MaterializeDue(ctx, pool, q, testZones, today)
	require.NoError(t, err)
	require.Equal(t, 2, n1)

	n2, err := MaterializeDue(ctx, pool, q, testZones, today)
	require.NoError(t, err)
	assert.Equal(t, 0, n2, "second run must insert 0 rows")

	assert.Equal(t, 1, countRuleExpenses(t, ctx, pool, rule.ID))
	var incomeCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM incomes WHERE source_rule_id = $1`, incomeRule.ID).Scan(&incomeCount))
	assert.Equal(t, 1, incomeCount)
}

// A future rule (start_date > today) produces nothing. An ended rule
// (end_date < today) is filled up to its end date and never past it.
func TestMaterializeDue_FutureRuleNothing_EndedRuleStopsAtEnd_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")

	endedEnd := "2026-09-09"
	ended := seedExpenseRule(t, ctx, q, userID, "daily", "2026-09-01", &endedEnd)
	future := seedExpenseRule(t, ctx, q, userID, "weekly", "2026-09-11", nil)

	n, err := MaterializeDue(ctx, pool, q, testZones, today)
	require.NoError(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, 9, countRuleExpenses(t, ctx, pool, ended.ID))
	assert.Equal(t, 0, countRuleExpenses(t, ctx, pool, future.ID))

	var last pgtype.Date
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT max(expense_date) FROM expenses WHERE source_rule_id = $1`, ended.ID).Scan(&last))
	assert.True(t, last.Time.Equal(d("2026-09-09")))

	_, err = MaterializeDue(ctx, pool, q, testZones, d("2026-09-20"))
	require.NoError(t, err)
	assert.Equal(t, 2, countRuleExpenses(t, ctx, pool, future.ID), "future rule gets 09-11 and 09-18 once it starts")
	assert.Equal(t, 9, countRuleExpenses(t, ctx, pool, ended.ID), "ended rule gets nothing after its end date")
}

// Monthly rule starting Jan 31: run on Feb 27 creates only Jan 31; run on
// Feb 28 adds the clamped Feb 28.
func TestMaterializeDue_MonthlyClamp_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()

	rule := seedExpenseRule(t, ctx, q, userID, "monthly", "2026-01-31", nil)

	n, err := MaterializeDue(ctx, pool, q, testZones, d("2026-02-27"))
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = MaterializeDue(ctx, pool, q, testZones, d("2026-02-28"))
	require.NoError(t, err)
	require.Equal(t, 1, n)
	assert.Equal(t, 2, countRuleExpenses(t, ctx, pool, rule.ID))

	var expenseDate pgtype.Date
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT max(expense_date) FROM expenses WHERE source_rule_id = $1`, rule.ID).Scan(&expenseDate))
	assert.True(t, expenseDate.Time.Equal(d("2026-02-28")))
}
