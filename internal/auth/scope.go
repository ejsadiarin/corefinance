package auth

import (
	"net/http"
)

// RequireServiceScope rejects requests without an established service
// identity carrying scope. User JWTs have no service identity and fail
// here by design: user and service authorization domains do not cross.
// Missing identity and insufficient scope both report 403 — identity was
// verified (or the request would have stopped at 401), privilege was not.
func RequireServiceScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			svc, ok := ServiceFromContext(r.Context())
			if !ok {
				forbidden(w, "service credentials required")
				return
			}
			for _, s := range svc.Scopes {
				if s == scope {
					next.ServeHTTP(w, r)
					return
				}
			}
			forbidden(w, "insufficient scope")
		})
	}
}

func forbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}
