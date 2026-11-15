package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/auth"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

// A request records the zone its token carries (#18), so background work can
// use it; a token without tz says nothing about the user's zone and records
// nothing.
func TestObserveTimezone_RecordsOnlyATokensZone_Integration(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	s := &Server{Pool: pool, Queries: db.New(pool)}
	seen := func(user uuid.UUID) string {
		tz, err := s.Queries.GetSeenTimezone(context.Background(), user)
		if errors.Is(err, pgx.ErrNoRows) {
			return "none"
		}
		require.NoError(t, err)
		return tz
	}
	request := func(user uuid.UUID, tz string) {
		ctx := auth.WithUserID(context.Background(), user)
		if tz != "" {
			ctx = auth.WithTimezone(ctx, tz)
		}
		rec := httptest.NewRecorder()
		s.observeTimezone(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/budget/expenses", nil).WithContext(ctx))
	}

	user := uuid.New()
	request(user, "")
	assert.Equal(t, "none", seen(user), "a tz-less token records nothing")

	request(user, "Asia/Manila")
	assert.Equal(t, "Asia/Manila", seen(user))

	request(user, "")
	assert.Equal(t, "Asia/Manila", seen(user), "a tz-less token keeps the last-seen zone")
}
