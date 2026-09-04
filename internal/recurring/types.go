package recurring

import (
	"github.com/shopspring/decimal"
)

type CreateIncomeRuleRequest struct {
	Amount        float64  `json:"amount"`
	Currency      string   `json:"currency"`
	Description   string   `json:"description"`
	RecurringType string   `json:"recurring_type"`
	StartDate     string   `json:"start_date"`
	EndDate       *string  `json:"end_date,omitempty"`
}

type UpdateIncomeRuleRequest struct {
	Amount        float64  `json:"amount"`
	Currency      string   `json:"currency"`
	Description   string   `json:"description"`
	RecurringType string   `json:"recurring_type"`
	StartDate     string   `json:"start_date"`
	EndDate       *string  `json:"end_date,omitempty"`
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
