package budget

import (
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

type BudgetExport struct {
	ExpenseCategories []db.ExpenseCategory              `json:"expense_categories"`
	IncomeCategories  []db.IncomeCategory               `json:"income_categories"`
	Tags              []db.Tag                          `json:"tags"`
	Expenses          []db.Expense                      `json:"expenses"`
	Incomes           []db.Income                       `json:"incomes"`
	RecurringExpenses []db.ListRecurringExpenseRulesRow `json:"recurring_expenses"`
	RecurringIncomes  []db.RecurringIncomeRule          `json:"recurring_incomes"`
	ExpenseTags       []db.ExpenseTag                   `json:"expense_tags"`
	IncomeTags        []db.IncomeTag                    `json:"income_tags"`
}

type BudgetRemainingResponse struct {
	PeriodStart  string      `json:"period_start"`
	PeriodEnd    string      `json:"period_end"`
	TotalIncome  interface{} `json:"total_income"`
	TotalExpense interface{} `json:"total_expense"`
	Remaining    interface{} `json:"remaining"`
}

type PriorityGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
