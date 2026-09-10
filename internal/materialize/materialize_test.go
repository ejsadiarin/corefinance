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

func seedExpenseRule(t *testing.T, ctx context.Context, q *db.Queries, userID uuid.UUID, cadence, start string, end *string) db.RecurringExpenseRule {
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

func seedIncomeRule(t *testing.T, ctx context.Context, q *db.Queries, userID uuid.UUID, cadence, start string) db.RecurringIncomeRule {
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

// Daily rule with start in the past: one pass creates exactly today's
// instance with one-time/NULL-start/NULL-end/source-set/posted shape.
func TestMaterializeDue_DailyCreatesTodaysInstance(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")

	rule := seedExpenseRule(t, ctx, q, userID, "daily", "2026-09-01", nil)

	n, err := MaterializeDue(ctx, pool, q, today)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	var got db.Expense
	err = pool.QueryRow(ctx,
		`SELECT id, user_id, category_id, amount, currency, description, notes, expense_date, recurring_type, priority, status, is_debt, start_date, end_date, source_rule_id, created_at, updated_at
		 FROM expenses WHERE source_rule_id = $1`, rule.ID).
		Scan(&got.ID, &got.UserID, &got.CategoryID, &got.Amount, &got.Currency,
			&got.Description, &got.Notes, &got.ExpenseDate, &got.RecurringType,
			&got.Priority, &got.Status, &got.IsDebt, &got.StartDate, &got.EndDate,
			&got.SourceRuleID, &got.CreatedAt, &got.UpdatedAt)
	require.NoError(t, err)
	assert.Equal(t, "one-time", got.RecurringType)
	assert.False(t, got.StartDate.Valid, "instance start_date must be NULL")
	assert.False(t, got.EndDate.Valid, "instance end_date must be NULL")
	assert.True(t, got.SourceRuleID.Valid)
	assert.Equal(t, rule.ID, uuid.UUID(got.SourceRuleID.Bytes))
	assert.Equal(t, "posted", got.Status)
	assert.True(t, got.ExpenseDate.Time.Equal(today))
}

// Double run for the same day inserts nothing the second time.
func TestMaterializeDue_DoubleRunIdempotent(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")

	rule := seedExpenseRule(t, ctx, q, userID, "daily", "2026-09-01", nil)
	incomeRule := seedIncomeRule(t, ctx, q, userID, "daily", "2026-09-01")

	n1, err := MaterializeDue(ctx, pool, q, today)
	require.NoError(t, err)
	require.Equal(t, 2, n1)

	n2, err := MaterializeDue(ctx, pool, q, today)
	require.NoError(t, err)
	assert.Equal(t, 0, n2, "second run must insert 0 rows")

	assert.Equal(t, 1, countRuleExpenses(t, ctx, pool, rule.ID))
	var incomeCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM incomes WHERE source_rule_id = $1`, incomeRule.ID).Scan(&incomeCount))
	assert.Equal(t, 1, incomeCount)
}

// Ended (end_date < today) and future (start_date > today) rules produce nothing.
func TestMaterializeDue_EndedAndFutureRulesProduceNothing(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-10")

	endedEnd := "2026-09-09"
	seedExpenseRule(t, ctx, q, userID, "daily", "2026-08-01", &endedEnd)
	seedExpenseRule(t, ctx, q, userID, "weekly", "2026-09-11", nil)

	n, err := MaterializeDue(ctx, pool, q, today)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	var total int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM expenses`).Scan(&total))
	assert.Equal(t, 0, total)
}

// Monthly rule starting Jan 31 materializes Feb 28 when run on Feb 28, and
// nothing when run on Feb 27.
func TestMaterializeDue_MonthlyClamp(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()

	rule := seedExpenseRule(t, ctx, q, userID, "monthly", "2026-01-31", nil)

	n, err := MaterializeDue(ctx, pool, q, d("2026-02-27"))
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	n, err = MaterializeDue(ctx, pool, q, d("2026-02-28"))
	require.NoError(t, err)
	require.Equal(t, 1, n)
	assert.Equal(t, 1, countRuleExpenses(t, ctx, pool, rule.ID))

	var expenseDate pgtype.Date
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT expense_date FROM expenses WHERE source_rule_id = $1`, rule.ID).Scan(&expenseDate))
	assert.True(t, expenseDate.Time.Equal(d("2026-02-28")))
}

// Legacy weekly income becomes a real rule plus history; the legacy row is
// flipped to one-time and zero legacy-predicate rows remain.
func TestBackfillLegacy_WeeklyIncome(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-08-15") // Saturdays: 08-01, 08-08, 08-15

	legacy, err := q.CreateIncome(ctx, db.CreateIncomeParams{
		UserID:        userID,
		Amount:        decimal.NewFromFloat(500),
		Currency:      "PHP",
		Description:   "allowance",
		Date:          pgtype.Date{Time: d("2026-08-01"), Valid: true},
		RecurringType: "weekly",
		Priority:      "want",
		Status:        "posted",
		StartDate:     pgtype.Date{Time: d("2026-08-01"), Valid: true},
	})
	require.NoError(t, err)

	before, err := CountLegacyRows(ctx, q)
	require.NoError(t, err)
	require.Equal(t, int64(1), before)

	res, err := BackfillLegacy(ctx, pool, q, today)
	require.NoError(t, err)
	assert.Equal(t, 1, res.IncomeRules)
	assert.Equal(t, 2, res.IncomeInstances, "08-08 and 08-15; 08-01 anchor kept")

	rules, err := q.ListRecurringIncomeRules(ctx, userID)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.True(t, decimal.NewFromFloat(500).Equal(rules[0].Amount))
	assert.Equal(t, "allowance", rules[0].Description)
	assert.Equal(t, "weekly", rules[0].RecurringType)

	var instanceCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM incomes WHERE source_rule_id = $1 AND status = 'posted' AND recurring_type = 'one-time'`,
		rules[0].ID).Scan(&instanceCount))
	assert.Equal(t, 2, instanceCount)

	var flipped db.Income
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT id, user_id, category_id, amount, currency, description, notes, date, recurring_type, priority, status, start_date, end_date, source_rule_id, created_at, updated_at
		 FROM incomes WHERE id = $1`, legacy.ID).
		Scan(&flipped.ID, &flipped.UserID, &flipped.CategoryID, &flipped.Amount, &flipped.Currency,
			&flipped.Description, &flipped.Notes, &flipped.Date, &flipped.RecurringType,
			&flipped.Priority, &flipped.Status, &flipped.StartDate, &flipped.EndDate,
			&flipped.SourceRuleID, &flipped.CreatedAt, &flipped.UpdatedAt))
	assert.Equal(t, "one-time", flipped.RecurringType)

	after, err := CountLegacyRows(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, int64(0), after, "zero rows must match the legacy predicate")
}

// Legacy expense conversion copies category/notes/priority/is_debt into the
// rule and its history.
func TestBackfillLegacy_ExpenseCopiesFields(t *testing.T) {
	pool, q := newMaterializeTest(t)
	ctx := context.Background()
	userID := uuid.New()
	today := d("2026-09-03")

	cat, err := q.CreateExpenseCategory(ctx, db.CreateExpenseCategoryParams{
		UserID: userID,
		Name:   "food",
	})
	require.NoError(t, err)
	notes := "weekly groceries"

	_, err = q.CreateExpense(ctx, db.CreateExpenseParams{
		UserID:        userID,
		CategoryID:    pgtype.UUID{Bytes: cat.ID, Valid: true},
		Amount:        decimal.NewFromFloat(2500),
		Currency:      "PHP",
		Description:   "groceries",
		Notes:         pgtype.Text{String: notes, Valid: true},
		ExpenseDate:   pgtype.Date{Time: d("2026-09-01"), Valid: true},
		RecurringType: "weekly",
		Priority:      "need",
		Status:        "posted",
		IsDebt:        true,
		StartDate:     pgtype.Date{Time: d("2026-09-01"), Valid: true},
	})
	require.NoError(t, err)

	res, err := BackfillLegacy(ctx, pool, q, today)
	require.NoError(t, err)
	assert.Equal(t, 1, res.ExpenseRules)
	assert.Equal(t, 0, res.ExpenseInstances, "only 09-01 due by 09-03 and it is the anchor")

	rules, err := q.ListRecurringExpenseRules(ctx, userID)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "groceries", rules[0].Description)
	assert.True(t, rules[0].CategoryID.Valid)
	assert.Equal(t, cat.ID, uuid.UUID(rules[0].CategoryID.Bytes))
	assert.Equal(t, "need", rules[0].Priority)

	after, err := CountLegacyRows(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, int64(0), after)
}
