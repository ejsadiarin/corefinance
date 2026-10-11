package auth

import (
	"crypto/ed25519"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// signTokenWithActAs mirrors signToken with an act_as claim.
func signTokenWithActAs(t *testing.T, priv ed25519.PrivateKey, kid, sub, scope, actAs string, iat, exp time.Time) string {
	t.Helper()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://gateway.internal",
			Audience:  jwt.ClaimStrings{"corefinance"},
			IssuedAt:  jwt.NewNumericDate(iat),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
		Azp:   "session",
		Scope: scope,
		ActAs: actAs,
	}
	if sub != "" {
		claims.Subject = sub
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tok.Header["kid"] = kid
	tokenString, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tokenString
}

func TestActAsClaimLandsInContext(t *testing.T) {
	v, _, priv, kid := setup(t)
	now := time.Now()
	self := uuid.New()
	other := uuid.New()
	tokenString := signTokenWithActAs(t, priv, kid, self.String(), "admin", other.String(), now, now.Add(5*time.Minute))

	var gotActAs, gotUser uuid.UUID
	var gotScopes []string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		gotActAs, ok = ActAsFromContext(r.Context())
		if !ok {
			t.Error("act_as missing from context")
		}
		gotScopes = ScopesFromContext(r.Context())
		gotUser, _ = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/budget/remaining", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()
	v.Middleware(next).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if gotUser != self {
		t.Errorf("sub = %v, want %v", gotUser, self)
	}
	if gotActAs != other {
		t.Errorf("act_as = %v, want %v", gotActAs, other)
	}
	found := false
	for _, s := range gotScopes {
		if s == "admin" {
			found = true
		}
	}
	if !found {
		t.Errorf("scopes = %v, want admin", gotScopes)
	}
}
