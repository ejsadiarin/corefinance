package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The worker's timezone lookup (#18) against coregateway's
// GET /api/users/{id}/timezone (ADR-0005).

func fakeGateway(t *testing.T, key string, zones map[string]string, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+key {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"no credential"}`))
			return
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":"nope"}`))
			return
		}
		id := r.PathValue("id")
		tz, saved := zones[id]
		if !saved {
			tz = "UTC"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user_id":"` + id + `","timezone":"` + tz + `","saved":` + map[bool]string{true: "true", false: "false"}[saved] + `}`))
	}))
	mux := http.NewServeMux()
	mux.Handle("GET /api/users/{id}/timezone", srv.Config.Handler)
	srv.Config.Handler = mux
	t.Cleanup(srv.Close)
	return srv
}

func TestZone_SavedAndUnsetUsers(t *testing.T) {
	manila := uuid.New()
	srv := fakeGateway(t, "k", map[string]string{manila.String(): "Asia/Manila"}, http.StatusOK)
	c := NewTimezoneClient(srv.URL, "k")

	loc, err := c.Zone(context.Background(), manila)
	require.NoError(t, err)
	assert.Equal(t, "Asia/Manila", loc.String())

	loc, err = c.Zone(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "UTC", loc.String(), "a user who hasn't set one is on UTC, not an error")
}

func TestZone_FailedLookupsAreErrors(t *testing.T) {
	for name, c := range map[string]*TimezoneClient{
		"wrong key":   NewTimezoneClient(fakeGateway(t, "k", nil, http.StatusOK).URL, "other"),
		"forbidden":   NewTimezoneClient(fakeGateway(t, "k", nil, http.StatusForbidden).URL, "k"),
		"server down": NewTimezoneClient(fakeGateway(t, "k", nil, http.StatusBadGateway).URL, "k"),
		"unreachable": NewTimezoneClient("http://127.0.0.1:1", "k"),
	} {
		_, err := c.Zone(context.Background(), uuid.New())
		assert.Error(t, err, name)
	}
}

func TestZone_UnknownZoneIsAnError(t *testing.T) {
	user := uuid.New()
	srv := fakeGateway(t, "k", map[string]string{user.String(): "Mars/Olympus"}, http.StatusOK)
	_, err := NewTimezoneClient(srv.URL, "k").Zone(context.Background(), user)
	assert.Error(t, err, "a zone this build can't load is not guessed")
}
