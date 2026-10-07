package budget

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/auth"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

var backfillAnchor = time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)

func adminCtx(userID uuid.UUID, scopes ...string) context.Context {
	ctx := auth.WithUserID(context.Background(), userID)
	return auth.WithScopes(ctx, scopes)
}

func TestAdminBackfill_RejectsWithoutAdminScope(t *testing.T) {
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/budget/admin/backfill", nil)
	req = req.WithContext(adminCtx(uuid.New(), "corefinance:read"))
	rec := httptest.NewRecorder()

	h.Backfill(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAdminBackfill_RejectsWithoutUser(t *testing.T) {
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/budget/admin/backfill", nil)
	req = req.WithContext(auth.WithScopes(context.Background(), []string{"admin"}))
	rec := httptest.NewRecorder()

	h.Backfill(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAdminBackfill_RejectsBadUserID(t *testing.T) {
	h := NewAdminHandler(nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/budget/admin/backfill?user_id=zzz", nil)
	req = req.WithContext(adminCtx(uuid.New(), "admin"))
	rec := httptest.NewRecorder()

	h.Backfill(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func seedLegacyIncome(t *testing.T, q *db.Queries, userID uuid.UUID) {
	t.Helper()
	_, err := q.CreateIncome(context.Background(), db.CreateIncomeParams{
		UserID:        userID,
		Amount:        decimal.NewFromFloat(500),
		Currency:      "PHP",
		Description:   "allowance",
		Date:          pgtype.Date{Time: backfillAnchor, Valid: true},
		RecurringType: "weekly",
		Priority:      "want",
		Status:        "posted",
		StartDate:     pgtype.Date{Time: backfillAnchor, Valid: true},
	})
	require.NoError(t, err)
}

func TestAdminBackfill_DryRunCountsWithoutMutating_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	h := NewAdminHandler(pool, q)
	admin := uuid.New()
	seedLegacyIncome(t, q, admin)

	req := httptest.NewRequest(http.MethodPost, "/api/budget/admin/backfill?dry_run=true", nil)
	req = req.WithContext(adminCtx(admin, "admin"))
	rec := httptest.NewRecorder()

	h.Backfill(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp BackfillResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.True(t, resp.DryRun)
	assert.Equal(t, admin, resp.TargetUserID)
	assert.Equal(t, int64(1), resp.LegacyTotal)

	var remaining int64
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM incomes WHERE recurring_type != 'one-time' AND source_rule_id IS NULL`).Scan(&remaining))
	assert.Equal(t, int64(1), remaining, "dry-run must not mutate")
}

func TestAdminBackfill_ExecuteDefaultsToSelf_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	h := NewAdminHandler(pool, q)
	admin := uuid.New()
	other := uuid.New()
	seedLegacyIncome(t, q, admin)
	seedLegacyIncome(t, q, other)

	req := httptest.NewRequest(http.MethodPost, "/api/budget/admin/backfill", nil)
	req = req.WithContext(adminCtx(admin, "admin"))
	rec := httptest.NewRecorder()

	h.Backfill(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp BackfillResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.False(t, resp.DryRun)
	assert.Equal(t, admin, resp.TargetUserID)
	assert.Equal(t, 1, resp.IncomeRules)
	assert.Equal(t, int64(0), resp.LegacyRemaining)

	rules, err := q.ListRecurringIncomeRules(context.Background(), other)
	require.NoError(t, err)
	assert.Empty(t, rules, "other user's rows must be untouched")
}

func TestAdminBackfill_ExecuteTargetsOtherUser_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	q := db.New(pool)
	h := NewAdminHandler(pool, q)
	admin := uuid.New()
	target := uuid.New()
	seedLegacyIncome(t, q, target)

	req := httptest.NewRequest(http.MethodPost, "/api/budget/admin/backfill?user_id="+target.String(), nil)
	req = req.WithContext(adminCtx(admin, "admin"))
	rec := httptest.NewRecorder()

	h.Backfill(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp BackfillResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, target, resp.TargetUserID)
	assert.Equal(t, 1, resp.IncomeRules)
	assert.Equal(t, int64(0), resp.LegacyRemaining)
}
