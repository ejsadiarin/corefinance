package category

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func TestCategoryService_ExpenseCRUD(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	t.Run("CreateExpense", func(t *testing.T) {
		cat, err := svc.CreateExpense(ctx, userID, CreateRequest{
			Name:  "Food",
			Color: testutil.StrPtr("#FF0000"),
			Icon:  testutil.StrPtr("food-icon"),
		})
		require.NoError(t, err)
		require.NotEmpty(t, cat.ID)
		assert.Equal(t, "Food", cat.Name)
		assert.Equal(t, "#FF0000", cat.Color.String)
		assert.Equal(t, "food-icon", cat.Icon.String)
		assert.True(t, cat.IsActive)
	})

	t.Run("GetExpense", func(t *testing.T) {
		created, err := svc.CreateExpense(ctx, userID, CreateRequest{
			Name: "Transport",
		})
		require.NoError(t, err)

		got, err := svc.GetExpense(ctx, created.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Transport", got.Name)
	})

	t.Run("GetExpense_NotFound", func(t *testing.T) {
		_, err := svc.GetExpense(ctx, uuid.New(), userID)
		require.Error(t, err)
	})

	t.Run("ListExpense", func(t *testing.T) {
		userID2 := uuid.New()
		for _, name := range []string{"Cat1", "Cat2", "Cat3"} {
			_, err := svc.CreateExpense(ctx, userID2, CreateRequest{Name: name})
			require.NoError(t, err)
		}

		cats, err := svc.ListExpense(ctx, userID2)
		require.NoError(t, err)
		assert.Len(t, cats, 3)
	})

	t.Run("UpdateExpense", func(t *testing.T) {
		created, err := svc.CreateExpense(ctx, userID, CreateRequest{
			Name: "Original",
		})
		require.NoError(t, err)

		updated, err := svc.UpdateExpense(ctx, created.ID, userID, UpdateRequest{
			Name:  "Updated",
			Color: testutil.StrPtr("#00FF00"),
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated", updated.Name)
		assert.Equal(t, "#00FF00", updated.Color.String)
	})

	t.Run("DeleteExpense", func(t *testing.T) {
		created, err := svc.CreateExpense(ctx, userID, CreateRequest{
			Name: "To Delete",
		})
		require.NoError(t, err)

		err = svc.DeleteExpense(ctx, created.ID, userID)
		require.NoError(t, err)

		_, err = svc.GetExpense(ctx, created.ID, userID)
		require.Error(t, err)
	})

	t.Run("DeleteExpense_NotFound", func(t *testing.T) {
		err := svc.DeleteExpense(ctx, uuid.New(), userID)
		require.Error(t, err)
	})
}

func TestCategoryService_IncomeCRUD(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	t.Run("CreateIncome", func(t *testing.T) {
		cat, err := svc.CreateIncome(ctx, userID, CreateRequest{
			Name:  "Salary",
			Color: testutil.StrPtr("#0000FF"),
			Icon:  testutil.StrPtr("salary-icon"),
		})
		require.NoError(t, err)
		require.NotEmpty(t, cat.ID)
		assert.Equal(t, "Salary", cat.Name)
		assert.Equal(t, "#0000FF", cat.Color.String)
		assert.True(t, cat.IsActive)
	})

	t.Run("GetIncome", func(t *testing.T) {
		created, err := svc.CreateIncome(ctx, userID, CreateRequest{
			Name: "Freelance",
		})
		require.NoError(t, err)

		got, err := svc.GetIncome(ctx, created.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Freelance", got.Name)
	})

	t.Run("GetIncome_NotFound", func(t *testing.T) {
		_, err := svc.GetIncome(ctx, uuid.New(), userID)
		require.Error(t, err)
	})

	t.Run("ListIncome", func(t *testing.T) {
		userID2 := uuid.New()
		for _, name := range []string{"Inc1", "Inc2"} {
			_, err := svc.CreateIncome(ctx, userID2, CreateRequest{Name: name})
			require.NoError(t, err)
		}

		cats, err := svc.ListIncome(ctx, userID2)
		require.NoError(t, err)
		assert.Len(t, cats, 2)
	})

	t.Run("UpdateIncome", func(t *testing.T) {
		created, err := svc.CreateIncome(ctx, userID, CreateRequest{
			Name: "Original",
		})
		require.NoError(t, err)

		updated, err := svc.UpdateIncome(ctx, created.ID, userID, UpdateRequest{
			Name:  "Updated Income",
			Color: testutil.StrPtr("#FFFF00"),
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Income", updated.Name)
		assert.Equal(t, "#FFFF00", updated.Color.String)
	})

	t.Run("DeleteIncome", func(t *testing.T) {
		created, err := svc.CreateIncome(ctx, userID, CreateRequest{
			Name: "To Delete",
		})
		require.NoError(t, err)

		err = svc.DeleteIncome(ctx, created.ID, userID)
		require.NoError(t, err)

		_, err = svc.GetIncome(ctx, created.ID, userID)
		require.Error(t, err)
	})

	t.Run("DeleteIncome_NotFound", func(t *testing.T) {
		err := svc.DeleteIncome(ctx, uuid.New(), userID)
		require.Error(t, err)
	})
}

func TestCategoryService_UserIsolation(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	user1 := uuid.New()
	user2 := uuid.New()

	_, err := svc.CreateExpense(ctx, user1, CreateRequest{Name: "User1 Cat"})
	require.NoError(t, err)

	_, err = svc.CreateExpense(ctx, user2, CreateRequest{Name: "User2 Cat"})
	require.NoError(t, err)

	cats1, err := svc.ListExpense(ctx, user1)
	require.NoError(t, err)
	assert.Len(t, cats1, 1)
	assert.Equal(t, "User1 Cat", cats1[0].Name)

	cats2, err := svc.ListExpense(ctx, user2)
	require.NoError(t, err)
	assert.Len(t, cats2, 1)
	assert.Equal(t, "User2 Cat", cats2[0].Name)
}
