package recurring

import (
	"github.com/shopspring/decimal"
)

type CreateIncomeRuleRequest struct {
	Amount        float64  `json:"amount"`
	Currency      string   `json:"currency"`
	Description   string   `json:"description"`
	CategoryID    *string  `json:"category_id,omitempty"`
	Notes         *string  `json:"notes,omitempty"`
	// Priority is need, want or savings; omitted keeps the current value (want on create).
	Priority      *string  `json:"priority,omitempty"`
	RecurringType string   `json:"recurring_type"`
	StartDate     string   `json:"start_date"`
	EndDate       *string  `json:"end_date,omitempty"`
}

type UpdateIncomeRuleRequest struct {
	Amount        float64  `json:"amount"`
	Currency      string   `json:"currency"`
	Description   string   `json:"description"`
	CategoryID    *string  `json:"category_id,omitempty"`
	Notes         *string  `json:"notes,omitempty"`
	// Priority is need, want or savings; omitted keeps the current value (want on create).
	Priority      *string  `json:"priority,omitempty"`
	RecurringType string   `json:"recurring_type"`
	StartDate     string   `json:"start_date"`
	EndDate       *string  `json:"end_date,omitempty"`
	IsActive      *bool    `json:"is_active,omitempty"`
	// EntriesAfterEnd is "skip" (default) or "keep": what happens to entries
	// the rule created after an end date moved into the past (#5).
	EntriesAfterEnd string `json:"entries_after_end,omitempty"`
}

type CreateExpenseRuleRequest struct {
	Amount        float64  `json:"amount"`
	Currency      string   `json:"currency"`
	Description   string   `json:"description"`
	CategoryID    *string  `json:"category_id,omitempty"`
	Notes         *string  `json:"notes,omitempty"`
	RecurringType string   `json:"recurring_type"`
	StartDate     string   `json:"start_date"`
	EndDate       *string  `json:"end_date,omitempty"`
	Priority      string   `json:"priority"`
}

type UpdateExpenseRuleRequest struct {
	Amount        float64  `json:"amount"`
	Currency      string   `json:"currency"`
	Description   string   `json:"description"`
	CategoryID    *string  `json:"category_id,omitempty"`
	Notes         *string  `json:"notes,omitempty"`
	RecurringType string   `json:"recurring_type"`
	StartDate     string   `json:"start_date"`
	EndDate       *string  `json:"end_date,omitempty"`
	Priority      string   `json:"priority"`
	IsActive      *bool    `json:"is_active,omitempty"`
	// EntriesAfterEnd is "skip" (default) or "keep": what happens to entries
	// the rule created after an end date moved into the past (#5).
	EntriesAfterEnd string `json:"entries_after_end,omitempty"`
}

type IncomeRuleResponse struct {
	ID            string          `json:"id"`
	UserID        string          `json:"user_id"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      string          `json:"currency"`
	Description   string          `json:"description"`
	RecurringType string          `json:"recurring_type"`
	StartDate     string          `json:"start_date"`
	EndDate       *string         `json:"end_date,omitempty"`
	CreatedAt     string          `json:"created_at"`
	UpdatedAt     string          `json:"updated_at"`
}
