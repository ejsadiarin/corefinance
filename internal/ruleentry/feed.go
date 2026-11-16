package ruleentry

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
)

// Feed is one page of the review feed (#13): entries in the order their
// transactions started, and the cursor to pass as ?after= next time.
type Feed struct {
	Entries []Entry `json:"entries"`
	Next    string  `json:"next"`
}

// cursor is a position in the feed: past every entry at or before (txid, id).
// The zero cursor is the start of the feed.
type cursor struct {
	txid uint64
	id   uuid.UUID
}

func (c cursor) String() string { return fmt.Sprintf("%d.%s", c.txid, c.id) }

func (c cursor) less(o cursor) bool {
	return c.txid < o.txid || (c.txid == o.txid && strings.Compare(c.id.String(), o.id.String()) < 0)
}

var errBadCursor = errors.New("after must be a cursor returned by an earlier feed page")

func parseCursor(s string) (cursor, error) {
	if s == "" {
		return cursor{}, nil
	}
	txid, id, ok := strings.Cut(s, ".")
	if !ok {
		return cursor{}, errBadCursor
	}
	n, err := strconv.ParseUint(txid, 10, 64)
	if err != nil {
		return cursor{}, errBadCursor
	}
	u, err := uuid.Parse(id)
	if err != nil {
		return cursor{}, errBadCursor
	}
	return cursor{txid: n, id: u}, nil
}

// Feed returns the caller's rule entries after ?after=, oldest transaction
// first, with the rule and date filters of List. It never returns entries
// from transactions newer than the oldest one still in flight, so a page's
// cursor can't move past an entry that commits later: that entry shows up
// on a later page instead. A long-running transaction anywhere in the
// database delays the feed; it never loses entries. A cursor is only valid
// with the filters that produced it: with other filters, its page can pass
// entries those filters would have returned before it.
func (h *Handler) Feed(w http.ResponseWriter, r *http.Request) {
	userID, ok := helper.ResolveTargetUser(w, r)
	if !ok {
		return
	}
	filters, err := parseFilters(r)
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	after, err := parseCursor(r.URL.Query().Get("after"))
	if err != nil {
		helper.RespondErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	size := min(max(helper.ParseQueryInt(r, "page_size", defaultPageSize), 1), maxPageSize)

	// One horizon per page: everything before it has finished, and the list
	// query's (later) snapshot sees all of it.
	horizon, err := h.queries.GetFeedHorizon(r.Context())
	if err != nil {
		slog.Error("ruleentry.Feed: horizon failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, "couldn't read the feed")
		return
	}
	rows, err := h.queries.ListRuleEntriesFeed(r.Context(), db.ListRuleEntriesFeedParams{
		UserID: userID, AfterTxid: after.txid, AfterID: after.id, Horizon: horizon,
		RuleID: filters.RuleID, StartDate: filters.StartDate, EndDate: filters.EndDate,
		PageLimit: int32(size),
	})
	if err != nil {
		slog.Error("ruleentry.Feed: failed", "error", err, "user_id", userID)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, "couldn't read the feed")
		return
	}

	out := Feed{Entries: make([]Entry, 0, len(rows))}
	next := after
	for _, row := range rows {
		out.Entries = append(out.Entries, toEntry(db.ListRuleEntriesRow{
			Kind: row.Kind, ID: row.ID, RuleID: row.RuleID, Date: row.Date, Amount: row.Amount,
			Currency: row.Currency, ExchangeRate: row.ExchangeRate, AmountPhp: row.AmountPhp,
			RateDate: row.RateDate, Description: row.Description, Notes: row.Notes,
			CategoryID: row.CategoryID, CategoryName: row.CategoryName, Priority: row.Priority,
			Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
			UserEditedAt: row.UserEditedAt,
		}))
		next = cursor{txid: row.Txid, id: row.ID}
	}
	// A short page read everything before the horizon: the next page starts
	// there. A full page continues from its last entry.
	if len(rows) < size {
		if h := (cursor{txid: horizon}); next.less(h) {
			next = h
		}
	}
	out.Next = next.String()
	helper.RespondJSON(w, http.StatusOK, out)
}
