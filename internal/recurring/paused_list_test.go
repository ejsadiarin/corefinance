package recurring

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/tag"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

// A paused rule leaves the default list but stays in the list that asks
// for paused rules, so the web can show it and resume it.
func TestPausedRulesListOnlyWhenAskedFor_Integration(t *testing.T) {
	ctx := context.Background()
	q := db.New(testutil.SetupTestDB(t))
	s := NewService(q)
	user := uuid.New()

	exp, err := s.CreateExpenseRule(ctx, user, CreateExpenseRuleRequest{
		Amount: 85, Currency: "PHP", Description: "spotify", RecurringType: "monthly",
		StartDate: "2026-02-23", Priority: "need",
	})
	require.NoError(t, err)
	inc, err := s.CreateIncomeRule(ctx, user, CreateIncomeRuleRequest{
		Amount: 500, Currency: "PHP", Description: "baon", RecurringType: "weekly", StartDate: "2026-11-19",
	})
	require.NoError(t, err)

	off := false
	_, err = s.UpdateExpenseRule(ctx, exp.ID, user, UpdateExpenseRuleRequest{
		Amount: 85, Currency: "PHP", Description: "spotify", RecurringType: "monthly",
		StartDate: "2026-02-23", Priority: "need", IsActive: &off,
	})
	require.NoError(t, err)
	_, err = s.UpdateIncomeRule(ctx, inc.ID, user, UpdateIncomeRuleRequest{
		Amount: 500, Currency: "PHP", Description: "baon", RecurringType: "weekly", StartDate: "2026-11-19", IsActive: &off,
	})
	require.NoError(t, err)

	active, err := s.ListExpenseRules(ctx, user)
	require.NoError(t, err)
	assert.Empty(t, active, "a paused expense rule leaves the default list")
	all, err := s.ListExpenseRulesWithPaused(ctx, user)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.False(t, all[0].IsActive)

	activeInc, err := s.ListIncomeRules(ctx, user)
	require.NoError(t, err)
	assert.Empty(t, activeInc)
	allInc, err := s.ListIncomeRulesWithPaused(ctx, user)
	require.NoError(t, err)
	require.Len(t, allInc, 1)
	assert.False(t, allInc[0].IsActive)

	other, err := s.ListExpenseRulesWithPaused(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, other, "another user's rules never list")
}

// Tag links name every tagged record of a user, and only theirs.
func TestTagLinksListEveryTaggedRecord_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	tags := tag.NewService(q)
	user := uuid.New()

	her, err := tags.Create(ctx, user, tag.CreateRequest{Name: "her"})
	require.NoError(t, err)
	var expenseID, incomeID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO expenses (user_id, description, amount, currency, exchange_rate, amount_php, rate_date, expense_date) VALUES ($1, 'lunch', 160, 'PHP', 1, 160, '2026-11-13', '2026-11-13') RETURNING id`,
		user).Scan(&expenseID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO incomes (user_id, description, amount, currency, exchange_rate, amount_php, rate_date, date) VALUES ($1, 'gift', 200, 'PHP', 1, 200, '2026-11-13', '2026-11-13') RETURNING id`,
		user).Scan(&incomeID))
	require.NoError(t, tags.AddTagToExpense(ctx, expenseID, her.ID))
	require.NoError(t, tags.AddTagToIncome(ctx, incomeID, her.ID))

	links, err := tags.Links(ctx, user)
	require.NoError(t, err)
	assert.Equal(t, []tag.Link{{RecordID: expenseID, TagID: her.ID}}, links.Expenses)
	assert.Equal(t, []tag.Link{{RecordID: incomeID, TagID: her.ID}}, links.Incomes)

	none, err := tags.Links(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, none.Expenses)
	assert.Empty(t, none.Incomes)
}
