package budget

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ejsadiarin/corefinance/internal/auth"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
	"github.com/ejsadiarin/corefinance/internal/materialize"
)

// BackfillResponse is the per-user legacy backfill result. dry_run=true
// returns counts only; otherwise the mutation counts plus what is left.
type BackfillResponse struct {
	DryRun           bool      `json:"dry_run"`
	TargetUserID     uuid.UUID `json:"target_user_id"`
	LegacyExpenses   int64     `json:"legacy_expenses,omitempty"`
	LegacyIncomes    int64     `json:"legacy_incomes,omitempty"`
	LegacyTotal      int64     `json:"legacy_total,omitempty"`
	ExpenseRules     int       `json:"expense_rules,omitempty"`
	IncomeRules      int       `json:"income_rules,omitempty"`
	ExpenseInstances int       `json:"expense_instances,omitempty"`
	IncomeInstances  int       `json:"income_instances,omitempty"`
	LegacyRemaining  int64     `json:"legacy_remaining,omitempty"`
}

// AdminHandler serves admin-only maintenance endpoints. It holds the pool
// directly because backfill needs the advisory-lock transaction, which the
// query-scoped Service cannot provide.
type AdminHandler struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewAdminHandler(pool *pgxpool.Pool, queries *db.Queries) *AdminHandler {
	return &AdminHandler{pool: pool, queries: queries}
}

// Backfill runs the per-user legacy inline-rule conversion. Default target
// is the calling admin (sub); ?user_id=<uuid> targets anyone. ?dry_run=true
// counts without mutating. Gateway gates /admin/* by scope; the HasScope
// check here is defense-in-depth for direct :6969 callers.
func (h *AdminHandler) Backfill(w http.ResponseWriter, r *http.Request) {
	if !auth.HasScope(r.Context(), "admin") {
		helper.RespondErrorJSON(w, http.StatusForbidden, "admin scope required")
		return
	}
	caller, ok := helper.GetUserID(w, r)
	if !ok {
		return
	}

	target := caller
	if raw := r.URL.Query().Get("user_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			helper.RespondErrorJSON(w, http.StatusBadRequest, "invalid user_id")
			return
		}
		target = id
	}

	if r.URL.Query().Get("dry_run") == "true" {
		exp, inc, err := materialize.CountLegacyRowsForUser(r.Context(), h.queries, target)
		if err != nil {
			slog.Error("budget.Admin.Backfill: dry-run count failed", "error", err, "caller", caller, "target", target)
			helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
			return
		}
		helper.RespondJSON(w, http.StatusOK, BackfillResponse{
			DryRun:         true,
			TargetUserID:   target,
			LegacyExpenses: exp,
			LegacyIncomes:  inc,
			LegacyTotal:    exp + inc,
		})
		return
	}

	slog.Info("budget.Admin.Backfill: starting", "caller", caller, "target", target)
	res, err := materialize.BackfillLegacyForUser(r.Context(), h.pool, h.queries, target, time.Now())
	if err != nil {
		slog.Error("budget.Admin.Backfill: failed", "error", err, "caller", caller, "target", target)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	exp, inc, err := materialize.CountLegacyRowsForUser(r.Context(), h.queries, target)
	if err != nil {
		slog.Error("budget.Admin.Backfill: verification query failed", "error", err, "caller", caller, "target", target)
		helper.RespondErrorJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("budget.Admin.Backfill: completed", "caller", caller, "target", target,
		"expense_rules", res.ExpenseRules, "income_rules", res.IncomeRules, "legacy_remaining", exp+inc)
	helper.RespondJSON(w, http.StatusOK, BackfillResponse{
		DryRun:           false,
		TargetUserID:     target,
		ExpenseRules:     res.ExpenseRules,
		IncomeRules:      res.IncomeRules,
		ExpenseInstances: res.ExpenseInstances,
		IncomeInstances:  res.IncomeInstances,
		LegacyRemaining:  exp + inc,
	})
}
