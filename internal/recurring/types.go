package recurring

import (
	"github.com/shopspring/decimal"
)

type CreateIncomeRuleRequest struct {
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
	CategoryID  *string `json:"category_id,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	// Priority is need, want or savings; omitted keeps the current value (want on create).
	Priority      *string `json:"priority,omitempty"`
	RecurringType string  `json:"recurring_type"`
	StartDate     string  `json:"start_date"`
	EndDate       *string `json:"end_date,omitempty"`
}

type UpdateIncomeRuleRequest struct {
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
	CategoryID  *string `json:"category_id,omitempty"`
	Notes       *string `json:"notes,omitempty"`
	// Priority is need, want or savings; omitted keeps the current value (want on create).
	Priority      *string `json:"priority,omitempty"`
	RecurringType string  `json:"recurring_type"`
	StartDate     string  `json:"start_date"`
	EndDate       *string `json:"end_date,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
	// StaleEntries is "skip" (default) or "keep": what happens to untouched
	// entries a schedule change leaves behind (#15).
	StaleEntries string `json:"stale_entries,omitempty"`
	// EntriesAfterEnd is the #5 name for StaleEntries, still accepted.
	EntriesAfterEnd string `json:"entries_after_end,omitempty"`
}

type CreateExpenseRuleRequest struct {
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Description   string  `json:"description"`
	CategoryID    *string `json:"category_id,omitempty"`
	Notes         *string `json:"notes,omitempty"`
	RecurringType string  `json:"recurring_type"`
	StartDate     string  `json:"start_date"`
	EndDate       *string `json:"end_date,omitempty"`
	Priority      string  `json:"priority"`
}

type UpdateExpenseRuleRequest struct {
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Description   string  `json:"description"`
	CategoryID    *string `json:"category_id,omitempty"`
	Notes         *string `json:"notes,omitempty"`
	RecurringType string  `json:"recurring_type"`
	StartDate     string  `json:"start_date"`
	EndDate       *string `json:"end_date,omitempty"`
	Priority      string  `json:"priority"`
	IsActive      *bool   `json:"is_active,omitempty"`
	// StaleEntries is "skip" (default) or "keep": what happens to untouched
	// entries a schedule change leaves behind (#15).
	StaleEntries string `json:"stale_entries,omitempty"`
	// EntriesAfterEnd is the #5 name for StaleEntries, still accepted.
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

// staleChoice is the caller's stale_entries choice, or its #5 name.
func staleChoice(stale, afterEnd string) string {
	if stale != "" {
		return stale
	}
	return afterEnd
}
