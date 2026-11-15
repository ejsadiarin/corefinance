package income

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateRequest struct {
	CategoryID    *string  `json:"category_id,omitempty"`
	Amount        float64  `json:"amount"`
	Currency      string   `json:"currency"`
	Description   string   `json:"description"`
	Notes         *string  `json:"notes,omitempty"`
	Date          string   `json:"date"`
	RecurringType string   `json:"recurring_type"`
	Priority      string   `json:"priority"`
	Status        string   `json:"status"`
	StartDate     *string  `json:"start_date,omitempty"`
	EndDate       *string  `json:"end_date,omitempty"`
	SourceRuleID  *string  `json:"source_rule_id,omitempty"`
}

type UpdateRequest struct {
	CategoryID    *string  `json:"category_id,omitempty"`
	Amount        float64  `json:"amount"`
	Currency      string   `json:"currency"`
	Description   string   `json:"description"`
	Notes         *string  `json:"notes,omitempty"`
	Date          string   `json:"date"`
	RecurringType string   `json:"recurring_type"`
	Priority      string   `json:"priority"`
	Status        string   `json:"status"`
	StartDate     *string  `json:"start_date,omitempty"`
	EndDate       *string  `json:"end_date,omitempty"`
	SourceRuleID  *string  `json:"source_rule_id,omitempty"`
}

type ListParams struct {
	StartDate     *string `json:"start_date,omitempty"`
	EndDate       *string `json:"end_date,omitempty"`
	CategoryID    *string `json:"category_id,omitempty"`
	Status        *string `json:"status,omitempty"`
	RecurringType *string `json:"recurring_type,omitempty"`
	Page          int     `json:"page"`
	PageSize      int     `json:"page_size"`
}

type StatsByCategory struct {
	CategoryID    uuid.UUID       `json:"category_id"`
	CategoryName  string          `json:"category_name"`
	CategoryColor *string         `json:"category_color,omitempty"`
	IncomeCount   int64           `json:"income_count"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
}
