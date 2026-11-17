package stats

import (
	"github.com/shopspring/decimal"
)

type Summary struct {
	TotalExpenses       decimal.Decimal `json:"total_expenses"`
	TotalIncomes        decimal.Decimal `json:"total_incomes"`
	TotalReimbursements decimal.Decimal `json:"total_reimbursements"`
	ExpenseCount        int64           `json:"expense_count"`
	IncomeCount         int64           `json:"income_count"`
}

type Trend struct {
	Month               string          `json:"month"`
	TotalExpenses       decimal.Decimal `json:"total_expenses"`
	TotalIncomes        decimal.Decimal `json:"total_incomes"`
	TotalReimbursements decimal.Decimal `json:"total_reimbursements"`
}

type CategoryBreakdownItem struct {
	CategoryID    string          `json:"category_id"`
	CategoryName  string          `json:"category_name"`
	CategoryColor *string         `json:"category_color,omitempty"`
	ExpenseCount  int64           `json:"expense_count"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
}

type SavingsRate struct {
	TotalIncomes        decimal.Decimal `json:"total_incomes"`
	TotalReimbursements decimal.Decimal `json:"total_reimbursements"`
	TotalExpenses       decimal.Decimal `json:"total_expenses"`
	SavingsRate         decimal.Decimal `json:"savings_rate"`
}

type FiftyThirtyTwenty struct {
	TotalIncomes        decimal.Decimal `json:"total_incomes"`
	TotalReimbursements decimal.Decimal `json:"total_reimbursements"`
	TotalExpenses       decimal.Decimal `json:"total_expenses"`
	Needs               decimal.Decimal `json:"needs"`
	Wants               decimal.Decimal `json:"wants"`
	Savings             decimal.Decimal `json:"savings"`
	NeedsPct            decimal.Decimal `json:"needs_pct"`
	WantsPct            decimal.Decimal `json:"wants_pct"`
	SavingsPct          decimal.Decimal `json:"savings_pct"`
}

type SpendingVelocity struct {
	AvgMonthlySpending decimal.Decimal `json:"avg_monthly_spending"`
	MonthsWithData     int             `json:"months_with_data"`
}

type CurrentTotalMoney struct {
	TotalMoney        decimal.Decimal `json:"total_money"`
	Committed         decimal.Decimal `json:"committed"`
	AverageDailySpend decimal.Decimal `json:"average_daily_spend"`
	RunwayDays        *int64          `json:"runway_days"`
}

type CashFlowBucket struct {
	StartDate string          `json:"start_date"`
	EndDate   string          `json:"end_date"`
	Income    decimal.Decimal `json:"income"`
	Spent     decimal.Decimal `json:"spent"`
	Net       decimal.Decimal `json:"net"`
}

type CashFlowSeries struct {
	Bucket  string           `json:"bucket"`
	Buckets []CashFlowBucket `json:"buckets"`
}

type PeriodBudget struct {
	Period               string           `json:"period"`
	StartDate            string           `json:"start_date"`
	EndDate              string           `json:"end_date"`
	Funded               decimal.Decimal  `json:"funded"`
	ProjectedIncome      decimal.Decimal  `json:"projected_income"`
	Spent                decimal.Decimal  `json:"spent"`
	BillsStillDue        decimal.Decimal  `json:"bills_still_due"`
	Left                 decimal.Decimal  `json:"left"`
	SafeToSpendPerDay    *decimal.Decimal `json:"safe_to_spend_per_day"`
	PreviousPeriodStart  string           `json:"previous_period_start"`
	PreviousPeriodEnd    string           `json:"previous_period_end"`
	PreviousSpentToPoint decimal.Decimal  `json:"previous_spent_to_point"`
	PreviousSpentTotal   decimal.Decimal  `json:"previous_spent_total"`
}

type MonthOverMonthTrend struct {
	Month       string          `json:"month"`
	TotalAmount decimal.Decimal `json:"total_amount"`
}
