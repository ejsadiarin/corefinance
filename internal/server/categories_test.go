package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/auth"
	"github.com/ejsadiarin/corefinance/internal/budget"
	"github.com/ejsadiarin/corefinance/internal/category"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/expense"
	"github.com/ejsadiarin/corefinance/internal/income"
	"github.com/ejsadiarin/corefinance/internal/recurring"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func call(h http.HandlerFunc, user uuid.UUID, method, path string, body any, params map[string]string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf).WithContext(auth.WithUserID(context.Background(), user))
	if len(params) > 0 {
		req = withURLParams(req, params)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func withURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func str(s string) *string { return &s }

// Categories are a fixed list (#23, ADR-0002): every owner reads the same
// list, and no one can write it.
func TestCategoriesAreAFixedList_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	h := category.NewHandler(category.NewService(db.New(pool)))

	for _, tc := range []struct {
		list http.HandlerFunc
		want []category.System
	}{{h.ListExpense, category.Expenses}, {h.ListIncome, category.Incomes}} {
		for _, owner := range []uuid.UUID{uuid.New(), uuid.New()} {
			rec := call(tc.list, owner, http.MethodGet, "/", nil, nil)
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var got []struct {
				ID    uuid.UUID `json:"id"`
				Name  string    `json:"name"`
				Color string    `json:"color"`
				Icon  string    `json:"icon"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			require.Len(t, got, len(tc.want))
			for i, w := range tc.want {
				assert.Equal(t, w.ID, got[i].ID)
				assert.Equal(t, w.Name, got[i].Name)
				assert.Equal(t, w.Color, got[i].Color)
				assert.Equal(t, w.Icon, got[i].Icon)
			}
		}
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		rec := call(h.Fixed, uuid.New(), method, "/", map[string]string{"name": "coffee"}, nil)
		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code, method)
		assert.Contains(t, rec.Body.String(), "system-defined")
	}
}

// A record or rule names a system category of its own kind, or none (#23).
func TestRecordsTakeOnlyASystemCategoryOfTheirKind_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	expenses := expense.NewHandler(expense.NewService(q))
	incomes := income.NewHandler(income.NewService(q))
	rules := recurring.NewHandler(recurring.NewServiceWithPool(q, pool))
	me := uuid.New()

	food, _ := category.ByName(category.Expenses, "food")
	salary, _ := category.ByName(category.Incomes, "salary")
	oldOwnerCategory := uuid.NewString()

	newExpense := func(cat *string) expense.CreateRequest {
		return expense.CreateRequest{CategoryID: cat, Amount: 120, Currency: "PHP", Description: "lunch",
			ExpenseDate: "2026-11-15", Priority: "want", Status: "posted"}
	}
	for _, tc := range []struct {
		name string
		cat  *string
		want int
	}{
		{"a system expense category", str(food.ID.String()), http.StatusCreated},
		{"no category", nil, http.StatusCreated},
		{"an empty category", str(""), http.StatusCreated},
		{"an old per-owner id", &oldOwnerCategory, http.StatusBadRequest},
		{"an income category", str(salary.ID.String()), http.StatusBadRequest},
		{"not a uuid", str("food"), http.StatusBadRequest},
	} {
		rec := call(expenses.Create, me, http.MethodPost, "/", newExpense(tc.cat), nil)
		assert.Equal(t, tc.want, rec.Code, "expense with %s: %s", tc.name, rec.Body.String())
	}

	created := call(expenses.Create, me, http.MethodPost, "/", newExpense(str(food.ID.String())), nil)
	var e struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &e))
	update := expense.UpdateRequest{CategoryID: &oldOwnerCategory, Amount: 120, Currency: "PHP",
		Description: "lunch", ExpenseDate: "2026-11-15", Priority: "want", Status: "posted"}
	rec := call(expenses.Update, me, http.MethodPut, "/", update, map[string]string{"id": e.ID.String()})
	assert.Equal(t, http.StatusBadRequest, rec.Code, "updating an expense to an unknown category")

	pay := income.CreateRequest{Amount: 20000, Currency: "PHP", Description: "pay", Date: "2026-11-15",
		Priority: "need", Status: "posted"}
	pay.CategoryID = str(food.ID.String())
	assert.Equal(t, http.StatusBadRequest, call(incomes.Create, me, http.MethodPost, "/", pay, nil).Code,
		"an income can't take an expense category")
	pay.CategoryID = str(salary.ID.String())
	assert.Equal(t, http.StatusCreated, call(incomes.Create, me, http.MethodPost, "/", pay, nil).Code)

	expenseRule := recurring.CreateExpenseRuleRequest{Amount: 85, Currency: "PHP", Description: "spotify",
		RecurringType: "monthly", StartDate: "2026-11-15", Priority: "need", CategoryID: str(salary.ID.String())}
	assert.Equal(t, http.StatusBadRequest, call(rules.CreateExpenseRule, me, http.MethodPost, "/", expenseRule, nil).Code)
	incomeRule := recurring.CreateIncomeRuleRequest{Amount: 500, Currency: "PHP", Description: "allowance",
		RecurringType: "weekly", StartDate: "2026-11-15", CategoryID: &oldOwnerCategory}
	assert.Equal(t, http.StatusBadRequest, call(rules.CreateIncomeRule, me, http.MethodPost, "/", incomeRule, nil).Code)
	incomeRule.CategoryID = str(category.Reimbursement.String())
	assert.Equal(t, http.StatusCreated, call(rules.CreateIncomeRule, me, http.MethodPost, "/", incomeRule, nil).Code)
}

// Import maps a file's categories by name, like the migration: no match is
// `other` plus a tag of the old name, reusing the owner's tag (#23).
func TestImportMapsCategoriesByName_Integration(t *testing.T) {
	ctx := context.Background()
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	me := uuid.New()
	_, err := q.CreateTag(ctx, db.CreateTagParams{UserID: me, Name: "coffee"})
	require.NoError(t, err)

	fileFood, fileCoffee, fileGig := uuid.New(), uuid.New(), uuid.New()
	day := pgtype.Date{Valid: true}
	require.NoError(t, day.Scan("2026-07-12"))
	expenseIn := func(cat uuid.UUID, desc string) db.Expense {
		return db.Expense{ID: uuid.New(), CategoryID: pgtype.UUID{Bytes: cat, Valid: true}, Amount: decimal.NewFromInt(100),
			Currency: "PHP", Description: desc, ExpenseDate: day, Priority: "want", Status: "posted"}
	}
	file := &budget.BudgetExport{
		ExpenseCategories: []db.ExpenseCategory{{ID: fileFood, Name: "Food"}, {ID: fileCoffee, Name: " Coffee"}},
		IncomeCategories:  []db.IncomeCategory{{ID: fileGig, Name: "side gig"}},
		Expenses:          []db.Expense{expenseIn(fileFood, "lunch"), expenseIn(fileCoffee, "latte")},
		Incomes: []db.Income{{ID: uuid.New(), CategoryID: pgtype.UUID{Bytes: fileGig, Valid: true},
			Amount: decimal.NewFromInt(500), Currency: "PHP", Description: "gig", Date: day, Priority: "want", Status: "posted"}},
	}
	raw, err := json.Marshal(file)
	require.NoError(t, err)
	data, version, err := budget.ReadPart(raw)
	require.NoError(t, err)
	res, err := budget.NewService(q, pool).Import(ctx, me, data, version, false)
	require.NoError(t, err)
	require.Equal(t, budget.StatusImported, res.Status, "errors: %+v", res.Errors)

	type row struct {
		Category uuid.UUID
		Tags     []string
	}
	read := func(table, tags, col, desc string) row {
		t.Helper()
		var r row
		require.NoError(t, pool.QueryRow(ctx, `
			SELECT x.category_id,
			       COALESCE(array_agg(t.name ORDER BY t.name) FILTER (WHERE t.id IS NOT NULL), '{}')
			FROM `+table+` x
			LEFT JOIN `+tags+` j ON j.`+col+` = x.id
			LEFT JOIN tags t ON t.id = j.tag_id
			WHERE x.user_id = $1 AND x.description = $2
			GROUP BY x.category_id`, me, desc).Scan(&r.Category, &r.Tags))
		return r
	}
	food, _ := category.ByName(category.Expenses, "food")
	assert.Equal(t, row{food.ID, []string{}}, read("expenses", "expense_tags", "expense_id", "lunch"))
	assert.Equal(t, row{category.ExpenseOther, []string{"coffee"}}, read("expenses", "expense_tags", "expense_id", "latte"))
	assert.Equal(t, row{category.IncomeOther, []string{"side gig"}}, read("incomes", "income_tags", "income_id", "gig"))

	var coffeeTags int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM tags WHERE user_id = $1 AND lower(name) = 'coffee'`, me).Scan(&coffeeTags))
	assert.Equal(t, 1, coffeeTags, "the owner's coffee tag is reused")
}
