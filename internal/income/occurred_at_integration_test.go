package income

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/auth"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func clock(s string) *string { return &s }

// An income's time is the owner's wall clock on its day, stored as UTC;
// an edit without a time clears it.
func TestIncomeTime_CreateAndUpdate_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(db.New(pool))
	ctx := auth.WithTimezone(context.Background(), "Asia/Manila")
	owner := uuid.New()

	created, err := svc.Create(ctx, owner, CreateRequest{
		Amount: 1000, Currency: "PHP", Description: "pay", Date: "2026-11-30",
		Time: clock("00:30"), Priority: "need", Status: "posted",
	})
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 11, 29, 16, 30, 0, 0, time.UTC), created.OccurredAt.Time.UTC())
	assert.Equal(t, "2026-11-30", created.Date.Time.Format("2006-01-02"))

	cleared, err := svc.Update(ctx, created.ID, owner, UpdateRequest{
		Amount: 1000, Currency: "PHP", Description: "pay", Date: "2026-11-30",
		Priority: "need", Status: "posted",
	})
	require.NoError(t, err)
	assert.False(t, cleared.OccurredAt.Valid)
}

func TestIncomeHandler_BadTimeIs400_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	h := NewHandler(NewService(db.New(pool)))
	ctx := auth.WithUserID(context.Background(), uuid.New())
	body := `{"amount": 5, "currency": "PHP", "description": "x", "date": "2026-11-24", "priority": "need", "status": "posted", "time": "9am"}`
	rec := httptest.NewRecorder()
	h.Create(rec, httptest.NewRequest(http.MethodPost, "/api/budget/incomes", strings.NewReader(body)).WithContext(ctx))
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}
