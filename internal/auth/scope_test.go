package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestRequireServiceScope(t *testing.T) {
	pass := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	withService := func(scopes []string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/internal/recurring/active", nil)
		return req.WithContext(WithService(req.Context(), "corereminder", scopes))
	}

	rec := httptest.NewRecorder()
	RequireServiceScope("finance:read")(pass).ServeHTTP(rec, withService([]string{"finance:read"}))
	if rec.Code != http.StatusNoContent {
		t.Errorf("scoped service: status = %d, want 204", rec.Code)
	}

	rec = httptest.NewRecorder()
	RequireServiceScope("finance:read")(pass).ServeHTTP(rec, withService([]string{"other"}))
	if rec.Code != http.StatusForbidden {
		t.Errorf("wrong scope: status = %d, want 403", rec.Code)
	}

	// User identity without service identity fails by design.
	userReq := httptest.NewRequest(http.MethodGet, "/internal/recurring/active", nil)
	userReq = userReq.WithContext(WithUserID(userReq.Context(), uuid.New()))
	rec = httptest.NewRecorder()
	RequireServiceScope("finance:read")(pass).ServeHTTP(rec, userReq)
	if rec.Code != http.StatusForbidden {
		t.Errorf("user token on service route: status = %d, want 403", rec.Code)
	}

	rec = httptest.NewRecorder()
	RequireServiceScope("finance:read")(pass).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusForbidden {
		t.Errorf("no identity: status = %d, want 403", rec.Code)
	}
}

func TestServiceIdentityRoundTrip(t *testing.T) {
	ctx := WithService(context.Background(), "corereminder", []string{"finance:read"})
	svc, ok := ServiceFromContext(ctx)
	if !ok || svc.Label != "corereminder" || len(svc.Scopes) != 1 {
		t.Errorf("service = %+v, %v", svc, ok)
	}
	if _, ok := ServiceFromContext(context.Background()); ok {
		t.Error("empty context must not yield a service")
	}
}
