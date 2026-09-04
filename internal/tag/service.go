package tag

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

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.Tag, error) {
	return s.queries.CreateTag(ctx, db.CreateTagParams{
		UserID: userID,
		Name:   req.Name,
		Color:  helper.ToPgText(req.Color),
	})
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]db.Tag, error) {
	return s.queries.ListTags(ctx, userID)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.Tag, error) {
	return s.queries.GetTag(ctx, db.GetTagParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.Tag, error) {
	return s.queries.UpdateTag(ctx, db.UpdateTagParams{
		ID:     id,
		UserID: userID,
		Name:   req.Name,
		Color:  helper.ToPgText(req.Color),
	})
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.queries.DeleteTag(ctx, db.DeleteTagParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) AddTagToExpense(ctx context.Context, expenseID, tagID uuid.UUID) error {
	return s.queries.AddTagToExpense(ctx, db.AddTagToExpenseParams{
		ExpenseID: expenseID,
		TagID:     tagID,
	})
}

func (s *Service) RemoveTagFromExpense(ctx context.Context, expenseID, tagID uuid.UUID) error {
	return s.queries.RemoveTagFromExpense(ctx, db.RemoveTagFromExpenseParams{
		ExpenseID: expenseID,
		TagID:     tagID,
	})
}

func (s *Service) GetTagsByExpenseID(ctx context.Context, expenseID uuid.UUID) ([]db.Tag, error) {
	return s.queries.GetTagsByExpenseID(ctx, expenseID)
}

func (s *Service) AddTagToIncome(ctx context.Context, incomeID, tagID uuid.UUID) error {
	return s.queries.AddTagToIncome(ctx, db.AddTagToIncomeParams{
		IncomeID: incomeID,
		TagID:    tagID,
	})
}

func (s *Service) RemoveTagFromIncome(ctx context.Context, incomeID, tagID uuid.UUID) error {
	return s.queries.RemoveTagFromIncome(ctx, db.RemoveTagFromIncomeParams{
		IncomeID: incomeID,
		TagID:    tagID,
	})
}

func (s *Service) GetTagsByIncomeID(ctx context.Context, incomeID uuid.UUID) ([]db.Tag, error) {
	return s.queries.GetTagsByIncomeID(ctx, incomeID)
}
