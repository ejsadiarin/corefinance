package income

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
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func seedIncome(t *testing.T, q *db.Queries, userID uuid.UUID, description string) uuid.UUID {
	t.Helper()
	day := pgtype.Date{Valid: true}
	require.NoError(t, day.Scan("2026-11-01"))
	i, err := q.CreateIncome(context.Background(), db.CreateIncomeParams{
		UserID:       userID,
		Amount:       decimal.NewFromInt(5000),
		Currency:     "PHP",
		ExchangeRate: decimal.NewFromInt(1),
		AmountPhp:    decimal.NewFromInt(5000),
		RateDate:     day,
		Description:  description,
		Date:         day,
		Priority:     "want",
		Status:       "posted",
	})
	require.NoError(t, err)
	return i.ID
}

func TestListByIDs_ReturnsOnlyTheCallersRemainingIncomes_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	h := NewHandler(NewService(q))
	me, someoneElse := uuid.New(), uuid.New()

	kept := seedIncome(t, q, me, "salary")
	deleted := seedIncome(t, q, me, "refund")
	require.NoError(t, q.DeleteIncome(context.Background(), db.DeleteIncomeParams{ID: deleted, UserID: me}))
	theirs := seedIncome(t, q, someoneElse, "their salary")
	seedIncome(t, q, me, "not asked for")

	req := httptest.NewRequest(http.MethodGet,
		"/api/budget/incomes?ids="+strings.Join([]string{kept.String(), deleted.String(), theirs.String()}, ","), nil)
	req = req.WithContext(auth.WithUserID(context.Background(), me))
	rec := httptest.NewRecorder()
	h.List(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var got []struct {
		ID          uuid.UUID `json:"id"`
		Description string    `json:"description"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, kept, got[0].ID)
	assert.Equal(t, "salary", got[0].Description)

	bad := httptest.NewRequest(http.MethodGet, "/api/budget/incomes?ids=nope", nil)
	bad = bad.WithContext(auth.WithUserID(context.Background(), me))
	rec = httptest.NewRecorder()
	h.List(rec, bad)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
