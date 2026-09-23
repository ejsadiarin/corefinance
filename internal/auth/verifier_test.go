package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type jwksServer struct {
	mu   sync.Mutex
	keys map[string]ed25519.PublicKey
}

func (s *jwksServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	type entry struct {
		Kty string `json:"kty"`
		Crv string `json:"crv"`
		Kid string `json:"kid"`
		X   string `json:"x"`
		Use string `json:"use"`
		Alg string `json:"alg"`
	}
	doc := struct {
		Keys []entry `json:"keys"`
	}{}
	for kid, pub := range s.keys {
		doc.Keys = append(doc.Keys, entry{
			Kty: "OKP", Crv: "Ed25519", Kid: kid,
			X:   base64.RawURLEncoding.EncodeToString(pub),
			Use: "sig", Alg: "EdDSA",
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

func genKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return pub, priv
}

func signToken(t *testing.T, priv ed25519.PrivateKey, kid, iss, aud, sub, azp, scope string, iat, exp time.Time) string {
	t.Helper()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    iss,
			Audience:  jwt.ClaimStrings{aud},
			IssuedAt:  jwt.NewNumericDate(iat),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
		Azp:   azp,
		Scope: scope,
	}
	if sub != "" {
		claims.Subject = sub
	}
	// The jwt lib does not set kid automatically; set it on the header.
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	tok.Header["kid"] = kid
	tokenString, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tokenString
}

func setup(t *testing.T) (*Verifier, *jwksServer, ed25519.PrivateKey, string) {
	t.Helper()
	pub, priv := genKey(t)
	const kid = "2026-09-a"
	srv := &jwksServer{keys: map[string]ed25519.PublicKey{kid: pub}}
	httpSrv := httptest.NewServer(srv)
	t.Cleanup(httpSrv.Close)

	v, err := NewVerifier(Config{
		Issuer:      "https://gateway.internal",
		Audience:    "corefinance",
		JWKSURL:     httpSrv.URL,
		PublicPaths: []string{"/api/budget/priority-groups"},
	})
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}
	return v, srv, priv, kid
}

func validToken(t *testing.T, priv ed25519.PrivateKey, kid, sub string) string {
	t.Helper()
	now := time.Now()
	return signToken(t, priv, kid, "https://gateway.internal", "corefinance", sub, "session", "", now, now.Add(5*time.Minute))
}

func TestValidUserToken(t *testing.T) {
	v, _, priv, kid := setup(t)
	userID := uuid.New()
	spoof := uuid.New()

	var ctxUser uuid.UUID
	var ctxOK, nextCalled bool
	var downstreamUserIDHeader string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		ctxUser, ctxOK = UserIDFromContext(r.Context())
		downstreamUserIDHeader = r.Header.Get("X-User-ID")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/budget/expenses/", nil)
	req.Header.Set("Authorization", "Bearer "+validToken(t, priv, kid, userID.String()))
	req.Header.Set("X-User-ID", spoof.String()) // spoof attempt must die here
	v.Middleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if !ctxOK || ctxUser != userID {
		t.Errorf("context user = %v, %v; want %v", ctxUser, ctxOK, userID)
	}
	if downstreamUserIDHeader != "" {
		t.Errorf("X-User-ID reached handler: %q", downstreamUserIDHeader)
	}
}

func TestUserScopesLandInContext(t *testing.T) {
	v, _, priv, kid := setup(t)
	now := time.Now()
	tokenString := signToken(t, priv, kid, "https://gateway.internal", "corefinance", uuid.New().String(), "session", "admin finance:read", now, now.Add(5*time.Minute))

	var scopes []string
	var admin bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scopes = ScopesFromContext(r.Context())
		admin = HasScope(r.Context(), "admin")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/budget/admin/backfill", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	v.Middleware(next).ServeHTTP(httptest.NewRecorder(), req)

	if !admin || len(scopes) != 2 {
		t.Errorf("scopes = %v, admin = %v", scopes, admin)
	}
}

func TestPublicPathBypasses(t *testing.T) {
	v, _, _, _ := setup(t)
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { nextCalled = true })

	req := httptest.NewRequest(http.MethodGet, "/api/budget/priority-groups", nil)
	rec := httptest.NewRecorder()
	v.Middleware(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("public path must reach handler without credentials")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestMissingOrMalformedBearer(t *testing.T) {
	v, _, _, _ := setup(t)
	for _, header := range []string{"", "Bearer", "Bearer ", "Token abc"} {
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { nextCalled = true })
		req := httptest.NewRequest(http.MethodGet, "/api/budget/expenses/", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		v.Middleware(next).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("header %q: status = %d, want 401", header, rec.Code)
		}
		if nextCalled {
			t.Errorf("header %q: next must not be called", header)
		}
	}
}

func TestWrongKeyRejected(t *testing.T) {
	v, _, _, kid := setup(t)
	_, otherPriv := genKey(t)
	req := httptest.NewRequest(http.MethodGet, "/api/budget/expenses/", nil)
	req.Header.Set("Authorization", "Bearer "+validToken(t, otherPriv, kid, uuid.New().String()))
	rec := httptest.NewRecorder()
	v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestWrongIssuerOrAudienceRejected(t *testing.T) {
	v, _, priv, kid := setup(t)
	now := time.Now()
	cases := map[string]string{
		"bad iss": signToken(t, priv, kid, "https://evil.example", "corefinance", uuid.New().String(), "session", "", now, now.Add(5*time.Minute)),
		"bad aud": signToken(t, priv, kid, "https://gateway.internal", "other", uuid.New().String(), "session", "", now, now.Add(5*time.Minute)),
	}
	for name, tokenString := range cases {
		req := httptest.NewRequest(http.MethodGet, "/api/budget/expenses/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rec := httptest.NewRecorder()
		v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", name, rec.Code)
		}
	}
}

func TestExpiryWithLeeway(t *testing.T) {
	v, _, priv, kid := setup(t)
	now := time.Now()

	within := signToken(t, priv, kid, "https://gateway.internal", "corefinance", uuid.New().String(), "session", "", now.Add(-5*time.Minute), now.Add(-30*time.Second))
	req := httptest.NewRequest(http.MethodGet, "/api/budget/expenses/", nil)
	req.Header.Set("Authorization", "Bearer "+within)
	rec := httptest.NewRecorder()
	nextCalled := false
	v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { nextCalled = true })).ServeHTTP(rec, req)
	if !nextCalled {
		t.Error("token expired 30s ago must pass within 60s leeway")
	}

	beyond := signToken(t, priv, kid, "https://gateway.internal", "corefinance", uuid.New().String(), "session", "", now.Add(-5*time.Minute), now.Add(-61*time.Second))
	req2 := httptest.NewRequest(http.MethodGet, "/api/budget/expenses/", nil)
	req2.Header.Set("Authorization", "Bearer "+beyond)
	rec2 := httptest.NewRecorder()
	v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("token expired 61s ago: status = %d, want 401", rec2.Code)
	}
}

func TestUnknownKidRefreshes(t *testing.T) {
	v, srv, _, _ := setup(t)
	pubB, privB := genKey(t)
	now := time.Now()
	tokenString := signToken(t, privB, "2026-09-b", "https://gateway.internal", "corefinance", uuid.New().String(), "session", "", now, now.Add(5*time.Minute))

	// Publish the new key mid-test: the verifier must refresh on unknown kid.
	srv.mu.Lock()
	srv.keys["2026-09-b"] = pubB
	srv.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/budget/expenses/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	nextCalled := false
	v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { nextCalled = true })).ServeHTTP(httptest.NewRecorder(), req)
	if !nextCalled {
		t.Error("token with newly published kid must verify after refresh")
	}
}

func TestServiceTokenHasNoUser(t *testing.T) {
	v, _, priv, kid := setup(t)
	now := time.Now()
	tokenString := signToken(t, priv, kid, "https://gateway.internal", "corefinance", "", "corereminder", "finance:read", now, now.Add(5*time.Minute))

	var svc *ServiceIdentity
	var svcOK, userOK, nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		_, userOK = UserIDFromContext(r.Context())
		svc, svcOK = ServiceFromContext(r.Context())
	})
	v.Middleware(next).ServeHTTP(httptest.NewRecorder(), func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/api/budget/expenses/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		return req
	}())

	if !nextCalled {
		t.Fatal("service token must pass verification")
	}
	if userOK {
		t.Error("service token must not establish a user")
	}
	if !svcOK || svc.Label != "corereminder" || len(svc.Scopes) != 1 || svc.Scopes[0] != "finance:read" {
		t.Errorf("service identity = %+v, %v", svc, svcOK)
	}
}

func TestNewVerifierFailsClosed(t *testing.T) {
	if _, err := NewVerifier(Config{}); err == nil {
		t.Error("expected error for empty config")
	}
	if _, err := NewVerifier(Config{
		Issuer:   "https://gateway.internal",
		Audience: "corefinance",
		JWKSURL:  "http://127.0.0.1:1/unreachable",
	}); err == nil {
		t.Error("expected error when JWKS is unreachable")
	}
}
