package auth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims mirrors the gateway's internal JWT envelope.
type Claims struct {
	jwt.RegisteredClaims
	Azp   string `json:"azp"`
	Scope string `json:"scope,omitempty"`
}

// clockSkewLeeway tolerates clock differences between gateway and verifiers.
const clockSkewLeeway = 60 * time.Second

// defaultCacheTTL bounds how long fetched JWKS keys are trusted without refresh.
const defaultCacheTTL = 5 * time.Minute

// ServiceIdentity is a non-user caller (e.g. another service) established
// by a service JWT. It carries no user and must never satisfy user-scoped
// endpoints.
type ServiceIdentity struct {
	Label  string
	Scopes []string
}

type serviceKey string

const serviceContextKey serviceKey = "service"

// WithService stores the service identity in the context.
func WithService(ctx context.Context, label string, scopes []string) context.Context {
	return context.WithValue(ctx, serviceContextKey, &ServiceIdentity{Label: label, Scopes: scopes})
}

// ServiceFromContext extracts the service identity from the context.
func ServiceFromContext(ctx context.Context) (*ServiceIdentity, bool) {
	svc, ok := ctx.Value(serviceContextKey).(*ServiceIdentity)
	return svc, ok
}

// Config configures JWT verification. The gateway is the sole signer;
// the verifier holds public material only.
type Config struct {
	Issuer      string
	Audience    string
	JWKSURL     string
	CacheTTL    time.Duration
	HTTPClient  *http.Client
	PublicPaths []string
}

// Verifier validates internal JWTs against the gateway JWKS and injects
// the established identity into the request context.
type Verifier struct {
	cfg       Config
	mu        sync.RWMutex
	keys      map[string]ed25519.PublicKey
	fetchedAt time.Time
	public    map[string]bool
}

// NewVerifier builds a Verifier and performs the initial JWKS fetch.
// It fails closed: without keys the service refuses to start.
func NewVerifier(cfg Config) (*Verifier, error) {
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("auth: JWT issuer is required")
	}
	if cfg.Audience == "" {
		return nil, fmt.Errorf("auth: JWT audience is required")
	}
	if cfg.JWKSURL == "" {
		return nil, fmt.Errorf("auth: JWKS URL is required")
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = defaultCacheTTL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	v := &Verifier{
		cfg:    cfg,
		keys:   map[string]ed25519.PublicKey{},
		public: map[string]bool{},
	}
	for _, p := range cfg.PublicPaths {
		v.public[p] = true
	}
	if err := v.refresh(); err != nil {
		return nil, fmt.Errorf("auth: initial JWKS fetch: %w", err)
	}
	return v, nil
}

type jwksDocument struct {
	Keys []struct {
		Kty string `json:"kty"`
		Crv string `json:"crv"`
		Kid string `json:"kid"`
		X   string `json:"x"`
		Use string `json:"use"`
		Alg string `json:"alg"`
	} `json:"keys"`
}

func (v *Verifier) fetch() (map[string]ed25519.PublicKey, error) {
	resp, err := v.cfg.HTTPClient.Get(v.cfg.JWKSURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS status %d", resp.StatusCode)
	}
	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	keys := make(map[string]ed25519.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "OKP" || k.Crv != "Ed25519" || k.Kid == "" {
			continue
		}
		raw, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			continue
		}
		keys[k.Kid] = ed25519.PublicKey(raw)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("JWKS contains no usable Ed25519 keys")
	}
	return keys, nil
}

func (v *Verifier) refresh() error {
	keys, err := v.fetch()
	if err != nil {
		return err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.keys = keys
	v.fetchedAt = time.Now()
	return nil
}

func (v *Verifier) keyFor(kid string) (ed25519.PublicKey, bool) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	fresh := time.Since(v.fetchedAt) < v.cfg.CacheTTL
	v.mu.RUnlock()
	if ok && fresh {
		return key, true
	}
	// Unknown kid or stale cache: exactly one refresh attempt.
	if err := v.refresh(); err != nil {
		return nil, false
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok = v.keys[kid]
	return key, ok
}

// Middleware verifies the internal JWT, strips any untrusted X-User-ID
// header, and injects the established identity into the context.
// Requests without valid credentials are rejected with 401 and never
// reach handlers.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// X-User-ID is never a trust source; drop it on every request.
		r.Header.Del("X-User-ID")

		if v.public[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		presented, ok := bearerToken(r)
		if !ok {
			unauthorized(w)
			return
		}

		var claims Claims
		parsed, err := jwt.ParseWithClaims(presented, &claims,
			func(t *jwt.Token) (any, error) {
				if t.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
					return nil, jwt.ErrTokenSignatureInvalid
				}
				kid, _ := t.Header["kid"].(string)
				if kid == "" {
					return nil, jwt.ErrTokenUnverifiable
				}
				key, ok := v.keyFor(kid)
				if !ok {
					return nil, jwt.ErrTokenUnverifiable
				}
				return key, nil
			},
			jwt.WithIssuer(v.cfg.Issuer),
			jwt.WithAudience(v.cfg.Audience),
			jwt.WithLeeway(clockSkewLeeway),
			jwt.WithExpirationRequired(),
		)
		if err != nil || !parsed.Valid {
			unauthorized(w)
			return
		}

		if claims.Subject != "" {
			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				unauthorized(w)
				return
			}
			ctx := WithUserID(r.Context(), userID)
			ctx = WithScopes(ctx, strings.Fields(claims.Scope))
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Service token: identity without a user. User-scoped endpoints
		// resolve no user from this context and reject with 401.
		next.ServeHTTP(w, r.WithContext(WithService(r.Context(), claims.Azp, strings.Fields(claims.Scope))))
	})
}

func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"valid internal credentials are required"}`))
}
