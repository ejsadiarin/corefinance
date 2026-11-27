package mock

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

var _ db.Querier = (*MockQuerier)(nil)

type MockQuerier struct {
	ImportExpenseFn                       func(ctx context.Context, arg db.ImportExpenseParams) (int64, error)
	ImportIncomeFn                        func(ctx context.Context, arg db.ImportIncomeParams) (int64, error)
	ImportRuleFn                          func(ctx context.Context, arg db.ImportRuleParams) (int64, error)
	ImportTagFn                           func(ctx context.Context, arg db.ImportTagParams) (int64, error)
	ListAllRulesByUserFn                  func(ctx context.Context, userID uuid.UUID) ([]db.Rule, error)
	ListForeignBudgetIDsFn                func(ctx context.Context, arg db.ListForeignBudgetIDsParams) ([]db.ListForeignBudgetIDsRow, error)
	AddTagToExpenseFn                     func(ctx context.Context, arg db.AddTagToExpenseParams) error
	AddTagToIncomeFn                      func(ctx context.Context, arg db.AddTagToIncomeParams) error
	CheckSkippedExpenseFn                 func(ctx context.Context, arg db.CheckSkippedExpenseParams) (bool, error)
	CheckSkippedIncomeFn                  func(ctx context.Context, arg db.CheckSkippedIncomeParams) (bool, error)
	CreateExpenseFn                       func(ctx context.Context, arg db.CreateExpenseParams) (db.Expense, error)
	CreateExpenseInstanceFn               func(ctx context.Context, arg db.CreateExpenseInstanceParams) (int64, error)
	CreateIncomeFn                        func(ctx context.Context, arg db.CreateIncomeParams) (db.Income, error)
	CreateIncomeInstanceFn                func(ctx context.Context, arg db.CreateIncomeInstanceParams) (int64, error)
	CreateRecurringExpenseRuleFn          func(ctx context.Context, arg db.CreateRecurringExpenseRuleParams) (db.Rule, error)
	CreateRecurringIncomeRuleFn           func(ctx context.Context, arg db.CreateRecurringIncomeRuleParams) (db.Rule, error)
	CreateTagFn                           func(ctx context.Context, arg db.CreateTagParams) (db.Tag, error)
	DeleteExpenseFn                       func(ctx context.Context, arg db.DeleteExpenseParams) error
	DeleteIncomeFn                        func(ctx context.Context, arg db.DeleteIncomeParams) error
	DeleteRecurringExpenseRuleFn          func(ctx context.Context, arg db.DeleteRecurringExpenseRuleParams) error
	DeleteRecurringIncomeRuleFn           func(ctx context.Context, arg db.DeleteRecurringIncomeRuleParams) error
	DeleteTagFn                           func(ctx context.Context, arg db.DeleteTagParams) error
	GetCategoryBreakdownFn                func(ctx context.Context, arg db.GetCategoryBreakdownParams) ([]db.GetCategoryBreakdownRow, error)
	GetMoneyOnHandFn                      func(ctx context.Context, arg db.GetMoneyOnHandParams) (db.GetMoneyOnHandRow, error)
	GetExpenseFn                          func(ctx context.Context, arg db.GetExpenseParams) (db.GetExpenseRow, error)
	GetExpenseStatsByCategoryFn           func(ctx context.Context, arg db.GetExpenseStatsByCategoryParams) ([]db.GetExpenseStatsByCategoryRow, error)
	GetFiftyThirtyTwentyFn                func(ctx context.Context, arg db.GetFiftyThirtyTwentyParams) (db.GetFiftyThirtyTwentyRow, error)
	GetIncomeFn                           func(ctx context.Context, arg db.GetIncomeParams) (db.GetIncomeRow, error)
	ListRuleEntriesFn                     func(ctx context.Context, arg db.ListRuleEntriesParams) ([]db.ListRuleEntriesRow, error)
	ListCashFlowBucketsFn                 func(ctx context.Context, arg db.ListCashFlowBucketsParams) ([]db.ListCashFlowBucketsRow, error)
	ListRuleEntryDatesFn                  func(ctx context.Context, arg db.ListRuleEntryDatesParams) ([]db.ListRuleEntryDatesRow, error)
	GetFeedHorizonFn                      func(ctx context.Context) (uint64, error)
	ListRuleEntriesFeedFn                 func(ctx context.Context, arg db.ListRuleEntriesFeedParams) ([]db.ListRuleEntriesFeedRow, error)
	LockUserMaterializeFn                 func(ctx context.Context, userID uuid.UUID) error
	AdvanceRuleWatermarkFn                func(ctx context.Context, arg db.AdvanceRuleWatermarkParams) error
	ListRuleEntriesToReconcileFn          func(ctx context.Context, arg db.ListRuleEntriesToReconcileParams) ([]db.ListRuleEntriesToReconcileRow, error)
	GetSeenTimezoneFn                     func(ctx context.Context, userID uuid.UUID) (string, error)
	RecordUserTimezoneFn                  func(ctx context.Context, arg db.RecordUserTimezoneParams) (bool, error)
	SkipExpenseEntriesFn                  func(ctx context.Context, arg db.SkipExpenseEntriesParams) error
	SkipIncomeEntriesFn                   func(ctx context.Context, arg db.SkipIncomeEntriesParams) error
	MarkExpenseEntriesKeptFn              func(ctx context.Context, arg db.MarkExpenseEntriesKeptParams) error
	MarkIncomeEntriesKeptFn               func(ctx context.Context, arg db.MarkIncomeEntriesKeptParams) error
	SetRuleNextDueFn                      func(ctx context.Context, arg db.SetRuleNextDueParams) error
	ListUsersDueFn                        func(ctx context.Context, now pgtype.Timestamptz) ([]uuid.UUID, error)
	ListUserActiveRulesFn                 func(ctx context.Context, userID uuid.UUID) ([]db.Rule, error)
	GetNextWakeFn                         func(ctx context.Context, now pgtype.Timestamptz) (pgtype.Timestamptz, error)
	GetIncomeOccurrencesFn                func(ctx context.Context, arg db.GetIncomeOccurrencesParams) ([]db.Income, error)
	GetMonthOverMonthTrendsFn             func(ctx context.Context, arg db.GetMonthOverMonthTrendsParams) ([]db.GetMonthOverMonthTrendsRow, error)
	GetPeriodBudgetTotalsFn               func(ctx context.Context, arg db.GetPeriodBudgetTotalsParams) (db.GetPeriodBudgetTotalsRow, error)
	GetRecurringExpenseRuleFn             func(ctx context.Context, arg db.GetRecurringExpenseRuleParams) (db.GetRecurringExpenseRuleRow, error)
	GetRecurringIncomeRuleFn              func(ctx context.Context, arg db.GetRecurringIncomeRuleParams) (db.Rule, error)
	GetSavingsRateFn                      func(ctx context.Context, arg db.GetSavingsRateParams) (db.GetSavingsRateRow, error)
	GetSpendingVelocityFn                 func(ctx context.Context, arg db.GetSpendingVelocityParams) (db.GetSpendingVelocityRow, error)
	GetSummaryFn                          func(ctx context.Context, arg db.GetSummaryParams) (db.GetSummaryRow, error)
	GetTagFn                              func(ctx context.Context, arg db.GetTagParams) (db.Tag, error)
	GetTagsByExpenseIDFn                  func(ctx context.Context, expenseID uuid.UUID) ([]db.Tag, error)
	GetTagsByIncomeIDFn                   func(ctx context.Context, incomeID uuid.UUID) ([]db.Tag, error)
	GetTotalExpensesByDateRangeFn         func(ctx context.Context, arg db.GetTotalExpensesByDateRangeParams) (decimal.Decimal, error)
	GetTotalIncomesByDateRangeFn          func(ctx context.Context, arg db.GetTotalIncomesByDateRangeParams) (decimal.Decimal, error)
	GetTrendsFn                           func(ctx context.Context, arg db.GetTrendsParams) ([]db.GetTrendsRow, error)
	GetUpcomingRecurringExpensesFn        func(ctx context.Context, userID uuid.UUID) ([]db.GetUpcomingRecurringExpensesRow, error)
	ListAllExpenseTagsByUserFn            func(ctx context.Context, userID uuid.UUID) ([]db.ExpenseTag, error)
	ListAllExpensesByUserFn               func(ctx context.Context, userID uuid.UUID) ([]db.Expense, error)
	ListAllIncomeTagsByUserFn             func(ctx context.Context, userID uuid.UUID) ([]db.IncomeTag, error)
	ListAllIncomesByUserFn                func(ctx context.Context, userID uuid.UUID) ([]db.Income, error)
	ListExpenseCategoriesFn               func(ctx context.Context) ([]db.ExpenseCategory, error)
	ListExpensesFn                        func(ctx context.Context, arg db.ListExpensesParams) ([]db.ListExpensesRow, error)
	ListIncomeCategoriesFn                func(ctx context.Context) ([]db.IncomeCategory, error)
	ListIncomesFn                         func(ctx context.Context, arg db.ListIncomesParams) ([]db.ListIncomesRow, error)
	ListExpensesByIDsFn                   func(ctx context.Context, arg db.ListExpensesByIDsParams) ([]db.ListExpensesByIDsRow, error)
	ListIncomesByIDsFn                    func(ctx context.Context, arg db.ListIncomesByIDsParams) ([]db.ListIncomesByIDsRow, error)
	ListRecurringExpenseRulesFn           func(ctx context.Context, userID uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error)
	ListRecurringIncomeRulesFn            func(ctx context.Context, userID uuid.UUID) ([]db.Rule, error)
	ListRecurringExpenseRulesWithPausedFn func(ctx context.Context, userID uuid.UUID) ([]db.ListRecurringExpenseRulesWithPausedRow, error)
	ListRecurringIncomeRulesWithPausedFn  func(ctx context.Context, userID uuid.UUID) ([]db.Rule, error)
	ListTagsFn                            func(ctx context.Context, userID uuid.UUID) ([]db.Tag, error)
	RemoveTagFromExpenseFn                func(ctx context.Context, arg db.RemoveTagFromExpenseParams) error
	RemoveTagFromIncomeFn                 func(ctx context.Context, arg db.RemoveTagFromIncomeParams) error
	SearchExpensesFn                      func(ctx context.Context, arg db.SearchExpensesParams) ([]db.SearchExpensesRow, error)
	SkipExpenseFn                         func(ctx context.Context, arg db.SkipExpenseParams) error
	SkipIncomeFn                          func(ctx context.Context, arg db.SkipIncomeParams) error
	UpdateExpenseFn                       func(ctx context.Context, arg db.UpdateExpenseParams) (db.Expense, error)
	UpdateIncomeFn                        func(ctx context.Context, arg db.UpdateIncomeParams) (db.Income, error)
	UpdateRecurringExpenseRuleFn          func(ctx context.Context, arg db.UpdateRecurringExpenseRuleParams) (db.Rule, error)
	UpdateRecurringIncomeRuleFn           func(ctx context.Context, arg db.UpdateRecurringIncomeRuleParams) (db.Rule, error)
	UpdateTagFn                           func(ctx context.Context, arg db.UpdateTagParams) (db.Tag, error)
}

func NewMockQuerier() *MockQuerier {
	return &MockQuerier{}
}

func (m *MockQuerier) AddTagToExpense(ctx context.Context, arg db.AddTagToExpenseParams) error {
	return m.AddTagToExpenseFn(ctx, arg)
}

func (m *MockQuerier) AddTagToIncome(ctx context.Context, arg db.AddTagToIncomeParams) error {
	return m.AddTagToIncomeFn(ctx, arg)
}

func (m *MockQuerier) CheckSkippedExpense(ctx context.Context, arg db.CheckSkippedExpenseParams) (bool, error) {
	return m.CheckSkippedExpenseFn(ctx, arg)
}

func (m *MockQuerier) CheckSkippedIncome(ctx context.Context, arg db.CheckSkippedIncomeParams) (bool, error) {
	return m.CheckSkippedIncomeFn(ctx, arg)
}

func (m *MockQuerier) CreateExpense(ctx context.Context, arg db.CreateExpenseParams) (db.Expense, error) {
	return m.CreateExpenseFn(ctx, arg)
}

func (m *MockQuerier) CreateExpenseInstance(ctx context.Context, arg db.CreateExpenseInstanceParams) (int64, error) {
	return m.CreateExpenseInstanceFn(ctx, arg)
}

func (m *MockQuerier) CreateIncome(ctx context.Context, arg db.CreateIncomeParams) (db.Income, error) {
	return m.CreateIncomeFn(ctx, arg)
}

func (m *MockQuerier) CreateIncomeInstance(ctx context.Context, arg db.CreateIncomeInstanceParams) (int64, error) {
	return m.CreateIncomeInstanceFn(ctx, arg)
}

func (m *MockQuerier) CreateRecurringExpenseRule(ctx context.Context, arg db.CreateRecurringExpenseRuleParams) (db.Rule, error) {
	return m.CreateRecurringExpenseRuleFn(ctx, arg)
}

func (m *MockQuerier) CreateRecurringIncomeRule(ctx context.Context, arg db.CreateRecurringIncomeRuleParams) (db.Rule, error) {
	return m.CreateRecurringIncomeRuleFn(ctx, arg)
}

func (m *MockQuerier) CreateTag(ctx context.Context, arg db.CreateTagParams) (db.Tag, error) {
	return m.CreateTagFn(ctx, arg)
}

func (m *MockQuerier) DeleteExpense(ctx context.Context, arg db.DeleteExpenseParams) error {
	return m.DeleteExpenseFn(ctx, arg)
}

func (m *MockQuerier) DeleteIncome(ctx context.Context, arg db.DeleteIncomeParams) error {
	return m.DeleteIncomeFn(ctx, arg)
}

func (m *MockQuerier) DeleteRecurringExpenseRule(ctx context.Context, arg db.DeleteRecurringExpenseRuleParams) error {
	return m.DeleteRecurringExpenseRuleFn(ctx, arg)
}

func (m *MockQuerier) DeleteRecurringIncomeRule(ctx context.Context, arg db.DeleteRecurringIncomeRuleParams) error {
	return m.DeleteRecurringIncomeRuleFn(ctx, arg)
}

func (m *MockQuerier) DeleteTag(ctx context.Context, arg db.DeleteTagParams) error {
	return m.DeleteTagFn(ctx, arg)
}

func (m *MockQuerier) GetCategoryBreakdown(ctx context.Context, arg db.GetCategoryBreakdownParams) ([]db.GetCategoryBreakdownRow, error) {
	return m.GetCategoryBreakdownFn(ctx, arg)
}

func (m *MockQuerier) GetMoneyOnHand(ctx context.Context, arg db.GetMoneyOnHandParams) (db.GetMoneyOnHandRow, error) {
	return m.GetMoneyOnHandFn(ctx, arg)
}

func (m *MockQuerier) GetExpense(ctx context.Context, arg db.GetExpenseParams) (db.GetExpenseRow, error) {
	return m.GetExpenseFn(ctx, arg)
}

func (m *MockQuerier) GetExpenseStatsByCategory(ctx context.Context, arg db.GetExpenseStatsByCategoryParams) ([]db.GetExpenseStatsByCategoryRow, error) {
	return m.GetExpenseStatsByCategoryFn(ctx, arg)
}

func (m *MockQuerier) GetFiftyThirtyTwenty(ctx context.Context, arg db.GetFiftyThirtyTwentyParams) (db.GetFiftyThirtyTwentyRow, error) {
	return m.GetFiftyThirtyTwentyFn(ctx, arg)
}

func (m *MockQuerier) GetIncome(ctx context.Context, arg db.GetIncomeParams) (db.GetIncomeRow, error) {
	return m.GetIncomeFn(ctx, arg)
}

func (m *MockQuerier) GetIncomeOccurrences(ctx context.Context, arg db.GetIncomeOccurrencesParams) ([]db.Income, error) {
	return m.GetIncomeOccurrencesFn(ctx, arg)
}

func (m *MockQuerier) GetMonthOverMonthTrends(ctx context.Context, arg db.GetMonthOverMonthTrendsParams) ([]db.GetMonthOverMonthTrendsRow, error) {
	return m.GetMonthOverMonthTrendsFn(ctx, arg)
}

func (m *MockQuerier) GetPeriodBudgetTotals(ctx context.Context, arg db.GetPeriodBudgetTotalsParams) (db.GetPeriodBudgetTotalsRow, error) {
	return m.GetPeriodBudgetTotalsFn(ctx, arg)
}

// GetRecurringExpenseRule defaults to an open-ended rule, which a rule
// update reads to compare schedules (#15).
func (m *MockQuerier) GetRecurringExpenseRule(ctx context.Context, arg db.GetRecurringExpenseRuleParams) (db.GetRecurringExpenseRuleRow, error) {
	if m.GetRecurringExpenseRuleFn == nil {
		return db.GetRecurringExpenseRuleRow{ID: arg.ID, UserID: arg.UserID}, nil
	}
	return m.GetRecurringExpenseRuleFn(ctx, arg)
}

// GetRecurringIncomeRule defaults to an open-ended rule (see
// GetRecurringExpenseRule).
func (m *MockQuerier) GetRecurringIncomeRule(ctx context.Context, arg db.GetRecurringIncomeRuleParams) (db.Rule, error) {
	if m.GetRecurringIncomeRuleFn == nil {
		return db.Rule{ID: arg.ID, UserID: arg.UserID}, nil
	}
	return m.GetRecurringIncomeRuleFn(ctx, arg)
}

func (m *MockQuerier) GetSavingsRate(ctx context.Context, arg db.GetSavingsRateParams) (db.GetSavingsRateRow, error) {
	return m.GetSavingsRateFn(ctx, arg)
}

func (m *MockQuerier) GetSpendingVelocity(ctx context.Context, arg db.GetSpendingVelocityParams) (db.GetSpendingVelocityRow, error) {
	return m.GetSpendingVelocityFn(ctx, arg)
}

func (m *MockQuerier) GetSummary(ctx context.Context, arg db.GetSummaryParams) (db.GetSummaryRow, error) {
	return m.GetSummaryFn(ctx, arg)
}

func (m *MockQuerier) GetTag(ctx context.Context, arg db.GetTagParams) (db.Tag, error) {
	return m.GetTagFn(ctx, arg)
}

func (m *MockQuerier) GetTagsByExpenseID(ctx context.Context, expenseID uuid.UUID) ([]db.Tag, error) {
	return m.GetTagsByExpenseIDFn(ctx, expenseID)
}

func (m *MockQuerier) GetTagsByIncomeID(ctx context.Context, incomeID uuid.UUID) ([]db.Tag, error) {
	return m.GetTagsByIncomeIDFn(ctx, incomeID)
}

func (m *MockQuerier) GetTotalExpensesByDateRange(ctx context.Context, arg db.GetTotalExpensesByDateRangeParams) (decimal.Decimal, error) {
	return m.GetTotalExpensesByDateRangeFn(ctx, arg)
}

func (m *MockQuerier) GetTotalIncomesByDateRange(ctx context.Context, arg db.GetTotalIncomesByDateRangeParams) (decimal.Decimal, error) {
	return m.GetTotalIncomesByDateRangeFn(ctx, arg)
}

func (m *MockQuerier) GetTrends(ctx context.Context, arg db.GetTrendsParams) ([]db.GetTrendsRow, error) {
	return m.GetTrendsFn(ctx, arg)
}

func (m *MockQuerier) GetUpcomingRecurringExpenses(ctx context.Context, userID uuid.UUID) ([]db.GetUpcomingRecurringExpensesRow, error) {
	return m.GetUpcomingRecurringExpensesFn(ctx, userID)
}

func (m *MockQuerier) ListAllExpenseTagsByUser(ctx context.Context, userID uuid.UUID) ([]db.ExpenseTag, error) {
	return m.ListAllExpenseTagsByUserFn(ctx, userID)
}

func (m *MockQuerier) ListAllExpensesByUser(ctx context.Context, userID uuid.UUID) ([]db.Expense, error) {
	return m.ListAllExpensesByUserFn(ctx, userID)
}

func (m *MockQuerier) ListAllIncomeTagsByUser(ctx context.Context, userID uuid.UUID) ([]db.IncomeTag, error) {
	return m.ListAllIncomeTagsByUserFn(ctx, userID)
}

func (m *MockQuerier) ListAllIncomesByUser(ctx context.Context, userID uuid.UUID) ([]db.Income, error) {
	return m.ListAllIncomesByUserFn(ctx, userID)
}

func (m *MockQuerier) ListExpenseCategories(ctx context.Context) ([]db.ExpenseCategory, error) {
	return m.ListExpenseCategoriesFn(ctx)
}

func (m *MockQuerier) ListExpenses(ctx context.Context, arg db.ListExpensesParams) ([]db.ListExpensesRow, error) {
	return m.ListExpensesFn(ctx, arg)
}

func (m *MockQuerier) ListExpensesByIDs(ctx context.Context, arg db.ListExpensesByIDsParams) ([]db.ListExpensesByIDsRow, error) {
	return m.ListExpensesByIDsFn(ctx, arg)
}

func (m *MockQuerier) ListIncomesByIDs(ctx context.Context, arg db.ListIncomesByIDsParams) ([]db.ListIncomesByIDsRow, error) {
	return m.ListIncomesByIDsFn(ctx, arg)
}

func (m *MockQuerier) ListIncomeCategories(ctx context.Context) ([]db.IncomeCategory, error) {
	return m.ListIncomeCategoriesFn(ctx)
}

func (m *MockQuerier) ListIncomes(ctx context.Context, arg db.ListIncomesParams) ([]db.ListIncomesRow, error) {
	return m.ListIncomesFn(ctx, arg)
}

func (m *MockQuerier) ListRecurringExpenseRules(ctx context.Context, userID uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
	return m.ListRecurringExpenseRulesFn(ctx, userID)
}

func (m *MockQuerier) ListRecurringIncomeRules(ctx context.Context, userID uuid.UUID) ([]db.Rule, error) {
	return m.ListRecurringIncomeRulesFn(ctx, userID)
}

func (m *MockQuerier) ListRecurringExpenseRulesWithPaused(ctx context.Context, userID uuid.UUID) ([]db.ListRecurringExpenseRulesWithPausedRow, error) {
	return m.ListRecurringExpenseRulesWithPausedFn(ctx, userID)
}

func (m *MockQuerier) ListRecurringIncomeRulesWithPaused(ctx context.Context, userID uuid.UUID) ([]db.Rule, error) {
	return m.ListRecurringIncomeRulesWithPausedFn(ctx, userID)
}

func (m *MockQuerier) ListTags(ctx context.Context, userID uuid.UUID) ([]db.Tag, error) {
	return m.ListTagsFn(ctx, userID)
}

func (m *MockQuerier) RemoveTagFromExpense(ctx context.Context, arg db.RemoveTagFromExpenseParams) error {
	return m.RemoveTagFromExpenseFn(ctx, arg)
}

func (m *MockQuerier) RemoveTagFromIncome(ctx context.Context, arg db.RemoveTagFromIncomeParams) error {
	return m.RemoveTagFromIncomeFn(ctx, arg)
}

func (m *MockQuerier) SearchExpenses(ctx context.Context, arg db.SearchExpensesParams) ([]db.SearchExpensesRow, error) {
	return m.SearchExpensesFn(ctx, arg)
}

func (m *MockQuerier) SkipExpense(ctx context.Context, arg db.SkipExpenseParams) error {
	return m.SkipExpenseFn(ctx, arg)
}

func (m *MockQuerier) SkipIncome(ctx context.Context, arg db.SkipIncomeParams) error {
	return m.SkipIncomeFn(ctx, arg)
}

func (m *MockQuerier) UpdateExpense(ctx context.Context, arg db.UpdateExpenseParams) (db.Expense, error) {
	return m.UpdateExpenseFn(ctx, arg)
}

func (m *MockQuerier) UpdateIncome(ctx context.Context, arg db.UpdateIncomeParams) (db.Income, error) {
	return m.UpdateIncomeFn(ctx, arg)
}

func (m *MockQuerier) UpdateRecurringExpenseRule(ctx context.Context, arg db.UpdateRecurringExpenseRuleParams) (db.Rule, error) {
	return m.UpdateRecurringExpenseRuleFn(ctx, arg)
}

func (m *MockQuerier) UpdateRecurringIncomeRule(ctx context.Context, arg db.UpdateRecurringIncomeRuleParams) (db.Rule, error) {
	return m.UpdateRecurringIncomeRuleFn(ctx, arg)
}

func (m *MockQuerier) UpdateTag(ctx context.Context, arg db.UpdateTagParams) (db.Tag, error) {
	return m.UpdateTagFn(ctx, arg)
}

// ListRuleEntries defaults to no entries.
func (m *MockQuerier) ListRuleEntries(ctx context.Context, arg db.ListRuleEntriesParams) ([]db.ListRuleEntriesRow, error) {
	if m.ListRuleEntriesFn == nil {
		return nil, nil
	}
	return m.ListRuleEntriesFn(ctx, arg)
}

// GetFeedHorizon defaults to 0: nothing is past the horizon.
func (m *MockQuerier) GetFeedHorizon(ctx context.Context) (uint64, error) {
	if m.GetFeedHorizonFn == nil {
		return 0, nil
	}
	return m.GetFeedHorizonFn(ctx)
}

// ListCashFlowBuckets defaults to no buckets.
func (m *MockQuerier) ListCashFlowBuckets(ctx context.Context, arg db.ListCashFlowBucketsParams) ([]db.ListCashFlowBucketsRow, error) {
	if m.ListCashFlowBucketsFn == nil {
		return nil, nil
	}
	return m.ListCashFlowBucketsFn(ctx, arg)
}

// ListRuleEntryDates defaults to no entries.
func (m *MockQuerier) ListRuleEntryDates(ctx context.Context, arg db.ListRuleEntryDatesParams) ([]db.ListRuleEntryDatesRow, error) {
	if m.ListRuleEntryDatesFn == nil {
		return nil, nil
	}
	return m.ListRuleEntryDatesFn(ctx, arg)
}

// ListRuleEntriesFeed defaults to no entries.
func (m *MockQuerier) ListRuleEntriesFeed(ctx context.Context, arg db.ListRuleEntriesFeedParams) ([]db.ListRuleEntriesFeedRow, error) {
	if m.ListRuleEntriesFeedFn == nil {
		return nil, nil
	}
	return m.ListRuleEntriesFeedFn(ctx, arg)
}

// LockUserMaterialize defaults to taking the lock.
func (m *MockQuerier) LockUserMaterialize(ctx context.Context, userID uuid.UUID) error {
	if m.LockUserMaterializeFn == nil {
		return nil
	}
	return m.LockUserMaterializeFn(ctx, userID)
}

// AdvanceRuleWatermark defaults to succeeding.
func (m *MockQuerier) AdvanceRuleWatermark(ctx context.Context, arg db.AdvanceRuleWatermarkParams) error {
	if m.AdvanceRuleWatermarkFn == nil {
		return nil
	}
	return m.AdvanceRuleWatermarkFn(ctx, arg)
}

// ListRuleEntriesToReconcile defaults to no entries.
func (m *MockQuerier) ListRuleEntriesToReconcile(ctx context.Context, arg db.ListRuleEntriesToReconcileParams) ([]db.ListRuleEntriesToReconcileRow, error) {
	if m.ListRuleEntriesToReconcileFn == nil {
		return nil, nil
	}
	return m.ListRuleEntriesToReconcileFn(ctx, arg)
}

// SkipExpenseEntries defaults to succeeding.
func (m *MockQuerier) SkipExpenseEntries(ctx context.Context, arg db.SkipExpenseEntriesParams) error {
	if m.SkipExpenseEntriesFn == nil {
		return nil
	}
	return m.SkipExpenseEntriesFn(ctx, arg)
}

// SkipIncomeEntries defaults to succeeding.
func (m *MockQuerier) SkipIncomeEntries(ctx context.Context, arg db.SkipIncomeEntriesParams) error {
	if m.SkipIncomeEntriesFn == nil {
		return nil
	}
	return m.SkipIncomeEntriesFn(ctx, arg)
}

// MarkExpenseEntriesKept defaults to succeeding.
func (m *MockQuerier) MarkExpenseEntriesKept(ctx context.Context, arg db.MarkExpenseEntriesKeptParams) error {
	if m.MarkExpenseEntriesKeptFn == nil {
		return nil
	}
	return m.MarkExpenseEntriesKeptFn(ctx, arg)
}

// MarkIncomeEntriesKept defaults to succeeding.
func (m *MockQuerier) MarkIncomeEntriesKept(ctx context.Context, arg db.MarkIncomeEntriesKeptParams) error {
	if m.MarkIncomeEntriesKeptFn == nil {
		return nil
	}
	return m.MarkIncomeEntriesKeptFn(ctx, arg)
}

// ListUsersDue defaults to nobody.
func (m *MockQuerier) ListUsersDue(ctx context.Context, now pgtype.Timestamptz) ([]uuid.UUID, error) {
	if m.ListUsersDueFn == nil {
		return nil, nil
	}
	return m.ListUsersDueFn(ctx, now)
}

// ListUserActiveRules defaults to no rules.
func (m *MockQuerier) ListUserActiveRules(ctx context.Context, userID uuid.UUID) ([]db.Rule, error) {
	if m.ListUserActiveRulesFn == nil {
		return nil, nil
	}
	return m.ListUserActiveRulesFn(ctx, userID)
}

// GetNextWake defaults to nothing due.
func (m *MockQuerier) GetNextWake(ctx context.Context, now pgtype.Timestamptz) (pgtype.Timestamptz, error) {
	if m.GetNextWakeFn == nil {
		return pgtype.Timestamptz{}, nil
	}
	return m.GetNextWakeFn(ctx, now)
}

// SetRuleNextDue defaults to succeeding.
func (m *MockQuerier) SetRuleNextDue(ctx context.Context, arg db.SetRuleNextDueParams) error {
	if m.SetRuleNextDueFn == nil {
		return nil
	}
	return m.SetRuleNextDueFn(ctx, arg)
}

// GetSeenTimezone defaults to "never seen".
func (m *MockQuerier) GetSeenTimezone(ctx context.Context, userID uuid.UUID) (string, error) {
	if m.GetSeenTimezoneFn == nil {
		return "", pgx.ErrNoRows
	}
	return m.GetSeenTimezoneFn(ctx, userID)
}

// RecordUserTimezone defaults to recording an unchanged zone.
func (m *MockQuerier) RecordUserTimezone(ctx context.Context, arg db.RecordUserTimezoneParams) (bool, error) {
	if m.RecordUserTimezoneFn == nil {
		return false, nil
	}
	return m.RecordUserTimezoneFn(ctx, arg)
}

func (m *MockQuerier) ImportExpense(ctx context.Context, arg db.ImportExpenseParams) (int64, error) {
	return m.ImportExpenseFn(ctx, arg)
}

func (m *MockQuerier) ImportIncome(ctx context.Context, arg db.ImportIncomeParams) (int64, error) {
	return m.ImportIncomeFn(ctx, arg)
}

func (m *MockQuerier) ImportRule(ctx context.Context, arg db.ImportRuleParams) (int64, error) {
	return m.ImportRuleFn(ctx, arg)
}

func (m *MockQuerier) ImportTag(ctx context.Context, arg db.ImportTagParams) (int64, error) {
	return m.ImportTagFn(ctx, arg)
}

func (m *MockQuerier) ListAllRulesByUser(ctx context.Context, userID uuid.UUID) ([]db.Rule, error) {
	return m.ListAllRulesByUserFn(ctx, userID)
}

func (m *MockQuerier) ListForeignBudgetIDs(ctx context.Context, arg db.ListForeignBudgetIDsParams) ([]db.ListForeignBudgetIDsRow, error) {
	return m.ListForeignBudgetIDsFn(ctx, arg)
}
