package expense

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/category"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func TestExpenseService_CRUD(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()
	catSvc := category.NewService(pool)

	t.Run("Create", func(t *testing.T) {
		cat := createCategory(t, ctx, catSvc, userID)

		req := CreateRequest{
			CategoryID:    &cat,
			Amount:        1500.50,
			Currency:      "PHP",
			Description:   "Test Expense",
			Notes:         testutil.StrPtr("Some notes"),
			ExpenseDate:   "2026-01-15",
			RecurringType: "one-time",
			Priority:      "want",
			Status:        "posted",
			IsDebt:        false,
		}

		expense, err := svc.Create(ctx, userID, req)
		require.NoError(t, err)
		require.NotEmpty(t, expense.ID)
		assert.Equal(t, userID, expense.UserID)
		assert.Equal(t, "Test Expense", expense.Description)
		assert.Equal(t, "PHP", expense.Currency)
		assert.Equal(t, "posted", expense.Status)
		assert.Equal(t, "want", expense.Priority)
		assert.False(t, expense.IsDebt)
	})

	t.Run("Get", func(t *testing.T) {
		cat := createCategory(t, ctx, catSvc, userID)

		created, err := svc.Create(ctx, userID, CreateRequest{
			CategoryID:    &cat,
			Amount:        500,
			Currency:      "PHP",
			Description:   "Get Test Expense",
			ExpenseDate:   "2026-02-01",
			RecurringType: "one-time",
			Priority:      "need",
			Status:        "posted",
			IsDebt:        false,
		})
		require.NoError(t, err)

		got, err := svc.Get(ctx, created.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Get Test Expense", got.Description)
		assert.Equal(t, "need", got.Priority)
	})

	t.Run("Get_NotFound", func(t *testing.T) {
		_, err := svc.Get(ctx, uuid.New(), userID)
		require.Error(t, err)
	})

	t.Run("List", func(t *testing.T) {
		userID2 := uuid.New()
		cat := createCategoryForUser(t, ctx, catSvc, userID2)

		for i := 0; i < 3; i++ {
			_, err := svc.Create(ctx, userID2, CreateRequest{
				CategoryID:    &cat,
				Amount:        float64(100 * (i + 1)),
				Currency:      "PHP",
				Description:   "List Expense",
				ExpenseDate:   "2026-03-01",
				RecurringType: "one-time",
				Priority:      "want",
				Status:        "posted",
				IsDebt:        false,
			})
			require.NoError(t, err)
		}

		results, err := svc.List(ctx, userID2, ListParams{Page: 1, PageSize: 50})
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("Update", func(t *testing.T) {
		cat := createCategory(t, ctx, catSvc, userID)

		created, err := svc.Create(ctx, userID, CreateRequest{
			CategoryID:    &cat,
			Amount:        100,
			Currency:      "PHP",
			Description:   "Original Description",
			ExpenseDate:   "2026-04-01",
			RecurringType: "one-time",
			Priority:      "want",
			Status:        "pending",
			IsDebt:        false,
		})
		require.NoError(t, err)

		updated, err := svc.Update(ctx, created.ID, userID, UpdateRequest{
			CategoryID:    &cat,
			Amount:        200,
			Currency:      "USD",
			Description:   "Updated Description",
			ExpenseDate:   "2026-04-02",
			RecurringType: "one-time",
			Priority:      "need",
			Status:        "posted",
			IsDebt:        true,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Description", updated.Description)
		assert.Equal(t, "USD", updated.Currency)
		assert.Equal(t, "need", updated.Priority)
		assert.Equal(t, "posted", updated.Status)
		assert.True(t, updated.IsDebt)
	})

	t.Run("Delete", func(t *testing.T) {
		cat := createCategory(t, ctx, catSvc, userID)

		created, err := svc.Create(ctx, userID, CreateRequest{
			CategoryID:    &cat,
			Amount:        300,
			Currency:      "PHP",
			Description:   "Delete Expense",
			ExpenseDate:   "2026-05-01",
			RecurringType: "one-time",
			Priority:      "want",
			Status:        "posted",
			IsDebt:        false,
		})
		require.NoError(t, err)

		err = svc.Delete(ctx, created.ID, userID)
		require.NoError(t, err)

		_, err = svc.Get(ctx, created.ID, userID)
		require.Error(t, err)
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		err := svc.Delete(ctx, uuid.New(), userID)
		require.Error(t, err)
	})
}

func createCategory(t *testing.T, ctx context.Context, svc *category.Service, userID uuid.UUID) string {
	t.Helper()
	cat, err := svc.CreateExpense(ctx, userID, category.CreateRequest{
		Name: "Test Expense Category",
	})
	require.NoError(t, err)
	return cat.ID.String()
}

func createCategoryForUser(t *testing.T, ctx context.Context, svc *category.Service, userID uuid.UUID) string {
	t.Helper()
	cat, err := svc.CreateExpense(ctx, userID, category.CreateRequest{
		Name: "Test Expense Category",
	})
	require.NoError(t, err)
	return cat.ID.String()
}
