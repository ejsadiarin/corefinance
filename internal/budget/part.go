package budget

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/ejsadiarin/corefinance/internal/category"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// The budget part of an export (coregateway ADR-0008). Its data is export
// types of their own, never database rows, so a migration can't silently
// change the file format. Bump PartVersion only for a breaking change, and
// teach ReadPart to upgrade the old version.
const (
	PartName    = "budget"
	PartFormat  = "coregateway.budget"
	PartVersion = 1
	PartSchema  = "https://raw.githubusercontent.com/ejsadiarin/corefinance/main/schemas/budget.v1.json"
)

// ErrUnreadable is a file that is not a budget part this service can read:
// not JSON, another format, or a version newer than PartVersion.
var ErrUnreadable = errors.New("not a readable budget part")

const dateLayout = "2006-01-02"

type Part struct {
	Schema     string     `json:"$schema"`
	Format     string     `json:"format"`
	Version    int        `json:"version"`
	ExportedAt time.Time  `json:"exported_at"`
	Source     PartSource `json:"source"`
	Data       BudgetData `json:"data"`
}

type PartSource struct {
	Service string `json:"service"`
}

type BudgetData struct {
	Tags     []PartTag     `json:"tags"`
	Rules    []PartRule    `json:"rules"`
	Expenses []PartExpense `json:"expenses"`
	Incomes  []PartIncome  `json:"incomes"`
}

type PartTag struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Color     *string    `json:"color"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

// PartRule is a recurring rule. MaterializedThrough is the last date it has
// filled (corefinance ADR-0001): an imported rule carries on after it and
// never fills a date the file already holds.
type PartRule struct {
	ID                  uuid.UUID       `json:"id"`
	Kind                string          `json:"kind"`
	Description         string          `json:"description"`
	Notes               *string         `json:"notes"`
	Amount              decimal.Decimal `json:"amount"`
	Currency            string          `json:"currency"`
	Category            *string         `json:"category"`
	Recurrence          string          `json:"recurrence"`
	StartDate           string          `json:"start_date"`
	EndDate             *string         `json:"end_date"`
	Priority            string          `json:"priority"`
	Paused              bool            `json:"paused"`
	MaterializedThrough *string         `json:"materialized_through"`
	CreatedAt           *time.Time      `json:"created_at,omitempty"`
}

// PartRecord is what an expense and an income share. Category is a system
// category's name (ADR-0002); conversion fields are kept as recorded
// (ADR-0003). RuleID marks a rule's entry, and EditedAt whether the owner
// edited it, which a later schedule change reads.
type PartRecord struct {
	ID           uuid.UUID       `json:"id"`
	Description  string          `json:"description"`
	Notes        *string         `json:"notes"`
	Amount       decimal.Decimal `json:"amount"`
	Currency     string          `json:"currency"`
	ExchangeRate decimal.Decimal `json:"exchange_rate"`
	AmountPHP    decimal.Decimal `json:"amount_php"`
	RateDate     *string         `json:"rate_date"`
	Date         string          `json:"date"`
	OccurredAt   *time.Time      `json:"occurred_at"`
	Category     *string         `json:"category"`
	Priority     string          `json:"priority"`
	Status       string          `json:"status"`
	RuleID       *uuid.UUID      `json:"rule_id"`
	EditedAt     *time.Time      `json:"edited_at"`
	Tags         []uuid.UUID     `json:"tags"`
	CreatedAt    *time.Time      `json:"created_at,omitempty"`
}

type PartExpense struct {
	PartRecord
	IsDebt bool `json:"is_debt"`
}

type PartIncome struct {
	PartRecord
}

// ReadPart reads a budget part, or the unversioned export that came before
// it (version 0), and returns its data upgraded to PartVersion with the
// version the file was written in.
func ReadPart(raw []byte) (*BudgetData, int, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrUnreadable, err)
	}
	if f, ok := top["format"]; ok {
		var format string
		var version int
		if err := json.Unmarshal(f, &format); err != nil || format != PartFormat {
			return nil, 0, fmt.Errorf("%w: format %s is not %s", ErrUnreadable, f, PartFormat)
		}
		if err := json.Unmarshal(top["version"], &version); err != nil || version < 1 {
			return nil, 0, fmt.Errorf("%w: version %s", ErrUnreadable, top["version"])
		}
		if version > PartVersion {
			return nil, 0, fmt.Errorf("%w: version %d is newer than %d", ErrUnreadable, version, PartVersion)
		}
		var data BudgetData
		if err := json.Unmarshal(top["data"], &data); err != nil {
			return nil, 0, fmt.Errorf("%w: data: %v", ErrUnreadable, err)
		}
		return &data, version, nil
	}
	for _, k := range []string{"expenses", "incomes", "tags", "recurring_expenses", "recurring_incomes"} {
		if _, ok := top[k]; ok {
			var v0 BudgetExport
			if err := json.Unmarshal(raw, &v0); err != nil {
				return nil, 0, fmt.Errorf("%w: %v", ErrUnreadable, err)
			}
			return upgradeV0(&v0), 0, nil
		}
	}
	return nil, 0, fmt.Errorf("%w: no format and no budget data", ErrUnreadable)
}

// upgradeV0 turns the unversioned export, which was database rows, into
// version 1 data. Its categories are named by the file's own category list.
func upgradeV0(v0 *BudgetExport) *BudgetData {
	names := map[uuid.UUID]string{}
	for _, c := range v0.ExpenseCategories {
		names[c.ID] = c.Name
	}
	for _, c := range v0.IncomeCategories {
		names[c.ID] = c.Name
	}
	categoryName := func(id pgtype.UUID, list []category.System) *string {
		if !id.Valid {
			return nil
		}
		if n, ok := names[id.Bytes]; ok {
			return &n
		}
		return systemName(id, list)
	}
	expenseTags := map[uuid.UUID][]uuid.UUID{}
	for _, et := range v0.ExpenseTags {
		expenseTags[et.ExpenseID] = append(expenseTags[et.ExpenseID], et.TagID)
	}
	incomeTags := map[uuid.UUID][]uuid.UUID{}
	for _, it := range v0.IncomeTags {
		incomeTags[it.IncomeID] = append(incomeTags[it.IncomeID], it.TagID)
	}

	data := &BudgetData{}
	for _, t := range v0.Tags {
		data.Tags = append(data.Tags, PartTag{ID: t.ID, Name: t.Name, Color: text(t.Color), CreatedAt: stamp(t.CreatedAt)})
	}
	for _, r := range v0.RecurringExpenses {
		data.Rules = append(data.Rules, partRule(db.Rule{
			ID: r.ID, Kind: "expense", Description: r.Description, Amount: r.Amount, Currency: r.Currency,
			CategoryID: r.CategoryID, Notes: r.Notes, RecurringType: r.RecurringType, StartDate: r.StartDate,
			EndDate: r.EndDate, Priority: r.Priority, IsActive: r.IsActive, MaterializedThrough: r.MaterializedThrough,
			CreatedAt: r.CreatedAt,
		}, categoryName(r.CategoryID, category.Expenses)))
	}
	for _, r := range v0.RecurringIncomes {
		r.Kind = "income"
		data.Rules = append(data.Rules, partRule(r, categoryName(r.CategoryID, category.Incomes)))
	}
	for _, e := range v0.Expenses {
		data.Expenses = append(data.Expenses, partExpense(e, categoryName(e.CategoryID, category.Expenses), expenseTags[e.ID]))
	}
	for _, i := range v0.Incomes {
		data.Incomes = append(data.Incomes, partIncome(i, categoryName(i.CategoryID, category.Incomes), incomeTags[i.ID]))
	}
	return data
}

func partRule(r db.Rule, cat *string) PartRule {
	return PartRule{
		ID: r.ID, Kind: r.Kind, Description: r.Description, Notes: text(r.Notes), Amount: r.Amount,
		Currency: r.Currency, Category: cat, Recurrence: r.RecurringType, StartDate: date(r.StartDate),
		EndDate: optDate(r.EndDate), Priority: r.Priority, Paused: !r.IsActive,
		MaterializedThrough: optDate(r.MaterializedThrough), CreatedAt: stamp(r.CreatedAt),
	}
}

func partExpense(e db.Expense, cat *string, tags []uuid.UUID) PartExpense {
	return PartExpense{
		PartRecord: PartRecord{
			ID: e.ID, Description: e.Description, Notes: text(e.Notes), Amount: e.Amount, Currency: e.Currency,
			ExchangeRate: e.ExchangeRate, AmountPHP: e.AmountPhp, RateDate: optDate(e.RateDate),
			Date: date(e.ExpenseDate), OccurredAt: stamp(e.OccurredAt), Category: cat, Priority: e.Priority,
			Status: e.Status, RuleID: optUUID(e.SourceRuleID), EditedAt: stamp(e.UserEditedAt),
			Tags: nonNil(tags), CreatedAt: stamp(e.CreatedAt),
		},
		IsDebt: e.IsDebt,
	}
}

func partIncome(i db.Income, cat *string, tags []uuid.UUID) PartIncome {
	return PartIncome{PartRecord: PartRecord{
		ID: i.ID, Description: i.Description, Notes: text(i.Notes), Amount: i.Amount, Currency: i.Currency,
		ExchangeRate: i.ExchangeRate, AmountPHP: i.AmountPhp, RateDate: optDate(i.RateDate),
		Date: date(i.Date), OccurredAt: stamp(i.OccurredAt), Category: cat, Priority: i.Priority,
		Status: i.Status, RuleID: optUUID(i.SourceRuleID), EditedAt: stamp(i.UserEditedAt),
		Tags: nonNil(tags), CreatedAt: stamp(i.CreatedAt),
	}}
}

// systemName is the name of the system category with this id, if any.
func systemName(id pgtype.UUID, list []category.System) *string {
	if !id.Valid {
		return nil
	}
	for _, c := range list {
		if c.ID == uuid.UUID(id.Bytes) {
			n := c.Name
			return &n
		}
	}
	return nil
}

// categoryFor is where an imported record's category lands (ADR-0002): the
// system category of that name, or `other` plus a tag of the name.
func categoryFor(list []category.System, other uuid.UUID, name *string) categoryTarget {
	if name == nil || strings.TrimSpace(*name) == "" {
		return categoryTarget{}
	}
	if sys, ok := category.ByName(list, *name); ok {
		return categoryTarget{ID: sys.ID}
	}
	return categoryTarget{ID: other, Tag: strings.TrimSpace(*name)}
}

func text(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func stamp(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time.UTC()
	return &v
}

func date(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format(dateLayout)
}

func optDate(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := date(d)
	return &s
}

func optUUID(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	v := uuid.UUID(id.Bytes)
	return &v
}

func nonNil(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}
