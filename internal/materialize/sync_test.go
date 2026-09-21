package materialize

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func countRuleIncomes(t *testing.T, ctx context.Context, pool *pgxpool.Pool, ruleID uuid.UUID) int {
	t.Helper()
	var n int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM incomes WHERE source_rule_id = $1`, ruleID).Scan(&n)
	require.NoError(t, err)
	return n
}

// Due expense rule: one insert, second call is a no-op (single row total).
func TestMaterializeExpenseRuleToday_DueInsertsAndIdempotent(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")

	rule := seedExpenseRule(t, ctx, q, userID, "daily", "2026-09-01", nil)

	n, err := MaterializeExpenseRuleToday(ctx, q, rule, today)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	assert.Equal(t, 1, countRuleExpenses(t, ctx, pool, rule.ID))

	n, err = MaterializeExpenseRuleToday(ctx, q, rule, today)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n, "second call must insert 0 rows")
	assert.Equal(t, 1, countRuleExpenses(t, ctx, pool, rule.ID))
}

// Expense rules outside the active window or missing today insert nothing.
func TestMaterializeExpenseRuleToday_NotDue(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")

	future := seedExpenseRule(t, ctx, q, userID, "daily", "2026-09-11", nil)
	ended := "2026-09-09"
	old := seedExpenseRule(t, ctx, q, userID, "daily", "2026-09-01", &ended)
	inactive := seedExpenseRule(t, ctx, q, userID, "daily", "2026-09-01", nil)
	inactive.IsActive = false
	// Weekly rule starting Monday 2026-09-07 is not due Thursday 2026-09-10.
	weekly := seedExpenseRule(t, ctx, q, userID, "weekly", "2026-09-07", nil)

	cases := []struct {
		name string
		call func() (int64, error)
		id   uuid.UUID
	}{
		{"future start", func() (int64, error) {
			return MaterializeExpenseRuleToday(ctx, q, future, today)
		}, future.ID},
		{"ended", func() (int64, error) {
			return MaterializeExpenseRuleToday(ctx, q, old, today)
		}, old.ID},
		{"inactive", func() (int64, error) {
			return MaterializeExpenseRuleToday(ctx, q, inactive, today)
		}, inactive.ID},
		{"weekly cadence misses today", func() (int64, error) {
			return MaterializeExpenseRuleToday(ctx, q, weekly, today)
		}, weekly.ID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n, err := tc.call()
			require.NoError(t, err)
			assert.Equal(t, int64(0), n)
			assert.Equal(t, 0, countRuleExpenses(t, ctx, pool, tc.id))
		})
	}
}

// Due income rule: one insert, second call is a no-op (single row total).
func TestMaterializeIncomeRuleToday_DueInsertsAndIdempotent(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")

	rule := seedIncomeRule(t, ctx, q, userID, "daily", "2026-09-01")

	n, err := MaterializeIncomeRuleToday(ctx, q, rule, today)
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	assert.Equal(t, 1, countRuleIncomes(t, ctx, pool, rule.ID))

	n, err = MaterializeIncomeRuleToday(ctx, q, rule, today)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n, "second call must insert 0 rows")
	assert.Equal(t, 1, countRuleIncomes(t, ctx, pool, rule.ID))
}

// Income rule starting in the future inserts nothing.
func TestMaterializeIncomeRuleToday_NotDue(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")

	rule := seedIncomeRule(t, ctx, q, userID, "daily", "2026-09-11")

	n, err := MaterializeIncomeRuleToday(ctx, q, rule, today)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
	assert.Equal(t, 0, countRuleIncomes(t, ctx, pool, rule.ID))
}
