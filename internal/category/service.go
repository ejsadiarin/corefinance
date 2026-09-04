package category

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
)

type Service struct {
	queries *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		queries: db.New(pool),
	}
}

// --- Expense Categories ---

func (s *Service) CreateExpense(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.ExpenseCategory, error) {
	return s.queries.CreateExpenseCategory(ctx, db.CreateExpenseCategoryParams{
		UserID: userID,
		Name:   req.Name,
		Color:  helper.ToPgText(req.Color),
		Icon:   helper.ToPgText(req.Icon),
	})
}

func (s *Service) ListExpense(ctx context.Context, userID uuid.UUID) ([]db.ExpenseCategory, error) {
	return s.queries.ListExpenseCategories(ctx, userID)
}

func (s *Service) GetExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.ExpenseCategory, error) {
	return s.queries.GetExpenseCategory(ctx, db.GetExpenseCategoryParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) UpdateExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.ExpenseCategory, error) {
	return s.queries.UpdateExpenseCategory(ctx, db.UpdateExpenseCategoryParams{
		ID:     id,
		UserID: userID,
		Name:   req.Name,
		Color:  helper.ToPgText(req.Color),
		Icon:   helper.ToPgText(req.Icon),
	})
}

func (s *Service) DeleteExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.queries.DeleteExpenseCategory(ctx, db.DeleteExpenseCategoryParams{
		ID:     id,
		UserID: userID,
	})
}

// --- Income Categories ---

func (s *Service) CreateIncome(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.IncomeCategory, error) {
	return s.queries.CreateIncomeCategory(ctx, db.CreateIncomeCategoryParams{
		UserID: userID,
		Name:   req.Name,
		Color:  helper.ToPgText(req.Color),
		Icon:   helper.ToPgText(req.Icon),
	})
}

func (s *Service) ListIncome(ctx context.Context, userID uuid.UUID) ([]db.IncomeCategory, error) {
	return s.queries.ListIncomeCategories(ctx, userID)
}

func (s *Service) GetIncome(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.IncomeCategory, error) {
	return s.queries.GetIncomeCategory(ctx, db.GetIncomeCategoryParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) UpdateIncome(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.IncomeCategory, error) {
	return s.queries.UpdateIncomeCategory(ctx, db.UpdateIncomeCategoryParams{
		ID:     id,
		UserID: userID,
		Name:   req.Name,
		Color:  helper.ToPgText(req.Color),
		Icon:   helper.ToPgText(req.Icon),
	})
}

func (s *Service) DeleteIncome(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.queries.DeleteIncomeCategory(ctx, db.DeleteIncomeCategoryParams{
		ID:     id,
		UserID: userID,
	})
}
