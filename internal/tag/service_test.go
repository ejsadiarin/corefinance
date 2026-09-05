package tag

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/category"
	"github.com/ejsadiarin/corefinance/internal/expense"
	"github.com/ejsadiarin/corefinance/internal/income"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func TestTagService_CRUD(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	t.Run("Create", func(t *testing.T) {
		tag, err := svc.Create(ctx, userID, CreateRequest{
			Name:  "Food",
			Color: testutil.StrPtr("#FF5733"),
		})
		require.NoError(t, err)
		require.NotEmpty(t, tag.ID)
		assert.Equal(t, "Food", tag.Name)
		assert.Equal(t, "#FF5733", tag.Color.String)
	})

	t.Run("Get", func(t *testing.T) {
		created, err := svc.Create(ctx, userID, CreateRequest{
			Name: "Transport",
		})
		require.NoError(t, err)

		got, err := svc.Get(ctx, created.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "Transport", got.Name)
	})

	t.Run("Get_NotFound", func(t *testing.T) {
		_, err := svc.Get(ctx, uuid.New(), userID)
		require.Error(t, err)
	})

	t.Run("List", func(t *testing.T) {
		userID2 := uuid.New()
		for _, name := range []string{"Tag1", "Tag2", "Tag3"} {
			_, err := svc.Create(ctx, userID2, CreateRequest{Name: name})
			require.NoError(t, err)
		}

		tags, err := svc.List(ctx, userID2)
		require.NoError(t, err)
		assert.Len(t, tags, 3)
	})

	t.Run("Update", func(t *testing.T) {
		created, err := svc.Create(ctx, userID, CreateRequest{
			Name: "Original",
		})
		require.NoError(t, err)

		updated, err := svc.Update(ctx, created.ID, userID, UpdateRequest{
			Name:  "Updated Tag",
			Color: testutil.StrPtr("#00FF00"),
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Tag", updated.Name)
		assert.Equal(t, "#00FF00", updated.Color.String)
	})

	t.Run("Delete", func(t *testing.T) {
		created, err := svc.Create(ctx, userID, CreateRequest{
			Name: "To Delete",
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

func TestTagService_ExpenseAssociation(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	expenseSvc := expense.NewService(pool)
	catSvc := category.NewService(pool)

	// Create a tag
	tag, err := svc.Create(ctx, userID, CreateRequest{Name: "Food"})
	require.NoError(t, err)

	// Create a category
	cat, err := catSvc.CreateExpense(ctx, userID, category.CreateRequest{Name: "Meals"})
	require.NoError(t, err)

	// Create an expense
	exp, err := expenseSvc.Create(ctx, userID, expense.CreateRequest{
		Amount:      100,
		Currency:    "PHP",
		Description: "Lunch",
		CategoryID:  ptrString(cat.ID.String()),
		ExpenseDate: "2026-01-15",
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)

	t.Run("AddTagToExpense", func(t *testing.T) {
		err := svc.AddTagToExpense(ctx, exp.ID, tag.ID)
		require.NoError(t, err)
	})

	t.Run("GetTagsByExpenseID", func(t *testing.T) {
		tags, err := svc.GetTagsByExpenseID(ctx, exp.ID)
		require.NoError(t, err)
		assert.Len(t, tags, 1)
		assert.Equal(t, "Food", tags[0].Name)
	})

	t.Run("RemoveTagFromExpense", func(t *testing.T) {
		err := svc.RemoveTagFromExpense(ctx, exp.ID, tag.ID)
		require.NoError(t, err)

		tags, err := svc.GetTagsByExpenseID(ctx, exp.ID)
		require.NoError(t, err)
		assert.Len(t, tags, 0)
	})
}

func TestTagService_IncomeAssociation(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(pool)
	ctx := context.Background()
	userID := testutil.TestUserIDUUID()

	incomeSvc := income.NewService(pool)
	catSvc := category.NewService(pool)

	// Create a tag
	tag, err := svc.Create(ctx, userID, CreateRequest{Name: "Salary"})
	require.NoError(t, err)

	// Create a category
	cat, err := catSvc.CreateIncome(ctx, userID, category.CreateRequest{Name: "Primary"})
	require.NoError(t, err)

	// Create an income
	inc, err := incomeSvc.Create(ctx, userID, income.CreateRequest{
		Amount:      5000,
		Currency:    "PHP",
		Description: "Monthly Salary",
		CategoryID:  ptrString(cat.ID.String()),
		Date:        "2026-01-15",
		Priority:    "need",
		Status:      "posted",
	})
	require.NoError(t, err)

	t.Run("AddTagToIncome", func(t *testing.T) {
		err := svc.AddTagToIncome(ctx, inc.ID, tag.ID)
		require.NoError(t, err)
	})

	t.Run("GetTagsByIncomeID", func(t *testing.T) {
		tags, err := svc.GetTagsByIncomeID(ctx, inc.ID)
		require.NoError(t, err)
		assert.Len(t, tags, 1)
		assert.Equal(t, "Salary", tags[0].Name)
	})

	t.Run("RemoveTagFromIncome", func(t *testing.T) {
		err := svc.RemoveTagFromIncome(ctx, inc.ID, tag.ID)
		require.NoError(t, err)

		tags, err := svc.GetTagsByIncomeID(ctx, inc.ID)
		require.NoError(t, err)
		assert.Len(t, tags, 0)
	})
}

func ptrString(s string) *string {
	return &s
}
