package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// The gateway asserts the timezone of the user a request is about in the
// internal token's tz claim (coregateway ADR-0005, corefinance #18).
func TestTimezoneClaimLandsInContext(t *testing.T) {
	v, _, priv, kid := setup(t)
	now := time.Now()
	for name, tc := range map[string]struct{ tz, want string }{
		"tz claim":           {"Asia/Manila", "Asia/Manila"},
		"older token, no tz": {"", ""},
	} {
		claims := Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer: "https://gateway.internal", Audience: jwt.ClaimStrings{"corefinance"},
				Subject: uuid.NewString(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
			},
			Azp: "session", Tz: tc.tz,
		}
		tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
		tok.Header["kid"] = kid
		signed, err := tok.SignedString(priv)
		if err != nil {
			t.Fatal(err)
		}
		var got string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = TimezoneFromContext(r.Context()) })
		req := httptest.NewRequest(http.MethodGet, "/api/budget/remaining", nil)
		req.Header.Set("Authorization", "Bearer "+signed)
		v.Middleware(next).ServeHTTP(httptest.NewRecorder(), req)
		if got != tc.want {
			t.Errorf("%s: tz = %q, want %q", name, got, tc.want)
		}
	}
}
