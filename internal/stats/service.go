package stats

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/ejsadiarin/corefinance/internal/auth"
	"github.com/ejsadiarin/corefinance/internal/category"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
	"github.com/ejsadiarin/corefinance/internal/materialize"
	"github.com/ejsadiarin/corefinance/internal/rates"
)

type Service struct {
	queries db.Querier
	now     func() time.Time
	rates   rates.Provider
}

func NewService(queries db.Querier) *Service {
	return NewServiceWithRates(queries, rates.PHPOnly{})
}

// NewServiceWithRates projects rule entries at their due dates through the
// same exchange-rate provider the materialize pass writes them with (ADR-0003).
func NewServiceWithRates(queries db.Querier, provider rates.Provider) *Service {
	if provider == nil {
		provider = rates.PHPOnly{}
	}
	return &Service{queries: queries, now: time.Now, rates: provider}
}

// ownerZone is the zone a read dates "today" in (coregateway ADR-0005): the
// zone in the request's token, or the zone last seen for the owner (UTC if
// none).
func ownerZone(ctx context.Context, q db.Querier, userID uuid.UUID) (*time.Location, error) {
	if tz := auth.TimezoneFromContext(ctx); tz != "" {
		return materialize.LoadZone(tz), nil
	}
	loc, err := materialize.SeenZones{Queries: q}.Zone(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("owner timezone: %w", err)
	}
	return loc, nil
}

func (s *Service) Summary(ctx context.Context, userID uuid.UUID, startDate, endDate *string) (Summary, error) {
	slog.Debug("stats.Service.Summary", "user_id", userID)
	row, err := s.queries.GetSummary(ctx, db.GetSummaryParams{
		UserID:          userID,
		StartDate:       helper.ToPgDatePtr(startDate),
		EndDate:         helper.ToPgDatePtr(endDate),
		ReimbursementID: category.Reimbursement,
	})
	if err != nil {
		return Summary{}, err
	}
	return Summary{
		TotalExpenses:       helper.ToDecimal(row.TotalExpenses),
		TotalIncomes:        helper.ToDecimal(row.TotalIncomes),
		TotalReimbursements: helper.ToDecimal(row.TotalReimbursements),
		ExpenseCount:        row.ExpenseCount,
		IncomeCount:         row.IncomeCount,
	}, nil
}

func (s *Service) Trends(ctx context.Context, userID uuid.UUID, startDate, endDate *string) ([]Trend, error) {
	slog.Debug("stats.Service.Trends", "user_id", userID)
	rows, err := s.queries.GetTrends(ctx, db.GetTrendsParams{
		UserID:          userID,
		StartDate:       helper.ToPgDatePtr(startDate),
		EndDate:         helper.ToPgDatePtr(endDate),
		ReimbursementID: category.Reimbursement,
	})
	if err != nil {
		return nil, err
	}
	result := make([]Trend, len(rows))
	for i, row := range rows {
		result[i] = Trend{
			Month:               row.Month,
			TotalExpenses:       helper.ToDecimal(row.TotalExpenses),
			TotalIncomes:        helper.ToDecimal(row.TotalIncomes),
			TotalReimbursements: helper.ToDecimal(row.TotalReimbursements),
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
		UserID:          userID,
		StartDate:       helper.ToPgDatePtr(startDate),
		EndDate:         helper.ToPgDatePtr(endDate),
		ReimbursementID: category.Reimbursement,
	})
	if err != nil {
		return SavingsRate{}, err
	}
	return SavingsRate{
		TotalIncomes:        helper.ToDecimal(row.TotalIncomes),
		TotalReimbursements: helper.ToDecimal(row.TotalReimbursements),
		TotalExpenses:       helper.ToDecimal(row.TotalExpenses),
		SavingsRate:         helper.ToDecimal(row.SavingsRate),
	}, nil
}

func (s *Service) FiftyThirtyTwenty(ctx context.Context, userID uuid.UUID, startDate, endDate *string) (FiftyThirtyTwenty, error) {
	slog.Debug("stats.Service.FiftyThirtyTwenty", "user_id", userID)
	row, err := s.queries.GetFiftyThirtyTwenty(ctx, db.GetFiftyThirtyTwentyParams{
		UserID:          userID,
		StartDate:       helper.ToPgDatePtr(startDate),
		EndDate:         helper.ToPgDatePtr(endDate),
		ReimbursementID: category.Reimbursement,
	})
	if err != nil {
		return FiftyThirtyTwenty{}, err
	}
	return FiftyThirtyTwenty{
		TotalIncomes:        helper.ToDecimal(row.TotalIncomes),
		TotalReimbursements: helper.ToDecimal(row.TotalReimbursements),
		TotalExpenses:       helper.ToDecimal(row.TotalExpenses),
		Needs:               helper.ToDecimal(row.Needs),
		Wants:               helper.ToDecimal(row.Wants),
		Savings:             helper.ToDecimal(row.Savings),
		NeedsPct:            helper.ToDecimal(row.NeedsPct),
		WantsPct:            helper.ToDecimal(row.WantsPct),
		SavingsPct:          helper.ToDecimal(row.SavingsPct),
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

func (s *Service) CurrentTotalMoney(ctx context.Context, userID uuid.UUID) (CurrentTotalMoney, error) {
	slog.Debug("stats.Service.CurrentTotalMoney", "user_id", userID)
	loc, err := ownerZone(ctx, s.queries, userID)
	if err != nil {
		return CurrentTotalMoney{}, err
	}
	today := materialize.DateIn(s.now(), loc)
	row, err := s.queries.GetMoneyOnHand(ctx, db.GetMoneyOnHandParams{
		UserID:          userID,
		Today:           pgtype.Date{Time: today, Valid: true},
		WindowStart:     pgtype.Date{Time: today.AddDate(0, 0, -29), Valid: true},
		ReimbursementID: category.Reimbursement,
	})
	if err != nil {
		return CurrentTotalMoney{}, err
	}
	average := row.NetSpend.Div(decimal.NewFromInt(30))
	var runway *int64
	if row.NetSpend.IsPositive() {
		// money / (net spend / 30), without dividing twice so exact
		// multiples don't truncate a day early.
		days := row.MoneyOnHand.Mul(decimal.NewFromInt(30)).Div(row.NetSpend).IntPart()
		runway = &days
	}
	return CurrentTotalMoney{
		TotalMoney:        row.MoneyOnHand,
		Committed:         row.Committed,
		AverageDailySpend: average.Round(2),
		RunwayDays:        runway,
	}, nil
}

func (s *Service) PeriodBudget(ctx context.Context, userID uuid.UUID, kind, anchor string) (PeriodBudget, error) {
	slog.Debug("stats.Service.PeriodBudget", "user_id", userID, "period", kind, "anchor", anchor)
	loc, err := ownerZone(ctx, s.queries, userID)
	if err != nil {
		return PeriodBudget{}, err
	}
	today := materialize.DateIn(s.now(), loc)
	anchorDate := today
	if anchor != "" {
		anchorDate, err = time.Parse("2006-01-02", anchor)
		if err != nil {
			return PeriodBudget{}, fmt.Errorf("date must be YYYY-MM-DD")
		}
	}
	period, previous, err := periodRanges(kind, anchorDate)
	if err != nil {
		return PeriodBudget{}, err
	}

	// Spent through the same point in the previous period: clamp the elapsed
	// days to the period so a past period compares in full and a future one
	// has none yet.
	elapsed := daysBetween(period.start, today)
	if elapsed < 0 {
		elapsed = 0
	}
	if days := daysInRange(period); elapsed > days {
		elapsed = days
	}
	previousToPointEnd := previous.start.AddDate(0, 0, -1)
	if elapsed > 0 {
		previousToPointEnd = previous.start.AddDate(0, 0, elapsed-1)
		if previousToPointEnd.After(previous.end) {
			previousToPointEnd = previous.end
		}
	}

	row, err := s.queries.GetPeriodBudgetTotals(ctx, db.GetPeriodBudgetTotalsParams{
		UserID:             userID,
		ReimbursementID:    category.Reimbursement,
		PeriodStart:        pgDate(period.start),
		PeriodEnd:          pgDate(period.end),
		Today:              pgDate(today),
		Tomorrow:           pgDate(today.AddDate(0, 0, 1)),
		PreviousStart:      pgDate(previous.start),
		PreviousToPointEnd: pgDate(previousToPointEnd),
		PreviousEnd:        pgDate(previous.end),
	})
	if err != nil {
		return PeriodBudget{}, err
	}

	projectedIncome, projectedExpenses, err := s.projectRuleOccurrences(ctx, userID, period, today)
	if err != nil {
		return PeriodBudget{}, err
	}

	spent := netSpend(row.SpentExpenses, row.SpentReimbursements)
	funded := row.PostedIncome.Add(projectedIncome)
	bills := row.FutureExpenses.Add(projectedExpenses)
	left := funded.Sub(spent).Sub(bills)

	// Safe to spend is empty for a past period. A future period counts every
	// day as remaining; the current one counts from today.
	var safe *decimal.Decimal
	if !period.end.Before(today) {
		remaining := daysInRange(period)
		if !period.start.After(today) {
			remaining = daysBetween(today, period.end)
		}
		value := left.DivRound(decimal.NewFromInt(int64(remaining)), 2)
		safe = &value
	}

	return PeriodBudget{
		Period:               kind,
		StartDate:            formatDate(period.start),
		EndDate:              formatDate(period.end),
		Funded:               funded,
		ProjectedIncome:      projectedIncome,
		Spent:                spent,
		BillsStillDue:        bills,
		Left:                 left,
		SafeToSpendPerDay:    safe,
		PreviousPeriodStart:  formatDate(previous.start),
		PreviousPeriodEnd:    formatDate(previous.end),
		PreviousSpentToPoint: netSpend(row.PreviousSpentExpenses, row.PreviousSpentReimbursements),
		PreviousSpentTotal:   netSpend(row.PreviousTotalExpenses, row.PreviousTotalReimbursements),
	}, nil
}

// maxCashFlowBuckets bounds one series so the read stays cheap.
const maxCashFlowBuckets = 366

func (s *Service) CashFlow(ctx context.Context, userID uuid.UUID, kind string, count int) (CashFlowSeries, error) {
	slog.Debug("stats.Service.CashFlow", "user_id", userID, "bucket", kind, "count", count)
	if count < 1 || count > maxCashFlowBuckets {
		return CashFlowSeries{}, fmt.Errorf("count must be between 1 and %d", maxCashFlowBuckets)
	}
	loc, err := ownerZone(ctx, s.queries, userID)
	if err != nil {
		return CashFlowSeries{}, err
	}
	today := materialize.DateIn(s.now(), loc)

	current, _, err := periodRanges(kind, today)
	if err != nil {
		return CashFlowSeries{}, err
	}
	ranges := make([]dateRange, count)
	ranges[count-1] = current
	for i := count - 2; i >= 0; i-- {
		ranges[i], _, err = periodRanges(kind, ranges[i+1].start.AddDate(0, 0, -1))
		if err != nil {
			return CashFlowSeries{}, err
		}
	}

	rows, err := s.queries.ListCashFlowBuckets(ctx, db.ListCashFlowBucketsParams{
		Bucket:          kind,
		UserID:          userID,
		StartDate:       pgDate(ranges[0].start),
		Today:           pgDate(today),
		ReimbursementID: category.Reimbursement,
	})
	if err != nil {
		return CashFlowSeries{}, err
	}

	buckets := make([]CashFlowBucket, count)
	for i, r := range ranges {
		buckets[i] = CashFlowBucket{
			StartDate: formatDate(r.start),
			EndDate:   formatDate(r.end),
		}
	}
	for _, row := range rows {
		i := bucketIndex(kind, ranges[0].start, row.BucketStart.Time)
		if i < 0 || i >= count {
			continue
		}
		income := row.Incomes
		spent := netSpend(row.Expenses, row.Reimbursements)
		buckets[i].Income = income
		buckets[i].Spent = spent
		buckets[i].Net = income.Sub(spent)
	}
	return CashFlowSeries{Bucket: kind, Buckets: buckets}, nil
}

// ruleEntryKey identifies a rule's entry on one date.
type ruleEntryKey struct {
	ruleID uuid.UUID
	date   string
}

// projectRuleOccurrences sums the PHP amounts of active rules' due dates in
// the period that the materialize pass has not written yet. The rule's
// watermark says where its last pass stopped, so dates after it are still
// owed even when they are in the past (a worker that was down catches them
// up); dates still in the future are written when they fall due. Dates with
// an entry already are skipped, and a rule's projected reimbursement income
// is not funding.
//
// Owed income counts wherever it falls in the period. Owed expenses dated
// before today are not bills still due, so only today onward is projected as
// expense.
func (s *Service) projectRuleOccurrences(ctx context.Context, userID uuid.UUID, window dateRange, today time.Time) (income, expense decimal.Decimal, err error) {
	income, expense = decimal.Zero, decimal.Zero
	rules, err := s.queries.ListUserActiveRules(ctx, userID)
	if err != nil {
		return income, expense, fmt.Errorf("list active rules: %w", err)
	}
	if len(rules) == 0 {
		return income, expense, nil
	}
	writtenRows, err := s.queries.ListRuleEntryDates(ctx, db.ListRuleEntryDatesParams{
		UserID:    userID,
		StartDate: pgDate(window.start),
		EndDate:   pgDate(window.end),
	})
	if err != nil {
		return income, expense, fmt.Errorf("list rule entry dates: %w", err)
	}
	written := make(map[ruleEntryKey]struct{}, len(writtenRows))
	for _, row := range writtenRows {
		written[ruleEntryKey{ruleID: row.SourceRuleID, date: formatDate(row.EntryDate.Time)}] = struct{}{}
	}

	for _, rule := range rules {
		from := window.start
		if rule.MaterializedThrough.Valid {
			afterWatermark := dateOnly(rule.MaterializedThrough.Time).AddDate(0, 0, 1)
			if afterWatermark.After(from) {
				from = afterWatermark
			}
		}
		if from.After(window.end) {
			continue
		}
		var end *time.Time
		if rule.EndDate.Valid {
			end = &rule.EndDate.Time
		}
		for _, due := range materialize.OccurrencesBetween(rule.StartDate.Time, rule.RecurringType, end, from.AddDate(0, 0, -1), window.end) {
			if _, exists := written[ruleEntryKey{ruleID: rule.ID, date: formatDate(due)}]; exists {
				continue
			}
			conv, err := rates.Convert(ctx, s.rates, rule.Currency, due, rule.Amount)
			if err != nil {
				// Materialize can't write this entry either without a rate.
				slog.Warn("stats.PeriodBudget: skipped projected occurrence",
					"rule_id", rule.ID, "date", formatDate(due), "error", err)
				continue
			}
			switch rule.Kind {
			case "income":
				if isReimbursement(rule.CategoryID) {
					continue
				}
				income = income.Add(conv.AmountPHP)
			case "expense":
				if due.Before(today) {
					continue
				}
				expense = expense.Add(conv.AmountPHP)
			}
		}
	}
	return income, expense, nil
}

func netSpend(expenses, reimbursements decimal.Decimal) decimal.Decimal {
	return expenses.Sub(reimbursements)
}

func isReimbursement(categoryID pgtype.UUID) bool {
	return categoryID.Valid && uuid.UUID(categoryID.Bytes) == category.Reimbursement
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
		UserID:          userID,
		StartDate:       helper.ToPgDate(start),
		EndDate:         helper.ToPgDate(end),
		ReimbursementID: category.Reimbursement,
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
