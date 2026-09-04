package stats

import (
	"context"

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

func (s *Service) Summary(ctx context.Context, userID uuid.UUID, startDate, endDate *string) (Summary, error) {
	row, err := s.queries.GetSummary(ctx, db.GetSummaryParams{
		UserID:    userID,
		StartDate: toPgDatePtr(startDate),
		EndDate:   toPgDatePtr(endDate),
	})
	if err != nil {
		return Summary{}, err
	}
	return Summary{
		TotalExpenses: toDecimal(row.TotalExpenses),
		TotalIncomes:  toDecimal(row.TotalIncomes),
		ExpenseCount:  row.ExpenseCount,
		IncomeCount:   row.IncomeCount,
	}, nil
}

func (s *Service) Trends(ctx context.Context, userID uuid.UUID, startDate, endDate *string) ([]Trend, error) {
	rows, err := s.queries.GetTrends(ctx, db.GetTrendsParams{
		UserID:    userID,
		StartDate: toPgDatePtr(startDate),
		EndDate:   toPgDatePtr(endDate),
	})
	if err != nil {
		return nil, err
	}
	result := make([]Trend, len(rows))
	for i, row := range rows {
		result[i] = Trend{
			Month:         row.Month,
			TotalExpenses: toDecimal(row.TotalExpenses),
			TotalIncomes:  toDecimal(row.TotalIncomes),
		}
	}
	return result, nil
}

func (s *Service) CategoryBreakdown(ctx context.Context, userID uuid.UUID, startDate, endDate *string) ([]CategoryBreakdownItem, error) {
	rows, err := s.queries.GetCategoryBreakdown(ctx, db.GetCategoryBreakdownParams{
		UserID:    userID,
		StartDate: toPgDatePtr(startDate),
		EndDate:   toPgDatePtr(endDate),
	})
	if err != nil {
		return nil, err
	}
	result := make([]CategoryBreakdownItem, len(rows))
	for i, row := range rows {
		color := ""
		if row.CategoryColor.Valid {
			color = row.CategoryColor.String
		}
		result[i] = CategoryBreakdownItem{
			CategoryID:    row.CategoryID.String(),
			CategoryName:  row.CategoryName,
			CategoryColor: &color,
			ExpenseCount:  row.ExpenseCount,
			TotalAmount:   toDecimal(row.TotalAmount),
		}
	}
	return result, nil
}

func (s *Service) SavingsRate(ctx context.Context, userID uuid.UUID, startDate, endDate *string) (SavingsRate, error) {
	row, err := s.queries.GetSavingsRate(ctx, db.GetSavingsRateParams{
		UserID:    userID,
		StartDate: toPgDatePtr(startDate),
		EndDate:   toPgDatePtr(endDate),
	})
	if err != nil {
		return SavingsRate{}, err
	}
	return SavingsRate{
		TotalIncomes:  toDecimal(row.TotalIncomes),
		TotalExpenses: toDecimal(row.TotalExpenses),
		SavingsRate:   toDecimal(row.SavingsRate),
	}, nil
}

func (s *Service) FiftyThirtyTwenty(ctx context.Context, userID uuid.UUID, startDate, endDate *string) (FiftyThirtyTwenty, error) {
	row, err := s.queries.GetFiftyThirtyTwenty(ctx, db.GetFiftyThirtyTwentyParams{
		UserID:    userID,
		StartDate: toPgDatePtr(startDate),
		EndDate:   toPgDatePtr(endDate),
	})
	if err != nil {
		return FiftyThirtyTwenty{}, err
	}
	return FiftyThirtyTwenty{
		TotalIncomes:  toDecimal(row.TotalIncomes),
		TotalExpenses: toDecimal(row.TotalExpenses),
		Needs:         toDecimal(row.Needs),
		Wants:         toDecimal(row.Wants),
		Savings:       toDecimal(row.Savings),
		NeedsPct:      toDecimal(row.NeedsPct),
		WantsPct:      toDecimal(row.WantsPct),
		SavingsPct:    toDecimal(row.SavingsPct),
	}, nil
}

func (s *Service) UpcomingBills(ctx context.Context, userID uuid.UUID) ([]db.GetUpcomingRecurringExpensesRow, error) {
	return s.queries.GetUpcomingRecurringExpenses(ctx, userID)
}

func (s *Service) SpendingVelocity(ctx context.Context, userID uuid.UUID, startDate, endDate string) (SpendingVelocity, error) {
	row, err := s.queries.GetSpendingVelocity(ctx, db.GetSpendingVelocityParams{
		UserID:    userID,
		StartDate: toPgDate(startDate),
		EndDate:   toPgDate(endDate),
	})
	if err != nil {
		return SpendingVelocity{}, err
	}
	return SpendingVelocity{
		AvgMonthlySpending: row.AvgMonthlySpending,
		MonthsWithData:     int(row.MonthsWithData),
	}, nil
}

func (s *Service) CurrentTotalMoney(ctx context.Context, userID uuid.UUID, startDate, endDate *string) (CurrentTotalMoney, error) {
	totalMoney, err := s.queries.GetCurrentTotalMoney(ctx, db.GetCurrentTotalMoneyParams{
		UserID:    userID,
		StartDate: toPgDatePtr(startDate),
		EndDate:   toPgDatePtr(endDate),
	})
	if err != nil {
		return CurrentTotalMoney{}, err
	}
	return CurrentTotalMoney{
		TotalMoney: totalMoney,
	}, nil
}

func (s *Service) MonthOverMonthTrends(ctx context.Context, userID uuid.UUID, startDate, endDate *string) ([]MonthOverMonthTrend, error) {
	start := ""
	end := ""
	if startDate != nil {
		start = *startDate
	}
	if endDate != nil {
		end = *endDate
	}
	rows, err := s.queries.GetMonthOverMonthTrends(ctx, db.GetMonthOverMonthTrendsParams{
		UserID:    userID,
		StartDate: toPgDate(start),
		EndDate:   toPgDate(end),
	})
	if err != nil {
		return nil, err
	}
	result := make([]MonthOverMonthTrend, len(rows))
	for i, row := range rows {
		result[i] = MonthOverMonthTrend{
			Month:       row.Month,
			TotalAmount: toDecimal(row.TotalAmount),
		}
	}
	return result, nil
}

// --- Helpers ---

func toPgDate(date string) pgtype.Date {
	if date == "" {
		return pgtype.Date{Valid: false}
	}
	t, err := parseDate(date)
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

func toDecimal(v interface{}) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	switch val := v.(type) {
	case decimal.Decimal:
		return val
	case float64:
		return decimal.NewFromFloat(val)
	case int64:
		return decimal.NewFromInt(val)
	case int:
		return decimal.NewFromInt(int64(val))
	default:
		return decimal.Zero
	}
}
