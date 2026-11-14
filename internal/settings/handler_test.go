package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/auth"
	db "github.com/ejsadiarin/corefinance/internal/db/sqlc"
	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func call(h http.HandlerFunc, method string, user uuid.UUID, body string) (*httptest.ResponseRecorder, Settings) {
	req := httptest.NewRequest(method, "/api/budget/settings", strings.NewReader(body))
	req = req.WithContext(auth.WithUserID(context.Background(), user))
	rec := httptest.NewRecorder()
	h(rec, req)
	var got Settings
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	return rec, got
}

// A user who never saved settings sees the default (the server clock's
// zone, UTC in deploys) and saved=false, so the web knows to adopt the
// device's zone.
func TestSettings_DefaultThenSavedTimezone_Integration(t *testing.T) {
	h := NewHandler(db.New(testutil.SetupTestDB(t)))
	user := uuid.New()

	rec, got := call(h.Get, http.MethodGet, user, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, Settings{Timezone: "UTC", Saved: false}, got)

	rec, got = call(h.Put, http.MethodPut, user, `{"timezone":"Asia/Manila"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, Settings{Timezone: "Asia/Manila", Saved: true}, got)

	rec, got = call(h.Get, http.MethodGet, user, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, Settings{Timezone: "Asia/Manila", Saved: true}, got)

	_, got = call(h.Get, http.MethodGet, uuid.New(), "")
	assert.False(t, got.Saved, "settings are per user")
}

func TestSettings_RejectsNonIANATimezone_Integration(t *testing.T) {
	h := NewHandler(db.New(testutil.SetupTestDB(t)))
	user := uuid.New()

	for _, tz := range []string{"", "Mars/Olympus", "+08:00", "Local"} {
		rec, _ := call(h.Put, http.MethodPut, user, `{"timezone":"`+tz+`"}`)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "timezone %q", tz)
	}
	_, got := call(h.Get, http.MethodGet, user, "")
	assert.False(t, got.Saved, "a rejected timezone is not stored")
}
