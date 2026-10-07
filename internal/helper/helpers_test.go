package helper_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ejsadiarin/corefinance/internal/auth"
	"github.com/ejsadiarin/corefinance/internal/helper"
	"github.com/google/uuid"
)

func TestRespond(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		w      http.ResponseWriter
		status int
		data   interface{}
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper.RespondJSON(tt.w, tt.status, tt.data)
		})
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	userID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithUserID(req.Context(), userID))
	if got, ok := helper.GetUserID(httptest.NewRecorder(), req); !ok || got != userID {
		t.Errorf("got %v, %v; want %v, true", got, ok, userID)
	}
}

func TestGetUserIDAnonymousRejected(t *testing.T) {
	rec := httptest.NewRecorder()
	if _, ok := helper.GetUserID(rec, httptest.NewRequest(http.MethodGet, "/", nil)); ok {
		t.Error("anonymous request must not satisfy user endpoints")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
