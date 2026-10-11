package budget

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/ejsadiarin/corefinance/internal/auth"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
)

// targetingStubQuerier returns per-user totals so tests can tell whose data
// the handler resolved.
type targetingStubQuerier struct {
	db.Querier
	income  map[uuid.UUID]decimal.Decimal
	expense map[uuid.UUID]decimal.Decimal
}

func (q *targetingStubQuerier) GetTotalIncomesByDateRange(_ context.Context, arg db.GetTotalIncomesByDateRangeParams) (decimal.Decimal, error) {
	return q.income[arg.UserID], nil
}

func (q *targetingStubQuerier) GetTotalExpensesByDateRange(_ context.Context, arg db.GetTotalExpensesByDateRangeParams) (decimal.Decimal, error) {
	return q.expense[arg.UserID], nil
}

type stubTransactioner struct{}

func (stubTransactioner) Begin(context.Context) (pgx.Tx, error) { return nil, nil }

func targetedRemaining(t *testing.T, scopes []string, self, other, actAs uuid.UUID, rawParam string) BudgetRemainingResponse {
	t.Helper()
	q := &targetingStubQuerier{
		income:  map[uuid.UUID]decimal.Decimal{self: decimal.NewFromInt(100), other: decimal.NewFromInt(1000)},
		expense: map[uuid.UUID]decimal.Decimal{self: decimal.NewFromInt(10), other: decimal.NewFromInt(100)},
	}
	h := NewHandler(NewService(q, stubTransactioner{}))
	url := "/api/budget/remaining"
	if rawParam != "" {
		url += "?user_id=" + rawParam
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	ctx := auth.WithUserID(req.Context(), self)
	ctx = auth.WithScopes(ctx, scopes)
	if actAs != uuid.Nil {
		ctx = auth.WithActAs(ctx, actAs)
	}
	rec := httptest.NewRecorder()
	h.Remaining(rec, req.WithContext(ctx))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp BudgetRemainingResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp
}

func remainingValue(t *testing.T, v interface{}) string {
	t.Helper()
	s, ok := v.(string)
	if !ok {
		t.Fatalf("remaining = %T, want string", v)
	}
	return s
}

func TestRemainingTargetedReturnsOthersData(t *testing.T) {
	self := uuid.New()
	other := uuid.New()
	resp := targetedRemaining(t, []string{"admin"}, self, other, other, "")
	// other: 1000 income - 100 expense = 900; self would be 90.
	if got := remainingValue(t, resp.Remaining); got != "900" {
		t.Errorf("remaining = %s, want 900 (other's data)", got)
	}
}

func TestRemainingForgedParamReturnsSubData(t *testing.T) {
	self := uuid.New()
	other := uuid.New()
	// No act_as in context (direct caller forging only the raw param):
	// downstream must ignore it and serve sub's data (100-10=90).
	resp := targetedRemaining(t, []string{"admin"}, self, other, uuid.Nil, uuid.New().String())
	if got := remainingValue(t, resp.Remaining); got != "90" {
		t.Errorf("remaining = %s, want 90 (sub's data)", got)
	}
}

func TestRemainingSmuggledClaimFallsBackToSub(t *testing.T) {
	self := uuid.New()
	other := uuid.New()
	// act_as present but no admin scope: ignored, sub's data.
	resp := targetedRemaining(t, []string{"corefinance:read"}, self, other, other, "")
	if got := remainingValue(t, resp.Remaining); got != "90" {
		t.Errorf("remaining = %s, want 90 (sub's data)", got)
	}
}
