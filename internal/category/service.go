package category

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
)

type Service struct {
	queries db.Querier
}

func NewService(queries db.Querier) *Service {
	return &Service{queries: queries}
}

// --- Expense Categories ---

func (s *Service) CreateExpense(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.ExpenseCategory, error) {
	slog.Debug("category.Service.CreateExpense", "user_id", userID)
	return s.queries.CreateExpenseCategory(ctx, db.CreateExpenseCategoryParams{
		UserID: userID,
		Name:   req.Name,
		Color:  helper.ToPgText(req.Color),
		Icon:   helper.ToPgText(req.Icon),
	})
}

func (s *Service) ListExpense(ctx context.Context, userID uuid.UUID) ([]db.ExpenseCategory, error) {
	slog.Debug("category.Service.ListExpense", "user_id", userID)
	return s.queries.ListExpenseCategories(ctx, userID)
}

func (s *Service) GetExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.ExpenseCategory, error) {
	slog.Debug("category.Service.GetExpense", "id", id, "user_id", userID)
	return s.queries.GetExpenseCategory(ctx, db.GetExpenseCategoryParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) UpdateExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.ExpenseCategory, error) {
	slog.Debug("category.Service.UpdateExpense", "id", id, "user_id", userID)
	return s.queries.UpdateExpenseCategory(ctx, db.UpdateExpenseCategoryParams{
		ID:     id,
		UserID: userID,
		Name:   req.Name,
		Color:  helper.ToPgText(req.Color),
		Icon:   helper.ToPgText(req.Icon),
	})
}

func (s *Service) DeleteExpense(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	slog.Debug("category.Service.DeleteExpense", "id", id, "user_id", userID)
	return s.queries.DeleteExpenseCategory(ctx, db.DeleteExpenseCategoryParams{
		ID:     id,
		UserID: userID,
	})
}

// --- Income Categories ---

func (s *Service) CreateIncome(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.IncomeCategory, error) {
	slog.Debug("category.Service.CreateIncome", "user_id", userID)
	return s.queries.CreateIncomeCategory(ctx, db.CreateIncomeCategoryParams{
		UserID: userID,
		Name:   req.Name,
		Color:  helper.ToPgText(req.Color),
		Icon:   helper.ToPgText(req.Icon),
	})
}

func (s *Service) ListIncome(ctx context.Context, userID uuid.UUID) ([]db.IncomeCategory, error) {
	slog.Debug("category.Service.ListIncome", "user_id", userID)
	return s.queries.ListIncomeCategories(ctx, userID)
}

func (s *Service) GetIncome(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.IncomeCategory, error) {
	slog.Debug("category.Service.GetIncome", "id", id, "user_id", userID)
	return s.queries.GetIncomeCategory(ctx, db.GetIncomeCategoryParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) UpdateIncome(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.IncomeCategory, error) {
	slog.Debug("category.Service.UpdateIncome", "id", id, "user_id", userID)
	return s.queries.UpdateIncomeCategory(ctx, db.UpdateIncomeCategoryParams{
		ID:     id,
		UserID: userID,
		Name:   req.Name,
		Color:  helper.ToPgText(req.Color),
		Icon:   helper.ToPgText(req.Icon),
	})
}

func (s *Service) DeleteIncome(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	slog.Debug("category.Service.DeleteIncome", "id", id, "user_id", userID)
	return s.queries.DeleteIncomeCategory(ctx, db.DeleteIncomeCategoryParams{
		ID:     id,
		UserID: userID,
	})
}
