package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ejsadiarin/corefinance/internal/auth"
	"github.com/google/uuid"
)

func requestAs(userID uuid.UUID, scopes []string, actAs *uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/budget/remaining", nil)
	ctx := auth.WithUserID(req.Context(), userID)
	ctx = auth.WithScopes(ctx, scopes)
	if actAs != nil {
		ctx = auth.WithActAs(ctx, *actAs)
	}
	return req.WithContext(ctx)
}

func TestResolveTargetUser(t *testing.T) {
	self := uuid.New()
	other := uuid.New()

	t.Run("absent claim resolves sub", func(t *testing.T) {
		rec := httptest.NewRecorder()
		got, ok := ResolveTargetUser(rec, requestAs(self, []string{"admin"}, nil))
		if !ok || got != self {
			t.Errorf("got %v, %v; want %v, true", got, ok, self)
		}
	})

	t.Run("admin with claim resolves target", func(t *testing.T) {
		rec := httptest.NewRecorder()
		got, ok := ResolveTargetUser(rec, requestAs(self, []string{"admin"}, &other))
		if !ok || got != other {
			t.Errorf("got %v, %v; want %v, true", got, ok, other)
		}
	})

	t.Run("non-admin with claim falls back to sub", func(t *testing.T) {
		rec := httptest.NewRecorder()
		got, ok := ResolveTargetUser(rec, requestAs(self, []string{"corefinance:read"}, &other))
		if !ok || got != self {
			t.Errorf("got %v, %v; want %v, true (smuggled claim ignored)", got, ok, self)
		}
	})

	t.Run("no identity is 401", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/budget/remaining", nil)
		if _, ok := ResolveTargetUser(rec, req); ok {
			t.Error("ok = true, want false")
		}
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})
}
