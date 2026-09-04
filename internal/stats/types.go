package stats

import (
	"github.com/shopspring/decimal"
)

type Summary struct {
	TotalExpenses decimal.Decimal `json:"total_expenses"`
	TotalIncomes  decimal.Decimal `json:"total_incomes"`
	ExpenseCount  int64           `json:"expense_count"`
	IncomeCount   int64           `json:"income_count"`
}

type Trend struct {
	Month         string          `json:"month"`
	TotalExpenses decimal.Decimal `json:"total_expenses"`
	TotalIncomes  decimal.Decimal `json:"total_incomes"`
}

type CategoryBreakdownItem struct {
	CategoryID    string          `json:"category_id"`
	CategoryName  string          `json:"category_name"`
	CategoryColor *string         `json:"category_color,omitempty"`
	ExpenseCount  int64           `json:"expense_count"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
}

type SavingsRate struct {
	TotalIncomes  decimal.Decimal `json:"total_incomes"`
	TotalExpenses decimal.Decimal `json:"total_expenses"`
	SavingsRate   decimal.Decimal `json:"savings_rate"`
}

type FiftyThirtyTwenty struct {
	TotalIncomes  decimal.Decimal `json:"total_incomes"`
	TotalExpenses decimal.Decimal `json:"total_expenses"`
	Needs         decimal.Decimal `json:"needs"`
	Wants         decimal.Decimal `json:"wants"`
	Savings       decimal.Decimal `json:"savings"`
	NeedsPct      decimal.Decimal `json:"needs_pct"`
	WantsPct      decimal.Decimal `json:"wants_pct"`
	SavingsPct    decimal.Decimal `json:"savings_pct"`
}

type SpendingVelocity struct {
	AvgMonthlySpending decimal.Decimal `json:"avg_monthly_spending"`
	MonthsWithData     int             `json:"months_with_data"`
}

type CurrentTotalMoney struct {
	TotalMoney decimal.Decimal `json:"total_money"`
}

type MonthOverMonthTrend struct {
	Month       string          `json:"month"`
	TotalAmount decimal.Decimal `json:"total_amount"`
}
