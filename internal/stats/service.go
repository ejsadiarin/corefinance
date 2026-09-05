package stats

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

func (s *Service) Summary(ctx context.Context, userID uuid.UUID, startDate, endDate *string) (Summary, error) {
	slog.Debug("stats.Service.Summary", "user_id", userID)
	row, err := s.queries.GetSummary(ctx, db.GetSummaryParams{
		UserID:    userID,
		StartDate: helper.ToPgDatePtr(startDate),
		EndDate:   helper.ToPgDatePtr(endDate),
	})
	if err != nil {
		return Summary{}, err
	}
	return Summary{
		TotalExpenses: helper.ToDecimal(row.TotalExpenses),
		TotalIncomes:  helper.ToDecimal(row.TotalIncomes),
		ExpenseCount:  row.ExpenseCount,
		IncomeCount:   row.IncomeCount,
	}, nil
}

func (s *Service) Trends(ctx context.Context, userID uuid.UUID, startDate, endDate *string) ([]Trend, error) {
	slog.Debug("stats.Service.Trends", "user_id", userID)
	rows, err := s.queries.GetTrends(ctx, db.GetTrendsParams{
		UserID:    userID,
		StartDate: helper.ToPgDatePtr(startDate),
		EndDate:   helper.ToPgDatePtr(endDate),
	})
	if err != nil {
		return nil, err
	}
	result := make([]Trend, len(rows))
	for i, row := range rows {
		result[i] = Trend{
			Month:         row.Month,
			TotalExpenses: helper.ToDecimal(row.TotalExpenses),
			TotalIncomes:  helper.ToDecimal(row.TotalIncomes),
		}
	}
	return result, nil
}

func (s *Service) CategoryBreakdown(ctx context.Context, userID uuid.UUID, startDate, endDate *string) ([]CategoryBreakdownItem, error) {
	slog.Debug("stats.Service.CategoryBreakdown", "user_id", userID)
	rows, err := s.queries.GetCategoryBreakdown(ctx, db.GetCategoryBreakdownParams{
		UserID:    userID,
		StartDate: helper.ToPgDatePtr(startDate),
		EndDate:   helper.ToPgDatePtr(endDate),
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
			TotalAmount:   helper.ToDecimal(row.TotalAmount),
		}
	}
	return result, nil
}

func (s *Service) SavingsRate(ctx context.Context, userID uuid.UUID, startDate, endDate *string) (SavingsRate, error) {
	slog.Debug("stats.Service.SavingsRate", "user_id", userID)
	row, err := s.queries.GetSavingsRate(ctx, db.GetSavingsRateParams{
		UserID:    userID,
		StartDate: helper.ToPgDatePtr(startDate),
		EndDate:   helper.ToPgDatePtr(endDate),
	})
	if err != nil {
		return SavingsRate{}, err
	}
	return SavingsRate{
		TotalIncomes:  helper.ToDecimal(row.TotalIncomes),
		TotalExpenses: helper.ToDecimal(row.TotalExpenses),
		SavingsRate:   helper.ToDecimal(row.SavingsRate),
	}, nil
}

func (s *Service) FiftyThirtyTwenty(ctx context.Context, userID uuid.UUID, startDate, endDate *string) (FiftyThirtyTwenty, error) {
	slog.Debug("stats.Service.FiftyThirtyTwenty", "user_id", userID)
	row, err := s.queries.GetFiftyThirtyTwenty(ctx, db.GetFiftyThirtyTwentyParams{
		UserID:    userID,
		StartDate: helper.ToPgDatePtr(startDate),
		EndDate:   helper.ToPgDatePtr(endDate),
	})
	if err != nil {
		return FiftyThirtyTwenty{}, err
	}
	return FiftyThirtyTwenty{
		TotalIncomes:  helper.ToDecimal(row.TotalIncomes),
		TotalExpenses: helper.ToDecimal(row.TotalExpenses),
		Needs:         helper.ToDecimal(row.Needs),
		Wants:         helper.ToDecimal(row.Wants),
		Savings:       helper.ToDecimal(row.Savings),
		NeedsPct:      helper.ToDecimal(row.NeedsPct),
		WantsPct:      helper.ToDecimal(row.WantsPct),
		SavingsPct:    helper.ToDecimal(row.SavingsPct),
	}, nil
}

func (s *Service) UpcomingBills(ctx context.Context, userID uuid.UUID) ([]db.GetUpcomingRecurringExpensesRow, error) {
	slog.Debug("stats.Service.UpcomingBills", "user_id", userID)
	return s.queries.GetUpcomingRecurringExpenses(ctx, userID)
}

func (s *Service) SpendingVelocity(ctx context.Context, userID uuid.UUID, startDate, endDate string) (SpendingVelocity, error) {
	slog.Debug("stats.Service.SpendingVelocity", "user_id", userID)
	row, err := s.queries.GetSpendingVelocity(ctx, db.GetSpendingVelocityParams{
		UserID:    userID,
		StartDate: helper.ToPgDate(startDate),
		EndDate:   helper.ToPgDate(endDate),
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
	slog.Debug("stats.Service.CurrentTotalMoney", "user_id", userID)
	totalMoney, err := s.queries.GetCurrentTotalMoney(ctx, db.GetCurrentTotalMoneyParams{
		UserID:    userID,
		StartDate: helper.ToPgDatePtr(startDate),
		EndDate:   helper.ToPgDatePtr(endDate),
	})
	if err != nil {
		return CurrentTotalMoney{}, err
	}
	return CurrentTotalMoney{
		TotalMoney: totalMoney,
	}, nil
}

func (s *Service) MonthOverMonthTrends(ctx context.Context, userID uuid.UUID, startDate, endDate *string) ([]MonthOverMonthTrend, error) {
	slog.Debug("stats.Service.MonthOverMonthTrends", "user_id", userID)
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
		StartDate: helper.ToPgDate(start),
		EndDate:   helper.ToPgDate(end),
	})
	if err != nil {
		return nil, err
	}
	result := make([]MonthOverMonthTrend, len(rows))
	for i, row := range rows {
		result[i] = MonthOverMonthTrend{
			Month:       row.Month,
			TotalAmount: helper.ToDecimal(row.TotalAmount),
		}
	}
	return result, nil
}
