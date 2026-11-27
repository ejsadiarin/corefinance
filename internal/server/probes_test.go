package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ejsadiarin/corefinance/internal/testutil"
)

func probe(t *testing.T, s *Server, path string) (int, map[string]string) {
	t.Helper()
	rec := httptest.NewRecorder()
	s.RegisterRoutes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), rec.Body.String())
	return rec.Code, body
}

// unreachablePool connects lazily to a port nothing listens on, so every
// query fails.
func unreachablePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://u:p@127.0.0.1:1/db?connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}

// Liveness never touches the database, so health checks don't wake it.
func TestHealthIsUpWithoutTheDatabase(t *testing.T) {
	for name, s := range map[string]*Server{
		"no pool":          {},
		"unreachable pool": {Pool: unreachablePool(t)},
	} {
		t.Run(name, func(t *testing.T) {
			code, body := probe(t, s, "/health")
			assert.Equal(t, http.StatusOK, code)
			assert.Equal(t, map[string]string{"status": "up"}, body)
		})
	}
}

func TestReadyIsDownWhenTheDatabaseIsUnreachable(t *testing.T) {
	code, body := probe(t, &Server{Pool: unreachablePool(t)}, "/ready")
	assert.Equal(t, http.StatusServiceUnavailable, code)
	assert.Equal(t, "down", body["status"])
}

func TestReadyIsUpWhenTheDatabaseAnswers_Integration(t *testing.T) {
	code, body := probe(t, &Server{Pool: testutil.SetupTestDB(t)}, "/ready")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "up", body["status"])
}
