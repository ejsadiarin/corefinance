package recurring

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

// Expense and income rules live in one rules table (#11); the rule endpoints
// keep their JSON and each sees only its own kind (#10).
func TestRulesShareOneTable_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	s := NewServiceWithPool(db.New(pool), pool)
	user := uuid.New()

	exp, err := s.CreateExpenseRule(ctx, user, CreateExpenseRuleRequest{
		Amount: 85, Currency: "PHP", Description: "spotify", RecurringType: "monthly", StartDate: "2026-02-23", Priority: "need",
	})
	require.NoError(t, err)
	inc, err := s.CreateIncomeRule(ctx, user, CreateIncomeRuleRequest{
		Amount: 500, Currency: "PHP", Description: "allowance", RecurringType: "weekly", StartDate: "2026-11-19",
	})
	require.NoError(t, err)

	kindOf := func(id uuid.UUID) string {
		var kind string
		require.NoError(t, pool.QueryRow(ctx, `SELECT kind FROM rules WHERE id = $1`, id).Scan(&kind))
		return kind
	}
	assert.Equal(t, "expense", kindOf(exp.ID))
	assert.Equal(t, "income", kindOf(inc.ID))

	// Each endpoint still sees only its own kind.
	expenses, err := s.ListExpenseRules(ctx, user)
	require.NoError(t, err)
	require.Len(t, expenses, 1)
	incomes, err := s.ListIncomeRules(ctx, user)
	require.NoError(t, err)
	require.Len(t, incomes, 1)

	for _, v := range []any{exp, inc, expenses[0], incomes[0]} {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		assert.NotContains(t, string(b), `"kind"`, "rule JSON is unchanged")
		assert.NotContains(t, string(b), `"next_due_at"`, "rule JSON is unchanged")
	}

	// A rule of the other kind is not found, and deleting it through the
	// wrong endpoint leaves it in place.
	_, err = s.GetExpenseRule(ctx, inc.ID, user)
	assert.Error(t, err, "an income rule is not an expense rule")
	_, err = s.GetIncomeRule(ctx, exp.ID, user)
	assert.Error(t, err, "an expense rule is not an income rule")
	require.NoError(t, s.DeleteExpenseRule(ctx, inc.ID, user))
	require.NoError(t, s.DeleteIncomeRule(ctx, exp.ID, user))
	assert.Equal(t, "expense", kindOf(exp.ID))
	assert.Equal(t, "income", kindOf(inc.ID))
}
