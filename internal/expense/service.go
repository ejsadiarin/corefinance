package expense

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
)

type Service struct {
	queries db.Querier
}

func NewService(queries db.Querier) *Service {
	return &Service{queries: queries}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.Expense, error) {
	slog.Debug("expense.Service.Create", "user_id", userID)
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
	slog.Debug("expense.Service.Get", "id", id, "user_id", userID)
	return s.queries.GetExpense(ctx, db.GetExpenseParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, params ListParams) ([]db.ListExpensesRow, error) {
	slog.Debug("expense.Service.List", "user_id", userID)
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
	slog.Debug("expense.Service.Search", "user_id", userID, "query", params.Query)
	page, pageSize := int32(params.Page), int32(params.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	return s.queries.SearchExpenses(ctx, db.SearchExpensesParams{
		UserID:     userID,
		Query:      helper.ToPgText(&params.Query),
		PageLimit:  pageSize,
		PageOffset: (page - 1) * pageSize,
	})
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.Expense, error) {
	slog.Debug("expense.Service.Update", "id", id, "user_id", userID)
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
	slog.Debug("expense.Service.Delete", "id", id, "user_id", userID)
	return s.queries.DeleteExpense(ctx, db.DeleteExpenseParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) Skip(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	slog.Debug("expense.Service.Skip", "id", id, "user_id", userID)
	return s.queries.SkipExpense(ctx, db.SkipExpenseParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) CheckSkipped(ctx context.Context, userID uuid.UUID, startDate, endDate string) (bool, error) {
	slog.Debug("expense.Service.CheckSkipped", "user_id", userID)
	return s.queries.CheckSkippedExpense(ctx, db.CheckSkippedExpenseParams{
		UserID:        userID,
		ExpenseDate:   helper.ToPgDate(startDate),
		ExpenseDate_2: helper.ToPgDate(endDate),
	})
}

func (s *Service) StatsByCategory(ctx context.Context, userID uuid.UUID, startDate, endDate *string) ([]db.GetExpenseStatsByCategoryRow, error) {
	slog.Debug("expense.Service.StatsByCategory", "user_id", userID)
	return s.queries.GetExpenseStatsByCategory(ctx, db.GetExpenseStatsByCategoryParams{
		UserID:    userID,
		StartDate: helper.ToPgDatePtr(startDate),
		EndDate:   helper.ToPgDatePtr(endDate),
	})
}

func (s *Service) TotalByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate string) (decimal.Decimal, error) {
	slog.Debug("expense.Service.TotalByDateRange", "user_id", userID)
	return s.queries.GetTotalExpensesByDateRange(ctx, db.GetTotalExpensesByDateRangeParams{
		UserID:        userID,
		ExpenseDate:   helper.ToPgDate(startDate),
		ExpenseDate_2: helper.ToPgDate(endDate),
	})
}
