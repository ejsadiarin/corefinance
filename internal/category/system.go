package category

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

// System is one system-defined category (ADR-0002). The rows in
// expense_categories and income_categories carry these ids, names, colors and
// icons; migration 00015 seeds them, and owners can't add to them.
type System struct {
	ID    uuid.UUID
	Name  string
	Color string
	Icon  string
}

var (
	ExpenseOther = uuid.MustParse("00000000-0000-4000-8000-000000000112")
	IncomeOther  = uuid.MustParse("00000000-0000-4000-8000-000000000205")
	// Reimbursement income is left out of report income (ADR-0002, #24).
	Reimbursement = uuid.MustParse("00000000-0000-4000-8000-000000000204")
)

// Expenses and Incomes are the fixed lists in display order, `other` last.
var Expenses = []System{
	{uuid.MustParse("00000000-0000-4000-8000-000000000101"), "food", "#f97316", "🍜"},
	{uuid.MustParse("00000000-0000-4000-8000-000000000102"), "transportation", "#3b82f6", "🚌"},
	{uuid.MustParse("00000000-0000-4000-8000-000000000103"), "gifts", "#ec4899", "🎁"},
	{uuid.MustParse("00000000-0000-4000-8000-000000000104"), "subscriptions", "#8b5cf6", "🔁"},
	{uuid.MustParse("00000000-0000-4000-8000-000000000105"), "bills & installments", "#ef4444", "🧾"},
	{uuid.MustParse("00000000-0000-4000-8000-000000000106"), "personal care", "#14b8a6", "🧴"},
	{uuid.MustParse("00000000-0000-4000-8000-000000000107"), "entertainment", "#eab308", "🎬"},
	{uuid.MustParse("00000000-0000-4000-8000-000000000108"), "travel", "#0ea5e9", "✈️"},
	{uuid.MustParse("00000000-0000-4000-8000-000000000109"), "government & school", "#64748b", "🏛️"},
	{uuid.MustParse("00000000-0000-4000-8000-000000000110"), "shopping", "#d946ef", "🛍️"},
	{uuid.MustParse("00000000-0000-4000-8000-000000000111"), "communication", "#22c55e", "📱"},
	{ExpenseOther, "other", "#94a3b8", "📦"},
}

var Incomes = []System{
	{uuid.MustParse("00000000-0000-4000-8000-000000000201"), "allowance", "#22c55e", "💵"},
	{uuid.MustParse("00000000-0000-4000-8000-000000000202"), "gift", "#ec4899", "🎁"},
	{uuid.MustParse("00000000-0000-4000-8000-000000000203"), "salary", "#3b82f6", "💼"},
	{Reimbursement, "reimbursement", "#14b8a6", "↩️"},
	{IncomeOther, "other", "#94a3b8", "📦"},
}

// ErrUnknown is returned for a category id that isn't a system category of
// the record's kind.
var ErrUnknown = errors.New("unknown category: pick one from the category list")

// CheckExpense accepts no category (nil or empty) or a system expense category.
func CheckExpense(id *string) error { return check(Expenses, id) }

// CheckIncome accepts no category (nil or empty) or a system income category.
func CheckIncome(id *string) error { return check(Incomes, id) }

func check(list []System, id *string) error {
	if id == nil || *id == "" {
		return nil
	}
	parsed, err := uuid.Parse(*id)
	if err != nil {
		return ErrUnknown
	}
	for _, c := range list {
		if c.ID == parsed {
			return nil
		}
	}
	return ErrUnknown
}

// ByName finds the system category an owner's category name maps to: same
// name ignoring case and surrounding spaces. ok is false when none matches,
// and the caller files the record under `other` with a tag of that name.
func ByName(list []System, name string) (System, bool) {
	n := strings.ToLower(strings.TrimSpace(name))
	for _, c := range list {
		if c.Name == n {
			return c, true
		}
	}
	return System{}, false
}
