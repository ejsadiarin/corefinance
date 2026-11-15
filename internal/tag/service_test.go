package tag

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/mock"
)

func TestTagService_Create(t *testing.T) {
	userID := uuid.New()
	tagID := uuid.New()
	color := "#FF5733"

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			CreateTagFn: func(ctx context.Context, arg db.CreateTagParams) (db.Tag, error) {
				assert.Equal(t, userID, arg.UserID)
				assert.Equal(t, "Urgent", arg.Name)
				assert.Equal(t, pgtype.Text{String: color, Valid: true}, arg.Color)
				return db.Tag{
					ID:     tagID,
					UserID: userID,
					Name:   "Urgent",
					Color:  pgtype.Text{String: color, Valid: true},
				}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.Create(context.Background(), userID, CreateRequest{
			Name:  "Urgent",
			Color: &color,
		})
		require.NoError(t, err)
		assert.Equal(t, tagID, result.ID)
		assert.Equal(t, "Urgent", result.Name)
	})

	t.Run("nil color", func(t *testing.T) {
		m := &mock.MockQuerier{
			CreateTagFn: func(ctx context.Context, arg db.CreateTagParams) (db.Tag, error) {
				assert.Equal(t, pgtype.Text{Valid: false}, arg.Color)
				return db.Tag{ID: tagID, Name: "NoColor"}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.Create(context.Background(), userID, CreateRequest{Name: "NoColor"})
		require.NoError(t, err)
		assert.Equal(t, "NoColor", result.Name)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			CreateTagFn: func(ctx context.Context, arg db.CreateTagParams) (db.Tag, error) {
				return db.Tag{}, errors.New("db error")
			},
		}
		svc := NewService(m)
		_, err := svc.Create(context.Background(), userID, CreateRequest{Name: "X"})
		require.Error(t, err)
	})
}

func TestTagService_List(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
				assert.Equal(t, userID, uid)
				return []db.Tag{
					{ID: uuid.New(), Name: "Tag1"},
					{ID: uuid.New(), Name: "Tag2"},
				}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.List(context.Background(), userID)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("empty", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
				return []db.Tag{}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.List(context.Background(), userID)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListTagsFn: func(ctx context.Context, uid uuid.UUID) ([]db.Tag, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewService(m)
		_, err := svc.List(context.Background(), userID)
		require.Error(t, err)
	})
}

func TestTagService_Get(t *testing.T) {
	userID := uuid.New()
	tagID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetTagFn: func(ctx context.Context, arg db.GetTagParams) (db.Tag, error) {
				assert.Equal(t, tagID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				return db.Tag{ID: tagID, Name: "Urgent"}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.Get(context.Background(), tagID, userID)
		require.NoError(t, err)
		assert.Equal(t, "Urgent", result.Name)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetTagFn: func(ctx context.Context, arg db.GetTagParams) (db.Tag, error) {
				return db.Tag{}, errors.New("not found")
			},
		}
		svc := NewService(m)
		_, err := svc.Get(context.Background(), tagID, userID)
		require.Error(t, err)
	})
}

func TestTagService_Update(t *testing.T) {
	userID := uuid.New()
	tagID := uuid.New()
	color := "#33FF57"

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			UpdateTagFn: func(ctx context.Context, arg db.UpdateTagParams) (db.Tag, error) {
				assert.Equal(t, tagID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				assert.Equal(t, "Updated Tag", arg.Name)
				assert.Equal(t, pgtype.Text{String: color, Valid: true}, arg.Color)
				return db.Tag{
					ID:   tagID,
					Name: "Updated Tag",
					Color: pgtype.Text{String: color, Valid: true},
				}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.Update(context.Background(), tagID, userID, UpdateRequest{
			Name:  "Updated Tag",
			Color: &color,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Tag", result.Name)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			UpdateTagFn: func(ctx context.Context, arg db.UpdateTagParams) (db.Tag, error) {
				return db.Tag{}, errors.New("db error")
			},
		}
		svc := NewService(m)
		_, err := svc.Update(context.Background(), tagID, userID, UpdateRequest{Name: "X"})
		require.Error(t, err)
	})
}

func TestTagService_Delete(t *testing.T) {
	userID := uuid.New()
	tagID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			DeleteTagFn: func(ctx context.Context, arg db.DeleteTagParams) error {
				assert.Equal(t, tagID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				return nil
			},
		}
		svc := NewService(m)
		err := svc.Delete(context.Background(), tagID, userID)
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			DeleteTagFn: func(ctx context.Context, arg db.DeleteTagParams) error {
				return errors.New("not found")
			},
		}
		svc := NewService(m)
		err := svc.Delete(context.Background(), tagID, userID)
		require.Error(t, err)
	})
}

func TestTagService_AddTagToExpense(t *testing.T) {
	expenseID := uuid.New()
	tagID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			AddTagToExpenseFn: func(ctx context.Context, arg db.AddTagToExpenseParams) error {
				assert.Equal(t, expenseID, arg.ExpenseID)
				assert.Equal(t, tagID, arg.TagID)
				return nil
			},
		}
		svc := NewService(m)
		err := svc.AddTagToExpense(context.Background(), expenseID, tagID)
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			AddTagToExpenseFn: func(ctx context.Context, arg db.AddTagToExpenseParams) error {
				return errors.New("duplicate")
			},
		}
		svc := NewService(m)
		err := svc.AddTagToExpense(context.Background(), expenseID, tagID)
		require.Error(t, err)
	})
}

func TestTagService_RemoveTagFromExpense(t *testing.T) {
	expenseID := uuid.New()
	tagID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			RemoveTagFromExpenseFn: func(ctx context.Context, arg db.RemoveTagFromExpenseParams) error {
				assert.Equal(t, expenseID, arg.ExpenseID)
				assert.Equal(t, tagID, arg.TagID)
				return nil
			},
		}
		svc := NewService(m)
		err := svc.RemoveTagFromExpense(context.Background(), expenseID, tagID)
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			RemoveTagFromExpenseFn: func(ctx context.Context, arg db.RemoveTagFromExpenseParams) error {
				return errors.New("not found")
			},
		}
		svc := NewService(m)
		err := svc.RemoveTagFromExpense(context.Background(), expenseID, tagID)
		require.Error(t, err)
	})
}

func TestTagService_GetTagsByExpenseID(t *testing.T) {
	expenseID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetTagsByExpenseIDFn: func(ctx context.Context, eid uuid.UUID) ([]db.Tag, error) {
				assert.Equal(t, expenseID, eid)
				return []db.Tag{
					{ID: uuid.New(), Name: "Tag1"},
					{ID: uuid.New(), Name: "Tag2"},
				}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.GetTagsByExpenseID(context.Background(), expenseID)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("empty", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetTagsByExpenseIDFn: func(ctx context.Context, eid uuid.UUID) ([]db.Tag, error) {
				return []db.Tag{}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.GetTagsByExpenseID(context.Background(), expenseID)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetTagsByExpenseIDFn: func(ctx context.Context, eid uuid.UUID) ([]db.Tag, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewService(m)
		_, err := svc.GetTagsByExpenseID(context.Background(), expenseID)
		require.Error(t, err)
	})
}

func TestTagService_AddTagToIncome(t *testing.T) {
	incomeID := uuid.New()
	tagID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			AddTagToIncomeFn: func(ctx context.Context, arg db.AddTagToIncomeParams) error {
				assert.Equal(t, incomeID, arg.IncomeID)
				assert.Equal(t, tagID, arg.TagID)
				return nil
			},
		}
		svc := NewService(m)
		err := svc.AddTagToIncome(context.Background(), incomeID, tagID)
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			AddTagToIncomeFn: func(ctx context.Context, arg db.AddTagToIncomeParams) error {
				return errors.New("duplicate")
			},
		}
		svc := NewService(m)
		err := svc.AddTagToIncome(context.Background(), incomeID, tagID)
		require.Error(t, err)
	})
}

func TestTagService_RemoveTagFromIncome(t *testing.T) {
	incomeID := uuid.New()
	tagID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			RemoveTagFromIncomeFn: func(ctx context.Context, arg db.RemoveTagFromIncomeParams) error {
				assert.Equal(t, incomeID, arg.IncomeID)
				assert.Equal(t, tagID, arg.TagID)
				return nil
			},
		}
		svc := NewService(m)
		err := svc.RemoveTagFromIncome(context.Background(), incomeID, tagID)
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			RemoveTagFromIncomeFn: func(ctx context.Context, arg db.RemoveTagFromIncomeParams) error {
				return errors.New("not found")
			},
		}
		svc := NewService(m)
		err := svc.RemoveTagFromIncome(context.Background(), incomeID, tagID)
		require.Error(t, err)
	})
}

func TestTagService_GetTagsByIncomeID(t *testing.T) {
	incomeID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetTagsByIncomeIDFn: func(ctx context.Context, iid uuid.UUID) ([]db.Tag, error) {
				assert.Equal(t, incomeID, iid)
				return []db.Tag{
					{ID: uuid.New(), Name: "IncomeTag1"},
					{ID: uuid.New(), Name: "IncomeTag2"},
					{ID: uuid.New(), Name: "IncomeTag3"},
				}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.GetTagsByIncomeID(context.Background(), incomeID)
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("empty", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetTagsByIncomeIDFn: func(ctx context.Context, iid uuid.UUID) ([]db.Tag, error) {
				return []db.Tag{}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.GetTagsByIncomeID(context.Background(), incomeID)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetTagsByIncomeIDFn: func(ctx context.Context, iid uuid.UUID) ([]db.Tag, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewService(m)
		_, err := svc.GetTagsByIncomeID(context.Background(), incomeID)
		require.Error(t, err)
	})
}
