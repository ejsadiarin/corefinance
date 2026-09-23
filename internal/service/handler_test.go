package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type stubQuerier struct {
	db.Querier
	expenses []db.ListActiveExpenseRulesAllUsersRow
	incomes  []db.ListActiveIncomeRulesAllUsersRow
	err      error
}

func (s *stubQuerier) ListActiveExpenseRulesAllUsers(context.Context) ([]db.ListActiveExpenseRulesAllUsersRow, error) {
	return s.expenses, s.err
}

func (s *stubQuerier) ListActiveIncomeRulesAllUsers(context.Context) ([]db.ListActiveIncomeRulesAllUsersRow, error) {
	return s.incomes, s.err
}

func TestActiveRecurringRules(t *testing.T) {
	stub := &stubQuerier{
		expenses: []db.ListActiveExpenseRulesAllUsersRow{
			{ID: uuid.New(), UserID: uuid.New(), Description: "Netflix", Amount: decimal.NewFromFloat(12.99), Currency: "USD", RecurringType: "monthly"},
		},
		incomes: []db.ListActiveIncomeRulesAllUsersRow{
			{ID: uuid.New(), UserID: uuid.New(), Description: "Salary", Amount: decimal.NewFromFloat(3000), Currency: "USD", RecurringType: "monthly"},
		},
	}
	h := NewHandler(NewService(stub))

	rec := httptest.NewRecorder()
	h.ActiveRecurringRules(rec, httptest.NewRequest(http.MethodGet, "/internal/recurring/active", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp struct {
		ExpenseRules []map[string]any `json:"expense_rules"`
		IncomeRules  []map[string]any `json:"income_rules"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.ExpenseRules) != 1 || resp.ExpenseRules[0]["description"] != "Netflix" {
		t.Errorf("expense_rules = %+v", resp.ExpenseRules)
	}
	if len(resp.IncomeRules) != 1 || resp.IncomeRules[0]["description"] != "Salary" {
		t.Errorf("income_rules = %+v", resp.IncomeRules)
	}
}

func TestActiveRecurringRulesError(t *testing.T) {
	h := NewHandler(NewService(&stubQuerier{err: errors.New("db down")}))

	rec := httptest.NewRecorder()
	h.ActiveRecurringRules(rec, httptest.NewRequest(http.MethodGet, "/internal/recurring/active", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}
