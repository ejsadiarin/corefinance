package expense

import (
	"context"
	"encoding/json"
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
	"github.com/ejsadiarin/corefinance/internal/occurred"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func manila() context.Context {
	return auth.WithTimezone(context.Background(), "Asia/Manila")
}

func clock(s string) *string { return &s }

// A time is the owner's wall clock on the record's day: 23:30 in Manila on
// Nov 24 is 15:30 UTC, and the day stays Nov 24.
func TestCreate_TimeIsStoredAsTheMomentInTheOwnersZone_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(db.New(pool))

	created, err := svc.Create(manila(), uuid.New(), CreateRequest{
		Amount: 250, Currency: "PHP", Description: "late dinner", ExpenseDate: "2026-11-24",
		Time: clock("23:30"), Priority: "want", Status: "posted",
	})
	require.NoError(t, err)

	require.True(t, created.OccurredAt.Valid)
	assert.Equal(t, time.Date(2026, 11, 24, 15, 30, 0, 0, time.UTC), created.OccurredAt.Time.UTC())
	assert.Equal(t, "2026-11-24", created.ExpenseDate.Time.Format("2006-01-02"))
}

// An edit sends the record's time every time; one without a time clears it.
func TestUpdate_SetsOrClearsTheTime_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(db.New(pool))
	owner := uuid.New()
	req := CreateRequest{
		Amount: 250, Currency: "PHP", Description: "lunch", ExpenseDate: "2026-11-24",
		Priority: "want", Status: "posted",
	}
	created, err := svc.Create(manila(), owner, req)
	require.NoError(t, err)
	assert.False(t, created.OccurredAt.Valid, "no time given")

	update := UpdateRequest{
		Amount: 250, Currency: "PHP", Description: "lunch", ExpenseDate: "2026-11-25",
		Time: clock("08:05"), Priority: "want", Status: "posted",
	}
	timed, err := svc.Update(manila(), created.ID, owner, update)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 11, 25, 0, 5, 0, 0, time.UTC), timed.OccurredAt.Time.UTC())

	update.Time = nil
	cleared, err := svc.Update(manila(), created.ID, owner, update)
	require.NoError(t, err)
	assert.False(t, cleared.OccurredAt.Valid)
}

func TestCreate_InvalidTimeWritesNothing_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	owner := uuid.New()
	for _, bad := range []string{"24:00", "7pm", "12:5"} {
		_, err := NewService(db.New(pool)).Create(manila(), owner, CreateRequest{
			Amount: 1, Currency: "PHP", Description: "x", ExpenseDate: "2026-11-24",
			Time: clock(bad), Priority: "want", Status: "posted",
		})
		require.ErrorIs(t, err, occurred.ErrInvalidTime, bad)
	}
	var count int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM expenses WHERE user_id = $1`, owner).Scan(&count))
	assert.Equal(t, 0, count)
}

// Over HTTP: a bad time is the client's mistake, and a saved record reports
// its moment as occurred_at.
func TestHandlerCreate_Time_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	h := NewHandler(NewService(db.New(pool)))
	post := func(body string) *httptest.ResponseRecorder {
		ctx := auth.WithTimezone(auth.WithUserID(context.Background(), uuid.New()), "Asia/Manila")
		req := httptest.NewRequest(http.MethodPost, "/api/budget/expenses", strings.NewReader(body)).WithContext(ctx)
		rec := httptest.NewRecorder()
		h.Create(rec, req)
		return rec
	}
	base := `"amount": 5, "currency": "PHP", "description": "x", "expense_date": "2026-11-24", "priority": "want", "status": "posted"`

	bad := post(`{` + base + `, "time": "25:00"}`)
	assert.Equal(t, http.StatusBadRequest, bad.Code, bad.Body.String())

	ok := post(`{` + base + `, "time": "09:15"}`)
	require.Equal(t, http.StatusCreated, ok.Code, ok.Body.String())
	var got struct {
		OccurredAt *time.Time `json:"occurred_at"`
	}
	require.NoError(t, json.Unmarshal(ok.Body.Bytes(), &got))
	require.NotNil(t, got.OccurredAt)
	assert.Equal(t, time.Date(2026, 11, 24, 1, 15, 0, 0, time.UTC), got.OccurredAt.UTC())

	none := post(`{` + base + `}`)
	require.Equal(t, http.StatusCreated, none.Code)
	assert.Contains(t, none.Body.String(), `"occurred_at":null`)
}

// Within a day, timed records come first, latest first; untimed ones follow
// in the order they were entered, latest first. Days stay newest first.
func TestList_OrdersByTimeWithinADay_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	svc := NewService(db.New(pool))
	owner := uuid.New()
	add := func(desc, date string, at *string) {
		_, err := svc.Create(manila(), owner, CreateRequest{
			Amount: 1, Currency: "PHP", Description: desc, ExpenseDate: date,
			Time: at, Priority: "want", Status: "posted",
		})
		require.NoError(t, err)
	}
	add("untimed first", "2026-11-24", nil)
	add("morning", "2026-11-24", clock("08:00"))
	add("evening", "2026-11-24", clock("20:00"))
	add("untimed later", "2026-11-24", nil)
	add("next day", "2026-11-25", nil)

	rows, err := svc.List(manila(), owner, ListParams{})
	require.NoError(t, err)
	var got []string
	for _, r := range rows {
		got = append(got, r.Description)
	}
	assert.Equal(t, []string{"next day", "evening", "morning", "untimed later", "untimed first"}, got)
}
