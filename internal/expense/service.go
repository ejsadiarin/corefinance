package expense

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

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

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.Expense, error) {
	return s.queries.CreateExpense(ctx, db.CreateExpenseParams{
		UserID:        userID,
		CategoryID:    helper.ToPgUUID(req.CategoryID),
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		Description:   req.Description,
		Notes:         helper.ToPgText(req.Notes),
		ExpenseDate:   helper.ToPgDate(req.ExpenseDate),
		RecurringType: req.RecurringType,
		Priority:      req.Priority,
		Status:        req.Status,
		IsDebt:        req.IsDebt,
		StartDate:     helper.ToPgDatePtr(req.StartDate),
		EndDate:       helper.ToPgDatePtr(req.EndDate),
		SourceRuleID:  helper.ToPgUUID(req.SourceRuleID),
	})
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.GetExpenseRow, error) {
	return s.queries.GetExpense(ctx, db.GetExpenseParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, params ListParams) ([]db.ListExpensesRow, error) {
	page, pageSize := int32(params.Page), int32(params.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	return s.queries.ListExpenses(ctx, db.ListExpensesParams{
		UserID:        userID,
		StartDate:     helper.ToPgDatePtr(params.StartDate),
		EndDate:       helper.ToPgDatePtr(params.EndDate),
		CategoryID:    helper.ToUUID(params.CategoryID),
		Priority:      helper.StrVal(params.Priority),
		Status:        helper.StrVal(params.Status),
		RecurringType: helper.StrVal(params.RecurringType),
		PageLimit:     pageSize,
		PageOffset:    (page - 1) * pageSize,
	})
}

func (s *Service) Search(ctx context.Context, userID uuid.UUID, params SearchParams) ([]db.SearchExpensesRow, error) {
	page, pageSize := int32(params.Page), int32(params.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	return s.queries.SearchExpenses(ctx, db.SearchExpensesParams{
		UserID:      userID,
		Query:       helper.ToPgText(&params.Query),
		PageLimit:   pageSize,
		PageOffset:  (page - 1) * pageSize,
	})
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.Expense, error) {
	return s.queries.UpdateExpense(ctx, db.UpdateExpenseParams{
		ID:            id,
		UserID:        userID,
		CategoryID:    helper.ToPgUUID(req.CategoryID),
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		Description:   req.Description,
		Notes:         helper.ToPgText(req.Notes),
		ExpenseDate:   helper.ToPgDate(req.ExpenseDate),
		RecurringType: req.RecurringType,
		Priority:      req.Priority,
		Status:        req.Status,
		IsDebt:        req.IsDebt,
		StartDate:     helper.ToPgDatePtr(req.StartDate),
		EndDate:       helper.ToPgDatePtr(req.EndDate),
		SourceRuleID:  helper.ToPgUUID(req.SourceRuleID),
	})
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.queries.DeleteExpense(ctx, db.DeleteExpenseParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) Skip(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.queries.SkipExpense(ctx, db.SkipExpenseParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) CheckSkipped(ctx context.Context, userID uuid.UUID, startDate, endDate string) (bool, error) {
	return s.queries.CheckSkippedExpense(ctx, db.CheckSkippedExpenseParams{
		UserID:        userID,
		ExpenseDate:   helper.ToPgDate(startDate),
		ExpenseDate_2: helper.ToPgDate(endDate),
	})
}

func (s *Service) StatsByCategory(ctx context.Context, userID uuid.UUID, startDate, endDate *string) ([]db.GetExpenseStatsByCategoryRow, error) {
	return s.queries.GetExpenseStatsByCategory(ctx, db.GetExpenseStatsByCategoryParams{
		UserID:    userID,
		StartDate: helper.ToPgDatePtr(startDate),
		EndDate:   helper.ToPgDatePtr(endDate),
	})
}

func (s *Service) TotalByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate string) (decimal.Decimal, error) {
	return s.queries.GetTotalExpensesByDateRange(ctx, db.GetTotalExpensesByDateRangeParams{
		UserID:        userID,
		ExpenseDate:   helper.ToPgDate(startDate),
		ExpenseDate_2: helper.ToPgDate(endDate),
	})
}
