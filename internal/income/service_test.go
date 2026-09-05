package income

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/category"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func TestIncomeService_CRUD(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()
	catSvc := category.NewService(pool)

	t.Run("Create", func(t *testing.T) {
		cat := createCategory(t, ctx, catSvc, userID)

		req := CreateRequest{
			CategoryID:    &cat,
			Amount:        5000.75,
			Currency:      "PHP",
			Description:   "Test Income",
			Notes:         testutil.StrPtr("Salary notes"),
			Date:          "2026-01-15",
			RecurringType: "monthly",
			Priority:      "need",
			Status:        "posted",
			StartDate:     testutil.StrPtr("2026-01-01"),
		}

		income, err := svc.Create(ctx, userID, req)
		require.NoError(t, err)
		require.NotEmpty(t, income.ID)
		assert.Equal(t, userID, income.UserID)
		assert.Equal(t, "Test Income", income.Description)
		assert.Equal(t, "PHP", income.Currency)
		assert.Equal(t, "posted", income.Status)
		assert.Equal(t, "monthly", income.RecurringType)
	})

	t.Run("Get", func(t *testing.T) {
		cat := createCategory(t, ctx, catSvc, userID)

		created, err := svc.Create(ctx, userID, CreateRequest{
			CategoryID:    &cat,
			Amount:        3000,
			Currency:      "PHP",
			Description:   "Get Test Income",
			Date:          "2026-02-01",
			RecurringType: "one-time",
			Priority:      "want",
			Status:        "posted",
		})
		require.NoError(t, err)

		got, err := svc.Get(ctx, created.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Get Test Income", got.Description)
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
				Amount:        float64(1000 * (i + 1)),
				Currency:      "PHP",
				Description:   "List Income",
				Date:          "2026-03-01",
				RecurringType: "one-time",
				Priority:      "want",
				Status:        "posted",
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
			Amount:        1000,
			Currency:      "PHP",
			Description:   "Original Income",
			Date:          "2026-04-01",
			RecurringType: "one-time",
			Priority:      "want",
			Status:        "pending",
		})
		require.NoError(t, err)

		updated, err := svc.Update(ctx, created.ID, userID, UpdateRequest{
			CategoryID:    &cat,
			Amount:        2000,
			Currency:      "USD",
			Description:   "Updated Income",
			Date:          "2026-04-02",
			RecurringType: "monthly",
			Priority:      "need",
			Status:        "posted",
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Income", updated.Description)
		assert.Equal(t, "USD", updated.Currency)
		assert.Equal(t, "need", updated.Priority)
		assert.Equal(t, "posted", updated.Status)
	})

	t.Run("Delete", func(t *testing.T) {
		cat := createCategory(t, ctx, catSvc, userID)

		created, err := svc.Create(ctx, userID, CreateRequest{
			CategoryID:    &cat,
			Amount:        500,
			Currency:      "PHP",
			Description:   "Delete Income",
			Date:          "2026-05-01",
			RecurringType: "one-time",
			Priority:      "want",
			Status:        "posted",
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
	cat, err := svc.CreateIncome(ctx, userID, category.CreateRequest{
		Name: "Test Income Category",
	})
	require.NoError(t, err)
	return cat.ID.String()
}

func createCategoryForUser(t *testing.T, ctx context.Context, svc *category.Service, userID uuid.UUID) string {
	t.Helper()
	cat, err := svc.CreateIncome(ctx, userID, category.CreateRequest{
		Name: "Test Income Category",
	})
	require.NoError(t, err)
	return cat.ID.String()
}
