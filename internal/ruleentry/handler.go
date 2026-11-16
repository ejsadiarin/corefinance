// Package ruleentry lists the expenses and incomes recurring rules created,
// for the review screen (#8): what the worker created, edited or skipped in
// one place.
package ruleentry

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
)

// Page sizes for GET /rule-entries.
const (
	defaultPageSize = 50
	maxPageSize     = 500
)

// Kind says which table a rule entry lives in.
type Kind string

const (
	KindExpense Kind = "expense"
	KindIncome  Kind = "income"
)

// Entry is one rule-created expense or income.
type Entry struct {
	Kind         Kind            `json:"kind"`
	ID           string          `json:"id"`
	RuleID       string          `json:"rule_id"`
	Date         string          `json:"date"`
	Amount       decimal.Decimal `json:"amount"`
	Currency     string          `json:"currency"`
	ExchangeRate decimal.Decimal `json:"exchange_rate"`
	AmountPHP    decimal.Decimal `json:"amount_php"`
	RateDate     string          `json:"rate_date"`
	Description  string          `json:"description"`
	Notes        *string         `json:"notes"`
	CategoryID   *string         `json:"category_id"`
	CategoryName *string         `json:"category_name"`
	Priority     string          `json:"priority"`
	Status       string          `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	// UserEditedAt is when the user last edited or skipped the entry; null
	// while it is as the rule created it (#12).
	UserEditedAt *time.Time `json:"user_edited_at"`
}

type Handler struct {
	queries db.Querier
}

func NewHandler(queries db.Querier) *Handler {
	return &Handler{queries: queries}
}

func nullableText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func nullableUUID(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes).String()
	return &id
}

func toEntry(r db.ListRuleEntriesRow) Entry {
	return Entry{
		Kind:         Kind(r.Kind),
		ID:           r.ID.String(),
		RuleID:       r.RuleID.String(),
		Date:         r.Date.Time.Format(time.DateOnly),
		Amount:       r.Amount,
		Currency:     r.Currency,
		ExchangeRate: r.ExchangeRate,
		AmountPHP:    r.AmountPhp,
		RateDate:     r.RateDate.Time.Format(time.DateOnly),
		Description:  r.Description,
		Notes:        nullableText(r.Notes),
		CategoryID:   nullableUUID(r.CategoryID),
		CategoryName: nullableText(r.CategoryName),
		Priority:     r.Priority,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
		UserEditedAt: nullableTime(r.UserEditedAt),
	}
}

func nullableTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// parseDate reads an optional YYYY-MM-DD query parameter.
func parseDate(r *http.Request, key string) (pgtype.Date, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return pgtype.Date{}, nil
	}
	d, err := time.Parse(time.DateOnly, v)
	if err != nil {
		return pgtype.Date{}, errors.New(key + " must be a date like 2026-09-01")
	}
	return pgtype.Date{Time: d, Valid: true}, nil
}

// parseFilters reads ?rule_id, ?start_date, ?end_date (YYYY-MM-DD, the
// entry's date) and ?created_since (RFC 3339, when the entry was inserted).
// An empty value means no filter; a malformed one is an error naming it.
func parseFilters(r *http.Request) (db.ListRuleEntriesParams, error) {
	var p db.ListRuleEntriesParams
	var err error
	if v := r.URL.Query().Get("rule_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return p, errors.New("rule_id must be a UUID")
		}
		p.RuleID = pgtype.UUID{Bytes: id, Valid: true}
	}
	if p.StartDate, err = parseDate(r, "start_date"); err != nil {
		return p, err
	}
	if p.EndDate, err = parseDate(r, "end_date"); err != nil {
		return p, err
	}
	if p.StartDate.Valid && p.EndDate.Valid && p.StartDate.Time.After(p.EndDate.Time) {
		return p, errors.New("start_date must not be after end_date")
	}
	if v := r.URL.Query().Get("created_since"); v != "" {
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return p, errors.New("created_since must be an RFC 3339 timestamp like 2026-09-01T08:00:00Z")
		}
		p.CreatedSince = pgtype.Timestamptz{Time: t, Valid: true}
	}
	return p, nil
}

// List returns the caller's rule-created entries, newest-inserted first,
// paged with ?page and ?page_size.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	p, err := parseFilters(r)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	page := max(helper.ParseQueryInt(r, "page", 1), 1)
	size := min(max(helper.ParseQueryInt(r, "page_size", defaultPageSize), 1), maxPageSize)
	p.UserID = userID
	p.PageLimit = int32(size)
	p.PageOffset = int32((page - 1) * size)

	slog.Debug("ruleentry.List: request received", "user_id", userID)
	rows, err := h.queries.ListRuleEntries(r.Context(), p)
	if err != nil {
		slog.Error("ruleentry.List: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, "couldn't list rule entries")
		return
	}
	out := make([]Entry, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntry(row))
	}
	slog.Debug("ruleentry.List: success", "user_id", userID, "count", len(out))
	helper.RespondJSON(w, http.StatusOK, out)
}
