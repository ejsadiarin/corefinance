package recurring

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/category"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func TestRecurringService_IncomeRulesCRUD(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	t.Run("CreateIncomeRule", func(t *testing.T) {
		rule, err := svc.CreateIncomeRule(ctx, userID, CreateIncomeRuleRequest{
			Amount:        5000,
			Currency:      "PHP",
			Description:   "Monthly Salary",
			RecurringType: "monthly",
			StartDate:     "2026-01-01",
		})
		require.NoError(t, err)
		require.NotEmpty(t, rule.ID)
		assert.Equal(t, userID, rule.UserID)
		assert.Equal(t, "Monthly Salary", rule.Description)
		assert.Equal(t, "monthly", rule.RecurringType)
		assert.Equal(t, "PHP", rule.Currency)
	})

	t.Run("GetIncomeRule", func(t *testing.T) {
		created, err := svc.CreateIncomeRule(ctx, userID, CreateIncomeRuleRequest{
			Amount:        3000,
			Currency:      "PHP",
			Description:   "Freelance Income",
			RecurringType: "monthly",
			StartDate:     "2026-02-01",
		})
		require.NoError(t, err)

		got, err := svc.GetIncomeRule(ctx, created.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Freelance Income", got.Description)
	})

	t.Run("GetIncomeRule_NotFound", func(t *testing.T) {
		_, err := svc.GetIncomeRule(ctx, uuid.New(), userID)
		require.Error(t, err)
	})

	t.Run("ListIncomeRules", func(t *testing.T) {
		userID2 := uuid.New()
		for i := 0; i < 3; i++ {
			_, err := svc.CreateIncomeRule(ctx, userID2, CreateIncomeRuleRequest{
				Amount:        float64(1000 * (i + 1)),
				Currency:      "PHP",
				Description:   "Income Rule",
				RecurringType: "monthly",
				StartDate:     "2026-03-01",
			})
			require.NoError(t, err)
		}

		rules, err := svc.ListIncomeRules(ctx, userID2)
		require.NoError(t, err)
		assert.Len(t, rules, 3)
	})

	t.Run("UpdateIncomeRule", func(t *testing.T) {
		created, err := svc.CreateIncomeRule(ctx, userID, CreateIncomeRuleRequest{
			Amount:        1000,
			Currency:      "PHP",
			Description:   "Original",
			RecurringType: "monthly",
			StartDate:     "2026-04-01",
		})
		require.NoError(t, err)

		updated, err := svc.UpdateIncomeRule(ctx, created.ID, userID, UpdateIncomeRuleRequest{
			Amount:        2000,
			Currency:      "USD",
			Description:   "Updated Rule",
			RecurringType: "weekly",
			StartDate:     "2026-04-01",
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Rule", updated.Description)
		assert.Equal(t, "USD", updated.Currency)
		assert.Equal(t, "weekly", updated.RecurringType)
	})

	t.Run("DeleteIncomeRule", func(t *testing.T) {
		created, err := svc.CreateIncomeRule(ctx, userID, CreateIncomeRuleRequest{
			Amount:        500,
			Currency:      "PHP",
			Description:   "To Delete",
			RecurringType: "monthly",
			StartDate:     "2026-05-01",
		})
		require.NoError(t, err)

		err = svc.DeleteIncomeRule(ctx, created.ID, userID)
		require.NoError(t, err)

		_, err = svc.GetIncomeRule(ctx, created.ID, userID)
		require.Error(t, err)
	})

	t.Run("DeleteIncomeRule_NotFound", func(t *testing.T) {
		err := svc.DeleteIncomeRule(ctx, uuid.New(), userID)
		require.Error(t, err)
	})
}

func TestRecurringService_ExpenseRulesCRUD(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()
	catSvc := category.NewService(pool)

	t.Run("CreateExpenseRule", func(t *testing.T) {
		cat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Food"})
		require.NoError(t, err)

		rule, err := svc.CreateExpenseRule(ctx, userID, CreateExpenseRuleRequest{
			Amount:        500,
			Currency:      "PHP",
			Description:   "Weekly Groceries",
			CategoryID:    ptrString(cat.ID.String()),
			RecurringType: "weekly",
			StartDate:     "2026-01-01",
			Priority:      "need",
		})
		require.NoError(t, err)
		require.NotEmpty(t, rule.ID)
		assert.Equal(t, "Weekly Groceries", rule.Description)
		assert.Equal(t, "weekly", rule.RecurringType)
		assert.Equal(t, "need", rule.Priority)
	})

	t.Run("GetExpenseRule", func(t *testing.T) {
		cat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Utilities"})
		require.NoError(t, err)

		created, err := svc.CreateExpenseRule(ctx, userID, CreateExpenseRuleRequest{
			Amount:        1000,
			Currency:      "PHP",
			Description:   "Monthly Utilities",
			CategoryID:    ptrString(cat.ID.String()),
			RecurringType: "monthly",
			StartDate:     "2026-02-01",
			Priority:      "need",
		})
		require.NoError(t, err)

		got, err := svc.GetExpenseRule(ctx, created.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Monthly Utilities", got.Description)
	})

	t.Run("GetExpenseRule_NotFound", func(t *testing.T) {
		_, err := svc.GetExpenseRule(ctx, uuid.New(), userID)
		require.Error(t, err)
	})

	t.Run("ListExpenseRules", func(t *testing.T) {
		userID2 := uuid.New()
		for i := 0; i < 3; i++ {
			_, err := svc.CreateExpenseRule(ctx, userID2, CreateExpenseRuleRequest{
				Amount:        float64(100 * (i + 1)),
				Currency:      "PHP",
				Description:   "Expense Rule",
				RecurringType: "monthly",
				StartDate:     "2026-03-01",
				Priority:      "want",
			})
			require.NoError(t, err)
		}

		rules, err := svc.ListExpenseRules(ctx, userID2)
		require.NoError(t, err)
		assert.Len(t, rules, 3)
	})

	t.Run("UpdateExpenseRule", func(t *testing.T) {
		cat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Entertainment"})
		require.NoError(t, err)

		created, err := svc.CreateExpenseRule(ctx, userID, CreateExpenseRuleRequest{
			Amount:        200,
			Currency:      "PHP",
			Description:   "Original",
			CategoryID:    ptrString(cat.ID.String()),
			RecurringType: "monthly",
			StartDate:     "2026-04-01",
			Priority:      "want",
		})
		require.NoError(t, err)

		updated, err := svc.UpdateExpenseRule(ctx, created.ID, userID, UpdateExpenseRuleRequest{
			Amount:        300,
			Currency:      "USD",
			Description:   "Updated Expense Rule",
			CategoryID:    ptrString(cat.ID.String()),
			RecurringType: "weekly",
			StartDate:     "2026-04-01",
			Priority:      "need",
			IsActive:      true,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Expense Rule", updated.Description)
		assert.Equal(t, "USD", updated.Currency)
		assert.Equal(t, "weekly", updated.RecurringType)
		assert.Equal(t, "need", updated.Priority)
	})

	t.Run("DeleteExpenseRule", func(t *testing.T) {
		created, err := svc.CreateExpenseRule(ctx, userID, CreateExpenseRuleRequest{
			Amount:        150,
			Currency:      "PHP",
			Description:   "To Delete",
			RecurringType: "monthly",
			StartDate:     "2026-05-01",
			Priority:      "want",
		})
		require.NoError(t, err)

		err = svc.DeleteExpenseRule(ctx, created.ID, userID)
		require.NoError(t, err)

		_, err = svc.GetExpenseRule(ctx, created.ID, userID)
		require.Error(t, err)
	})

	t.Run("DeleteExpenseRule_NotFound", func(t *testing.T) {
		err := svc.DeleteExpenseRule(ctx, uuid.New(), userID)
		require.Error(t, err)
	})
}

func TestRecurringService_UserIsolation(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	user1 := uuid.New()
	user2 := uuid.New()

	_, err := svc.CreateIncomeRule(ctx, user1, CreateIncomeRuleRequest{
		Amount:        5000,
		Currency:      "PHP",
		Description:   "User1 Rule",
		RecurringType: "monthly",
		StartDate:     "2026-01-01",
	})
	require.NoError(t, err)

	_, err = svc.CreateIncomeRule(ctx, user2, CreateIncomeRuleRequest{
		Amount:        3000,
		Currency:      "PHP",
		Description:   "User2 Rule",
		RecurringType: "monthly",
		StartDate:     "2026-01-01",
	})
	require.NoError(t, err)

	rules1, err := svc.ListIncomeRules(ctx, user1)
	require.NoError(t, err)
	assert.Len(t, rules1, 1)
	assert.Equal(t, "User1 Rule", rules1[0].Description)

	rules2, err := svc.ListIncomeRules(ctx, user2)
	require.NoError(t, err)
	assert.Len(t, rules2, 1)
	assert.Equal(t, "User2 Rule", rules2[0].Description)
}

func ptrString(s string) *string {
	return &s
}
