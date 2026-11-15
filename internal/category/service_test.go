package category

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

func TestCategoryService_CreateExpense(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()
	color := "#FF0000"
	icon := "food-icon"

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			CreateExpenseCategoryFn: func(ctx context.Context, arg db.CreateExpenseCategoryParams) (db.ExpenseCategory, error) {
				assert.Equal(t, userID, arg.UserID)
				assert.Equal(t, "Food", arg.Name)
				assert.Equal(t, pgtype.Text{String: color, Valid: true}, arg.Color)
				assert.Equal(t, pgtype.Text{String: icon, Valid: true}, arg.Icon)
				return db.ExpenseCategory{
					ID:     catID,
					UserID: userID,
					Name:   "Food",
					Color:  pgtype.Text{String: color, Valid: true},
					Icon:   pgtype.Text{String: icon, Valid: true},
				}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.CreateExpense(context.Background(), userID, CreateRequest{
			Name:  "Food",
			Color: &color,
			Icon:  &icon,
		})
		require.NoError(t, err)
		assert.Equal(t, catID, result.ID)
		assert.Equal(t, "Food", result.Name)
	})

	t.Run("nil optional fields", func(t *testing.T) {
		m := &mock.MockQuerier{
			CreateExpenseCategoryFn: func(ctx context.Context, arg db.CreateExpenseCategoryParams) (db.ExpenseCategory, error) {
				assert.Equal(t, pgtype.Text{Valid: false}, arg.Color)
				assert.Equal(t, pgtype.Text{Valid: false}, arg.Icon)
				return db.ExpenseCategory{
					ID:     catID,
					UserID: userID,
					Name:   "Bare",
				}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.CreateExpense(context.Background(), userID, CreateRequest{Name: "Bare"})
		require.NoError(t, err)
		assert.Equal(t, catID, result.ID)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			CreateExpenseCategoryFn: func(ctx context.Context, arg db.CreateExpenseCategoryParams) (db.ExpenseCategory, error) {
				return db.ExpenseCategory{}, errors.New("db error")
			},
		}
		svc := NewService(m)
		_, err := svc.CreateExpense(context.Background(), userID, CreateRequest{Name: "X"})
		require.Error(t, err)
		assert.Equal(t, "db error", err.Error())
	})
}

func TestCategoryService_ListExpense(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context, uid uuid.UUID) ([]db.ExpenseCategory, error) {
				assert.Equal(t, userID, uid)
				return []db.ExpenseCategory{
					{ID: uuid.New(), Name: "Food"},
					{ID: uuid.New(), Name: "Transport"},
				}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.ListExpense(context.Background(), userID)
		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("empty", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context, uid uuid.UUID) ([]db.ExpenseCategory, error) {
				return []db.ExpenseCategory{}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.ListExpense(context.Background(), userID)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListExpenseCategoriesFn: func(ctx context.Context, uid uuid.UUID) ([]db.ExpenseCategory, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewService(m)
		_, err := svc.ListExpense(context.Background(), userID)
		require.Error(t, err)
	})
}

func TestCategoryService_GetExpense(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetExpenseCategoryFn: func(ctx context.Context, arg db.GetExpenseCategoryParams) (db.ExpenseCategory, error) {
				assert.Equal(t, catID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				return db.ExpenseCategory{ID: catID, Name: "Food"}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.GetExpense(context.Background(), catID, userID)
		require.NoError(t, err)
		assert.Equal(t, "Food", result.Name)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetExpenseCategoryFn: func(ctx context.Context, arg db.GetExpenseCategoryParams) (db.ExpenseCategory, error) {
				return db.ExpenseCategory{}, errors.New("not found")
			},
		}
		svc := NewService(m)
		_, err := svc.GetExpense(context.Background(), catID, userID)
		require.Error(t, err)
	})
}

func TestCategoryService_UpdateExpense(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()
	color := "#00FF00"

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			UpdateExpenseCategoryFn: func(ctx context.Context, arg db.UpdateExpenseCategoryParams) (db.ExpenseCategory, error) {
				assert.Equal(t, catID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				assert.Equal(t, "Updated", arg.Name)
				assert.Equal(t, pgtype.Text{String: color, Valid: true}, arg.Color)
				return db.ExpenseCategory{
					ID:   catID,
					Name: "Updated",
					Color: pgtype.Text{String: color, Valid: true},
				}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.UpdateExpense(context.Background(), catID, userID, UpdateRequest{
			Name:  "Updated",
			Color: &color,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated", result.Name)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			UpdateExpenseCategoryFn: func(ctx context.Context, arg db.UpdateExpenseCategoryParams) (db.ExpenseCategory, error) {
				return db.ExpenseCategory{}, errors.New("db error")
			},
		}
		svc := NewService(m)
		_, err := svc.UpdateExpense(context.Background(), catID, userID, UpdateRequest{Name: "X"})
		require.Error(t, err)
	})
}

func TestCategoryService_DeleteExpense(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			DeleteExpenseCategoryFn: func(ctx context.Context, arg db.DeleteExpenseCategoryParams) error {
				assert.Equal(t, catID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				return nil
			},
		}
		svc := NewService(m)
		err := svc.DeleteExpense(context.Background(), catID, userID)
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			DeleteExpenseCategoryFn: func(ctx context.Context, arg db.DeleteExpenseCategoryParams) error {
				return errors.New("not found")
			},
		}
		svc := NewService(m)
		err := svc.DeleteExpense(context.Background(), catID, userID)
		require.Error(t, err)
	})
}

func TestCategoryService_CreateIncome(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()
	color := "#0000FF"

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			CreateIncomeCategoryFn: func(ctx context.Context, arg db.CreateIncomeCategoryParams) (db.IncomeCategory, error) {
				assert.Equal(t, userID, arg.UserID)
				assert.Equal(t, "Salary", arg.Name)
				assert.Equal(t, pgtype.Text{String: color, Valid: true}, arg.Color)
				return db.IncomeCategory{
					ID:     catID,
					UserID: userID,
					Name:   "Salary",
					Color:  pgtype.Text{String: color, Valid: true},
				}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.CreateIncome(context.Background(), userID, CreateRequest{
			Name:  "Salary",
			Color: &color,
		})
		require.NoError(t, err)
		assert.Equal(t, catID, result.ID)
		assert.Equal(t, "Salary", result.Name)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			CreateIncomeCategoryFn: func(ctx context.Context, arg db.CreateIncomeCategoryParams) (db.IncomeCategory, error) {
				return db.IncomeCategory{}, errors.New("db error")
			},
		}
		svc := NewService(m)
		_, err := svc.CreateIncome(context.Background(), userID, CreateRequest{Name: "X"})
		require.Error(t, err)
	})
}

func TestCategoryService_ListIncome(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListIncomeCategoriesFn: func(ctx context.Context, uid uuid.UUID) ([]db.IncomeCategory, error) {
				assert.Equal(t, userID, uid)
				return []db.IncomeCategory{
					{ID: uuid.New(), Name: "Salary"},
					{ID: uuid.New(), Name: "Freelance"},
					{ID: uuid.New(), Name: "Investments"},
				}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.ListIncome(context.Background(), userID)
		require.NoError(t, err)
		assert.Len(t, result, 3)
	})

	t.Run("empty", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListIncomeCategoriesFn: func(ctx context.Context, uid uuid.UUID) ([]db.IncomeCategory, error) {
				return []db.IncomeCategory{}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.ListIncome(context.Background(), userID)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			ListIncomeCategoriesFn: func(ctx context.Context, uid uuid.UUID) ([]db.IncomeCategory, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewService(m)
		_, err := svc.ListIncome(context.Background(), userID)
		require.Error(t, err)
	})
}

func TestCategoryService_GetIncome(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetIncomeCategoryFn: func(ctx context.Context, arg db.GetIncomeCategoryParams) (db.IncomeCategory, error) {
				assert.Equal(t, catID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				return db.IncomeCategory{ID: catID, Name: "Salary"}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.GetIncome(context.Background(), catID, userID)
		require.NoError(t, err)
		assert.Equal(t, "Salary", result.Name)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			GetIncomeCategoryFn: func(ctx context.Context, arg db.GetIncomeCategoryParams) (db.IncomeCategory, error) {
				return db.IncomeCategory{}, errors.New("not found")
			},
		}
		svc := NewService(m)
		_, err := svc.GetIncome(context.Background(), catID, userID)
		require.Error(t, err)
	})
}

func TestCategoryService_UpdateIncome(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()
	color := "#FFFF00"

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			UpdateIncomeCategoryFn: func(ctx context.Context, arg db.UpdateIncomeCategoryParams) (db.IncomeCategory, error) {
				assert.Equal(t, catID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				assert.Equal(t, "Updated Income", arg.Name)
				assert.Equal(t, pgtype.Text{String: color, Valid: true}, arg.Color)
				return db.IncomeCategory{
					ID:   catID,
					Name: "Updated Income",
					Color: pgtype.Text{String: color, Valid: true},
				}, nil
			},
		}
		svc := NewService(m)
		result, err := svc.UpdateIncome(context.Background(), catID, userID, UpdateRequest{
			Name:  "Updated Income",
			Color: &color,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Income", result.Name)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			UpdateIncomeCategoryFn: func(ctx context.Context, arg db.UpdateIncomeCategoryParams) (db.IncomeCategory, error) {
				return db.IncomeCategory{}, errors.New("db error")
			},
		}
		svc := NewService(m)
		_, err := svc.UpdateIncome(context.Background(), catID, userID, UpdateRequest{Name: "X"})
		require.Error(t, err)
	})
}

func TestCategoryService_DeleteIncome(t *testing.T) {
	userID := uuid.New()
	catID := uuid.New()

	t.Run("success", func(t *testing.T) {
		m := &mock.MockQuerier{
			DeleteIncomeCategoryFn: func(ctx context.Context, arg db.DeleteIncomeCategoryParams) error {
				assert.Equal(t, catID, arg.ID)
				assert.Equal(t, userID, arg.UserID)
				return nil
			},
		}
		svc := NewService(m)
		err := svc.DeleteIncome(context.Background(), catID, userID)
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		m := &mock.MockQuerier{
			DeleteIncomeCategoryFn: func(ctx context.Context, arg db.DeleteIncomeCategoryParams) error {
				return errors.New("not found")
			},
		}
		svc := NewService(m)
		err := svc.DeleteIncome(context.Background(), catID, userID)
		require.Error(t, err)
	})
}
