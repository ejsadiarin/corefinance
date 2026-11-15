package expense

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/auth"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/helper"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func seedExpense(t *testing.T, q *db.Queries, userID uuid.UUID, description string) uuid.UUID {
	t.Helper()
	day := pgtype.Date{Valid: true}
	require.NoError(t, day.Scan("2026-11-01"))
	e, err := q.CreateExpense(context.Background(), db.CreateExpenseParams{
		UserID:      userID,
		Amount:      decimal.NewFromInt(120),
		Currency:    "PHP",
		Description: description,
		ExpenseDate: day,
		Priority:    "want",
		Status:      "posted",
	})
	require.NoError(t, err)
	return e.ID
}

func listAs(h *Handler, userID uuid.UUID, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/budget/expenses?"+query, nil)
	req = req.WithContext(auth.WithUserID(context.Background(), userID))
	rec := httptest.NewRecorder()
	h.List(rec, req)
	return rec
}

func TestListByIDs_ReturnsOnlyTheCallersRemainingExpenses_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	h := NewHandler(NewService(q))
	me, someoneElse := uuid.New(), uuid.New()

	kept := seedExpense(t, q, me, "groceries")
	deleted := seedExpense(t, q, me, "taxi")
	require.NoError(t, q.DeleteExpense(context.Background(), db.DeleteExpenseParams{ID: deleted, UserID: me}))
	theirs := seedExpense(t, q, someoneElse, "their lunch")
	seedExpense(t, q, me, "not asked for")

	rec := listAs(h, me, "ids="+strings.Join([]string{kept.String(), deleted.String(), theirs.String()}, ","))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var got []struct {
		ID          uuid.UUID `json:"id"`
		Description string    `json:"description"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, kept, got[0].ID)
	assert.Equal(t, "groceries", got[0].Description)
}

func TestListByIDs_RejectsMalformedOrTooManyIDs_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	h := NewHandler(NewService(db.New(pool)))
	me := uuid.New()

	ids := make([]string, helper.MaxIDsPerRequest+1)
	for i := range ids {
		ids[i] = uuid.NewString()
	}
	for name, query := range map[string]string{
		"not a uuid": "ids=" + uuid.NewString() + ",nope",
		"too many":   "ids=" + strings.Join(ids, ","),
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, http.StatusBadRequest, listAs(h, me, query).Code)
		})
	}

	empty := listAs(h, me, "ids=")
	require.Equal(t, http.StatusOK, empty.Code, empty.Body.String())
	assert.JSONEq(t, "[]", empty.Body.String())

	rec := listAs(h, me, "ids="+strings.Join(ids[:helper.MaxIDsPerRequest], ","))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, "[]", rec.Body.String())
}
