package materialize

import (
	"time"

	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

func countRuleIncomes(t *testing.T, ctx context.Context, pool *pgxpool.Pool, ruleID uuid.UUID) int {
	t.Helper()
	var n int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM incomes WHERE source_rule_id = $1`, ruleID).Scan(&n)
	require.NoError(t, err)
	return n
}

// The materialize pass skips paused income rules and picks them back up
// after resume, preserving already-materialized history.
func TestMaterializeDue_PausedIncomeRuleSkippedThenResumed_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")

	rule := seedIncomeRule(t, ctx, q, userID, "daily", "2026-09-01")

	paused, err := q.UpdateRecurringIncomeRule(ctx, db.UpdateRecurringIncomeRuleParams{
		ID:            rule.ID,
		UserID:        rule.UserID,
		Amount:        rule.Amount,
		Currency:      rule.Currency,
		Description:   rule.Description,
		RecurringType: rule.RecurringType,
		StartDate:     rule.StartDate,
		EndDate:       rule.EndDate,
		IsActive:      pgtype.Bool{Bool: false, Valid: true},
	})
	require.NoError(t, err)
	require.False(t, paused.IsActive)

	n, err := MaterializeDue(ctx, pool, q, testZones, today)
	require.NoError(t, err)
	assert.Equal(t, 0, n, "paused rule must produce no instances")
	assert.Equal(t, 0, countRuleIncomes(t, ctx, pool, rule.ID))

	through := ResumeThrough(time.UTC, today)
	resumed, err := q.UpdateRecurringIncomeRule(ctx, db.UpdateRecurringIncomeRuleParams{
		ID:            rule.ID,
		UserID:        rule.UserID,
		Amount:        rule.Amount,
		Currency:      rule.Currency,
		Description:   rule.Description,
		RecurringType: rule.RecurringType,
		StartDate:     rule.StartDate,
		EndDate:       rule.EndDate,
		IsActive:      pgtype.Bool{Bool: true, Valid: true},
		ResumeThrough: through,
	})
	require.NoError(t, err)
	require.True(t, resumed.IsActive)

	n, err = MaterializeDue(ctx, pool, q, testZones, today)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "resumed rule fills from today, not the paused stretch")
	assert.Equal(t, 1, countRuleIncomes(t, ctx, pool, rule.ID))
}

// Income rules carry category, notes and priority (migration 00005) and every
// instance they create copies them, as expense rules already do.
func TestMaterializeIncome_CopiesCategoryNotesPriority_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")
	categoryID := uuid.New()

	rule, err := q.CreateRecurringIncomeRule(ctx, db.CreateRecurringIncomeRuleParams{
		UserID:        userID,
		Amount:        decimal.NewFromFloat(500),
		Currency:      "PHP",
		Description:   "baon",
		RecurringType: "daily",
		StartDate:     pgtype.Date{Time: d("2026-09-10"), Valid: true},
		CategoryID:    pgtype.UUID{Bytes: categoryID, Valid: true},
		Notes:         pgtype.Text{String: "from mom", Valid: true},
		Priority:      pgtype.Text{String: "need", Valid: true},
	})
	require.NoError(t, err)

	check := func(t *testing.T) {
		var cat pgtype.UUID
		var notes pgtype.Text
		var priority string
		err := pool.QueryRow(ctx,
			`SELECT category_id, notes, priority FROM incomes WHERE source_rule_id = $1`, rule.ID).
			Scan(&cat, &notes, &priority)
		require.NoError(t, err)
		assert.Equal(t, categoryID, uuid.UUID(cat.Bytes))
		assert.Equal(t, "from mom", notes.String)
		assert.Equal(t, "need", priority)
	}

	t.Run("worker pass", func(t *testing.T) {
		_, err := pool.Exec(ctx, `DELETE FROM incomes WHERE source_rule_id = $1`, rule.ID)
		require.NoError(t, err)
		n, err := MaterializeDue(ctx, pool, q, testZones, today)
		require.NoError(t, err)
		require.Equal(t, 1, n)
		check(t)
	})
}

// A rule created without a priority defaults to 'want', matching how
// income instances were stamped before income rules had the column.
func TestCreateIncomeRule_PriorityDefaultsToWant_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	var priority string
	rule := seedIncomeRule(t, ctx, q, uuid.New(), "daily", "2026-09-01")
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT priority FROM recurring_income_rules WHERE id = $1`, rule.ID).Scan(&priority))
	assert.Equal(t, "want", priority)
}

// An income occurrence is an income a recurring rule created: the worker
// stamps rule instances one-time with source_rule_id set, so occurrences
// follow the link, not the type. A hand-entered income is never one.
func TestGetIncomeOccurrences_AreRuleCreatedIncomes_Integration(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")

	rule := seedIncomeRule(t, ctx, q, userID, "daily", "2026-09-10")
	_, err := MaterializeDue(ctx, pool, q, testZones, today)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO incomes (user_id, amount, description, date) VALUES ($1, 100, 'gift', '2026-09-10')`, userID)
	require.NoError(t, err)

	got, err := q.GetIncomeOccurrences(ctx, db.GetIncomeOccurrencesParams{UserID: userID})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, rule.ID, uuid.UUID(got[0].SourceRuleID.Bytes))
}
