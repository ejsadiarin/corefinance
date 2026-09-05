package mock

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

var _ db.Querier = (*MockQuerier)(nil)

type MockQuerier struct {
	AddTagToExpenseFn               func(ctx context.Context, arg db.AddTagToExpenseParams) error
	AddTagToIncomeFn                func(ctx context.Context, arg db.AddTagToIncomeParams) error
	CheckSkippedExpenseFn           func(ctx context.Context, arg db.CheckSkippedExpenseParams) (bool, error)
	CheckSkippedIncomeFn            func(ctx context.Context, arg db.CheckSkippedIncomeParams) (bool, error)
	CreateExpenseFn                 func(ctx context.Context, arg db.CreateExpenseParams) (db.Expense, error)
	CreateExpenseCategoryFn         func(ctx context.Context, arg db.CreateExpenseCategoryParams) (db.ExpenseCategory, error)
	CreateIncomeFn                  func(ctx context.Context, arg db.CreateIncomeParams) (db.Income, error)
	CreateIncomeCategoryFn          func(ctx context.Context, arg db.CreateIncomeCategoryParams) (db.IncomeCategory, error)
	CreateRecurringExpenseRuleFn    func(ctx context.Context, arg db.CreateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error)
	CreateRecurringIncomeRuleFn     func(ctx context.Context, arg db.CreateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error)
	CreateTagFn                     func(ctx context.Context, arg db.CreateTagParams) (db.Tag, error)
	DeleteExpenseFn                 func(ctx context.Context, arg db.DeleteExpenseParams) error
	DeleteExpenseCategoryFn         func(ctx context.Context, arg db.DeleteExpenseCategoryParams) error
	DeleteIncomeFn                  func(ctx context.Context, arg db.DeleteIncomeParams) error
	DeleteIncomeCategoryFn          func(ctx context.Context, arg db.DeleteIncomeCategoryParams) error
	DeleteRecurringExpenseRuleFn    func(ctx context.Context, arg db.DeleteRecurringExpenseRuleParams) error
	DeleteRecurringIncomeRuleFn     func(ctx context.Context, arg db.DeleteRecurringIncomeRuleParams) error
	DeleteTagFn                     func(ctx context.Context, arg db.DeleteTagParams) error
	GetCategoryBreakdownFn          func(ctx context.Context, arg db.GetCategoryBreakdownParams) ([]db.GetCategoryBreakdownRow, error)
	GetCurrentTotalMoneyFn          func(ctx context.Context, arg db.GetCurrentTotalMoneyParams) (decimal.Decimal, error)
	GetExpenseFn                    func(ctx context.Context, arg db.GetExpenseParams) (db.GetExpenseRow, error)
	GetExpenseCategoryFn            func(ctx context.Context, arg db.GetExpenseCategoryParams) (db.ExpenseCategory, error)
	GetExpenseStatsByCategoryFn     func(ctx context.Context, arg db.GetExpenseStatsByCategoryParams) ([]db.GetExpenseStatsByCategoryRow, error)
	GetFiftyThirtyTwentyFn          func(ctx context.Context, arg db.GetFiftyThirtyTwentyParams) (db.GetFiftyThirtyTwentyRow, error)
	GetIncomeFn                     func(ctx context.Context, arg db.GetIncomeParams) (db.GetIncomeRow, error)
	GetIncomeCategoryFn             func(ctx context.Context, arg db.GetIncomeCategoryParams) (db.IncomeCategory, error)
	GetIncomeOccurrencesFn          func(ctx context.Context, arg db.GetIncomeOccurrencesParams) ([]db.Income, error)
	GetMonthOverMonthTrendsFn       func(ctx context.Context, arg db.GetMonthOverMonthTrendsParams) ([]db.GetMonthOverMonthTrendsRow, error)
	GetRecurringExpenseRuleFn       func(ctx context.Context, arg db.GetRecurringExpenseRuleParams) (db.GetRecurringExpenseRuleRow, error)
	GetRecurringIncomeRuleFn        func(ctx context.Context, arg db.GetRecurringIncomeRuleParams) (db.RecurringIncomeRule, error)
	GetSavingsRateFn                func(ctx context.Context, arg db.GetSavingsRateParams) (db.GetSavingsRateRow, error)
	GetSpendingVelocityFn           func(ctx context.Context, arg db.GetSpendingVelocityParams) (db.GetSpendingVelocityRow, error)
	GetSummaryFn                    func(ctx context.Context, arg db.GetSummaryParams) (db.GetSummaryRow, error)
	GetTagFn                        func(ctx context.Context, arg db.GetTagParams) (db.Tag, error)
	GetTagsByExpenseIDFn            func(ctx context.Context, expenseID uuid.UUID) ([]db.Tag, error)
	GetTagsByIncomeIDFn             func(ctx context.Context, incomeID uuid.UUID) ([]db.Tag, error)
	GetTotalExpensesByDateRangeFn   func(ctx context.Context, arg db.GetTotalExpensesByDateRangeParams) (decimal.Decimal, error)
	GetTotalIncomesByDateRangeFn    func(ctx context.Context, arg db.GetTotalIncomesByDateRangeParams) (decimal.Decimal, error)
	GetTrendsFn                     func(ctx context.Context, arg db.GetTrendsParams) ([]db.GetTrendsRow, error)
	GetUpcomingRecurringExpensesFn  func(ctx context.Context, userID uuid.UUID) ([]db.GetUpcomingRecurringExpensesRow, error)
	ListAllExpenseTagsByUserFn      func(ctx context.Context, userID uuid.UUID) ([]db.ExpenseTag, error)
	ListAllExpensesByUserFn         func(ctx context.Context, userID uuid.UUID) ([]db.Expense, error)
	ListAllIncomeTagsByUserFn       func(ctx context.Context, userID uuid.UUID) ([]db.IncomeTag, error)
	ListAllIncomesByUserFn          func(ctx context.Context, userID uuid.UUID) ([]db.Income, error)
	ListExpenseCategoriesFn         func(ctx context.Context, userID uuid.UUID) ([]db.ExpenseCategory, error)
	ListExpensesFn                  func(ctx context.Context, arg db.ListExpensesParams) ([]db.ListExpensesRow, error)
	ListIncomeCategoriesFn          func(ctx context.Context, userID uuid.UUID) ([]db.IncomeCategory, error)
	ListIncomesFn                   func(ctx context.Context, arg db.ListIncomesParams) ([]db.ListIncomesRow, error)
	ListRecurringExpenseRulesFn     func(ctx context.Context, userID uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error)
	ListRecurringIncomeRulesFn      func(ctx context.Context, userID uuid.UUID) ([]db.RecurringIncomeRule, error)
	ListTagsFn                      func(ctx context.Context, userID uuid.UUID) ([]db.Tag, error)
	RemoveTagFromExpenseFn          func(ctx context.Context, arg db.RemoveTagFromExpenseParams) error
	RemoveTagFromIncomeFn           func(ctx context.Context, arg db.RemoveTagFromIncomeParams) error
	SearchExpensesFn                func(ctx context.Context, arg db.SearchExpensesParams) ([]db.SearchExpensesRow, error)
	SkipExpenseFn                   func(ctx context.Context, arg db.SkipExpenseParams) error
	SkipIncomeFn                    func(ctx context.Context, arg db.SkipIncomeParams) error
	UpdateExpenseFn                 func(ctx context.Context, arg db.UpdateExpenseParams) (db.Expense, error)
	UpdateExpenseCategoryFn         func(ctx context.Context, arg db.UpdateExpenseCategoryParams) (db.ExpenseCategory, error)
	UpdateIncomeFn                  func(ctx context.Context, arg db.UpdateIncomeParams) (db.Income, error)
	UpdateIncomeCategoryFn          func(ctx context.Context, arg db.UpdateIncomeCategoryParams) (db.IncomeCategory, error)
	UpdateRecurringExpenseRuleFn    func(ctx context.Context, arg db.UpdateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error)
	UpdateRecurringIncomeRuleFn     func(ctx context.Context, arg db.UpdateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error)
	UpdateTagFn                     func(ctx context.Context, arg db.UpdateTagParams) (db.Tag, error)
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

func (m *MockQuerier) CreateExpenseCategory(ctx context.Context, arg db.CreateExpenseCategoryParams) (db.ExpenseCategory, error) {
	return m.CreateExpenseCategoryFn(ctx, arg)
}

func (m *MockQuerier) CreateIncome(ctx context.Context, arg db.CreateIncomeParams) (db.Income, error) {
	return m.CreateIncomeFn(ctx, arg)
}

func (m *MockQuerier) CreateIncomeCategory(ctx context.Context, arg db.CreateIncomeCategoryParams) (db.IncomeCategory, error) {
	return m.CreateIncomeCategoryFn(ctx, arg)
}

func (m *MockQuerier) CreateRecurringExpenseRule(ctx context.Context, arg db.CreateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error) {
	return m.CreateRecurringExpenseRuleFn(ctx, arg)
}

func (m *MockQuerier) CreateRecurringIncomeRule(ctx context.Context, arg db.CreateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
	return m.CreateRecurringIncomeRuleFn(ctx, arg)
}

func (m *MockQuerier) CreateTag(ctx context.Context, arg db.CreateTagParams) (db.Tag, error) {
	return m.CreateTagFn(ctx, arg)
}

func (m *MockQuerier) DeleteExpense(ctx context.Context, arg db.DeleteExpenseParams) error {
	return m.DeleteExpenseFn(ctx, arg)
}

func (m *MockQuerier) DeleteExpenseCategory(ctx context.Context, arg db.DeleteExpenseCategoryParams) error {
	return m.DeleteExpenseCategoryFn(ctx, arg)
}

func (m *MockQuerier) DeleteIncome(ctx context.Context, arg db.DeleteIncomeParams) error {
	return m.DeleteIncomeFn(ctx, arg)
}

func (m *MockQuerier) DeleteIncomeCategory(ctx context.Context, arg db.DeleteIncomeCategoryParams) error {
	return m.DeleteIncomeCategoryFn(ctx, arg)
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

func (m *MockQuerier) GetCurrentTotalMoney(ctx context.Context, arg db.GetCurrentTotalMoneyParams) (decimal.Decimal, error) {
	return m.GetCurrentTotalMoneyFn(ctx, arg)
}

func (m *MockQuerier) GetExpense(ctx context.Context, arg db.GetExpenseParams) (db.GetExpenseRow, error) {
	return m.GetExpenseFn(ctx, arg)
}

func (m *MockQuerier) GetExpenseCategory(ctx context.Context, arg db.GetExpenseCategoryParams) (db.ExpenseCategory, error) {
	return m.GetExpenseCategoryFn(ctx, arg)
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

func (m *MockQuerier) GetIncomeCategory(ctx context.Context, arg db.GetIncomeCategoryParams) (db.IncomeCategory, error) {
	return m.GetIncomeCategoryFn(ctx, arg)
}

func (m *MockQuerier) GetIncomeOccurrences(ctx context.Context, arg db.GetIncomeOccurrencesParams) ([]db.Income, error) {
	return m.GetIncomeOccurrencesFn(ctx, arg)
}

func (m *MockQuerier) GetMonthOverMonthTrends(ctx context.Context, arg db.GetMonthOverMonthTrendsParams) ([]db.GetMonthOverMonthTrendsRow, error) {
	return m.GetMonthOverMonthTrendsFn(ctx, arg)
}

func (m *MockQuerier) GetRecurringExpenseRule(ctx context.Context, arg db.GetRecurringExpenseRuleParams) (db.GetRecurringExpenseRuleRow, error) {
	return m.GetRecurringExpenseRuleFn(ctx, arg)
}

func (m *MockQuerier) GetRecurringIncomeRule(ctx context.Context, arg db.GetRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
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

func (m *MockQuerier) ListExpenseCategories(ctx context.Context, userID uuid.UUID) ([]db.ExpenseCategory, error) {
	return m.ListExpenseCategoriesFn(ctx, userID)
}

func (m *MockQuerier) ListExpenses(ctx context.Context, arg db.ListExpensesParams) ([]db.ListExpensesRow, error) {
	return m.ListExpensesFn(ctx, arg)
}

func (m *MockQuerier) ListIncomeCategories(ctx context.Context, userID uuid.UUID) ([]db.IncomeCategory, error) {
	return m.ListIncomeCategoriesFn(ctx, userID)
}

func (m *MockQuerier) ListIncomes(ctx context.Context, arg db.ListIncomesParams) ([]db.ListIncomesRow, error) {
	return m.ListIncomesFn(ctx, arg)
}

func (m *MockQuerier) ListRecurringExpenseRules(ctx context.Context, userID uuid.UUID) ([]db.ListRecurringExpenseRulesRow, error) {
	return m.ListRecurringExpenseRulesFn(ctx, userID)
}

func (m *MockQuerier) ListRecurringIncomeRules(ctx context.Context, userID uuid.UUID) ([]db.RecurringIncomeRule, error) {
	return m.ListRecurringIncomeRulesFn(ctx, userID)
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

func (m *MockQuerier) UpdateExpenseCategory(ctx context.Context, arg db.UpdateExpenseCategoryParams) (db.ExpenseCategory, error) {
	return m.UpdateExpenseCategoryFn(ctx, arg)
}

func (m *MockQuerier) UpdateIncome(ctx context.Context, arg db.UpdateIncomeParams) (db.Income, error) {
	return m.UpdateIncomeFn(ctx, arg)
}

func (m *MockQuerier) UpdateIncomeCategory(ctx context.Context, arg db.UpdateIncomeCategoryParams) (db.IncomeCategory, error) {
	return m.UpdateIncomeCategoryFn(ctx, arg)
}

func (m *MockQuerier) UpdateRecurringExpenseRule(ctx context.Context, arg db.UpdateRecurringExpenseRuleParams) (db.RecurringExpenseRule, error) {
	return m.UpdateRecurringExpenseRuleFn(ctx, arg)
}

func (m *MockQuerier) UpdateRecurringIncomeRule(ctx context.Context, arg db.UpdateRecurringIncomeRuleParams) (db.RecurringIncomeRule, error) {
	return m.UpdateRecurringIncomeRuleFn(ctx, arg)
}

func (m *MockQuerier) UpdateTag(ctx context.Context, arg db.UpdateTagParams) (db.Tag, error) {
	return m.UpdateTagFn(ctx, arg)
}
