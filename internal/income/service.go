package income

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

type Service struct {
	queries *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		queries: db.New(pool),
	}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (db.Income, error) {
	return s.queries.CreateIncome(ctx, db.CreateIncomeParams{
		UserID:        userID,
		CategoryID:    toPgUUID(req.CategoryID),
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		Description:   req.Description,
		Notes:         toPgText(req.Notes),
		Date:          toPgDate(req.Date),
		RecurringType: req.RecurringType,
		Priority:      req.Priority,
		Status:        req.Status,
		StartDate:     toPgDatePtr(req.StartDate),
		EndDate:       toPgDatePtr(req.EndDate),
		SourceRuleID:  toPgUUID(req.SourceRuleID),
	})
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, userID uuid.UUID) (db.GetIncomeRow, error) {
	return s.queries.GetIncome(ctx, db.GetIncomeParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, params ListParams) ([]db.ListIncomesRow, error) {
	page, pageSize := int32(params.Page), int32(params.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	return s.queries.ListIncomes(ctx, db.ListIncomesParams{
		UserID:        userID,
		StartDate:     toPgDatePtr(params.StartDate),
		EndDate:       toPgDatePtr(params.EndDate),
		CategoryID:    toUUID(params.CategoryID),
		Status:        strVal(params.Status),
		RecurringType: strVal(params.RecurringType),
		PageLimit:     pageSize,
		PageOffset:    (page - 1) * pageSize,
	})
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req UpdateRequest) (db.Income, error) {
	return s.queries.UpdateIncome(ctx, db.UpdateIncomeParams{
		ID:            id,
		UserID:        userID,
		CategoryID:    toPgUUID(req.CategoryID),
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      req.Currency,
		Description:   req.Description,
		Notes:         toPgText(req.Notes),
		Date:          toPgDate(req.Date),
		RecurringType: req.RecurringType,
		Priority:      req.Priority,
		Status:        req.Status,
		StartDate:     toPgDatePtr(req.StartDate),
		EndDate:       toPgDatePtr(req.EndDate),
		SourceRuleID:  toPgUUID(req.SourceRuleID),
	})
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.queries.DeleteIncome(ctx, db.DeleteIncomeParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) Skip(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.queries.SkipIncome(ctx, db.SkipIncomeParams{
		ID:     id,
		UserID: userID,
	})
}

func (s *Service) CheckSkipped(ctx context.Context, userID uuid.UUID, startDate, endDate string) (bool, error) {
	return s.queries.CheckSkippedIncome(ctx, db.CheckSkippedIncomeParams{
		UserID: userID,
		Date:   toPgDate(startDate),
		Date_2: toPgDate(endDate),
	})
}

func (s *Service) Occurrences(ctx context.Context, userID uuid.UUID, startDate, endDate *string) ([]db.Income, error) {
	return s.queries.GetIncomeOccurrences(ctx, db.GetIncomeOccurrencesParams{
		UserID:    userID,
		StartDate: toPgDatePtr(startDate),
		EndDate:   toPgDatePtr(endDate),
	})
}

func (s *Service) TotalByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate string) (decimal.Decimal, error) {
	return s.queries.GetTotalIncomesByDateRange(ctx, db.GetTotalIncomesByDateRangeParams{
		UserID: userID,
		Date:   toPgDate(startDate),
		Date_2: toPgDate(endDate),
	})
}

// --- Helpers ---

func toPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func toPgUUID(id *string) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	parsed, err := uuid.Parse(*id)
	if err != nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}
}

func toUUID(id *string) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	parsed, err := uuid.Parse(*id)
	if err != nil {
		return uuid.Nil
	}
	return parsed
}

func toPgDate(date string) pgtype.Date {
	if date == "" {
		return pgtype.Date{Valid: false}
	}
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: t, Valid: true}
}

func toPgDatePtr(date *string) pgtype.Date {
	if date == nil {
		return pgtype.Date{Valid: false}
	}
	return toPgDate(*date)
}
