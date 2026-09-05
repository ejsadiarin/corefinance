package tag

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

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.Tag, error) {
	slog.Debug("tag.Service.Create", "user_id", userID)
	return s.queries.CreateTag(ctx, db.CreateTagParams{
		UserID: userID,
		Name:   req.Name,
		Color:  helper.ToPgText(req.Color),
	})
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]db.Tag, error) {
	slog.Debug("tag.Service.List", "user_id", userID)
	return s.queries.ListTags(ctx, userID)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.Tag, error) {
	slog.Debug("tag.Service.Get", "id", id, "user_id", userID)
	return s.queries.GetTag(ctx, db.GetTagParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.Tag, error) {
	slog.Debug("tag.Service.Update", "id", id, "user_id", userID)
	return s.queries.UpdateTag(ctx, db.UpdateTagParams{
		ID:     id,
		UserID: userID,
		Name:   req.Name,
		Color:  helper.ToPgText(req.Color),
	})
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	slog.Debug("tag.Service.Delete", "id", id, "user_id", userID)
	return s.queries.DeleteTag(ctx, db.DeleteTagParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) AddTagToExpense(ctx context.Context, expenseID, tagID uuid.UUID) error {
	slog.Debug("tag.Service.AddTagToExpense", "expense_id", expenseID, "tag_id", tagID)
	return s.queries.AddTagToExpense(ctx, db.AddTagToExpenseParams{
		ExpenseID: expenseID,
		TagID:     tagID,
	})
}

func (s *Service) RemoveTagFromExpense(ctx context.Context, expenseID, tagID uuid.UUID) error {
	slog.Debug("tag.Service.RemoveTagFromExpense", "expense_id", expenseID, "tag_id", tagID)
	return s.queries.RemoveTagFromExpense(ctx, db.RemoveTagFromExpenseParams{
		ExpenseID: expenseID,
		TagID:     tagID,
	})
}

func (s *Service) GetTagsByExpenseID(ctx context.Context, expenseID uuid.UUID) ([]db.Tag, error) {
	slog.Debug("tag.Service.GetTagsByExpenseID", "expense_id", expenseID)
	return s.queries.GetTagsByExpenseID(ctx, expenseID)
}

func (s *Service) AddTagToIncome(ctx context.Context, incomeID, tagID uuid.UUID) error {
	slog.Debug("tag.Service.AddTagToIncome", "income_id", incomeID, "tag_id", tagID)
	return s.queries.AddTagToIncome(ctx, db.AddTagToIncomeParams{
		IncomeID: incomeID,
		TagID:    tagID,
	})
}

func (s *Service) RemoveTagFromIncome(ctx context.Context, incomeID, tagID uuid.UUID) error {
	slog.Debug("tag.Service.RemoveTagFromIncome", "income_id", incomeID, "tag_id", tagID)
	return s.queries.RemoveTagFromIncome(ctx, db.RemoveTagFromIncomeParams{
		IncomeID: incomeID,
		TagID:    tagID,
	})
}

func (s *Service) GetTagsByIncomeID(ctx context.Context, incomeID uuid.UUID) ([]db.Tag, error) {
	slog.Debug("tag.Service.GetTagsByIncomeID", "income_id", incomeID)
	return s.queries.GetTagsByIncomeID(ctx, incomeID)
}
