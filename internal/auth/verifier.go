package auth

import (
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
	"golang.org/x/sync/singleflight"
)

// Claims mirrors the gateway's internal JWT envelope.
type Claims struct {
	jwt.RegisteredClaims
	Azp   string `json:"azp"`
	Scope string `json:"scope,omitempty"`
	ActAs string `json:"act_as,omitempty"`
}

// clockSkewLeeway tolerates clock differences between gateway and verifiers.
const clockSkewLeeway = 60 * time.Second

// defaultCacheTTL bounds how long fetched JWKS keys are trusted without refresh.
const defaultCacheTTL = 5 * time.Minute

// negativeCacheTTL bounds how long an unknown kid is rejected without a
// fetch after a refresh succeeded yet the kid was still absent
// (forgery/mis-issue). Fetch failures never write negative entries, so a
// JWKS outage cannot poison rotation; success never writes them either.
const negativeCacheTTL = 30 * time.Second

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
	// etag is the validator from the last 200 response; empty before
	// the first fetch. Guarded by mu.
	etag   string
	public map[string]bool
	// negative caches unknown kids (expiry per kid) seen absent after a
	// successful refresh. Guarded by mu.
	negative map[string]time.Time
	// sf coalesces concurrent refresh-triggering verifications into one
	// in-flight JWKS fetch.
	sf singleflight.Group
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
		cfg:      cfg,
		keys:     map[string]ed25519.PublicKey{},
		public:   map[string]bool{},
		negative: map[string]time.Time{},
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

// fetch retrieves the JWKS. After the first success it sends the stored
// ETag, so an unchanged key set costs a 304 and no body; notModified then
// tells the caller to keep the keys it already has. A 304 with no stored
// validator cannot come from a compliant gateway and is an error, not an
// empty key set.
func (v *Verifier) fetch() (keys map[string]ed25519.PublicKey, etag string, notModified bool, err error) {
	req, err := http.NewRequest(http.MethodGet, v.cfg.JWKSURL, nil)
	if err != nil {
		return nil, "", false, err
	}
	v.mu.RLock()
	stored := v.etag
	v.mu.RUnlock()
	if stored != "" {
		req.Header.Set("If-None-Match", stored)
	}

	resp, err := v.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		if stored == "" {
			return nil, "", false, fmt.Errorf("JWKS 304 without a cached validator")
		}
		return nil, stored, true, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", false, fmt.Errorf("JWKS status %d", resp.StatusCode)
	}

	var doc jwksDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, "", false, err
	}
	keys = make(map[string]ed25519.PublicKey, len(doc.Keys))
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
		return nil, "", false, fmt.Errorf("JWKS contains no usable Ed25519 keys")
	}
	return keys, resp.Header.Get("ETag"), false, nil
}

func (v *Verifier) refresh() error {
	keys, etag, notModified, err := v.fetch()
	if err != nil {
		return err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if !notModified {
		v.keys = keys
	}
	if etag != "" {
		v.etag = etag
	}
	v.fetchedAt = time.Now()
	return nil
}

func (v *Verifier) keyFor(kid string) (ed25519.PublicKey, bool) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	fresh := time.Since(v.fetchedAt) < v.cfg.CacheTTL
	if ok && fresh {
		v.mu.RUnlock()
		return key, true
	}
	if until, neg := v.negative[kid]; neg && time.Now().Before(until) {
		v.mu.RUnlock()
		return nil, false
	}
	v.mu.RUnlock()

	// Unknown kid or stale cache: exactly one refresh attempt, coalesced
	// with concurrent verifications into a single in-flight JWKS fetch.
	// On fetch failure the caller gets a 401 and the next request retries;
	// nothing is cached.
	_, err, _ := v.sf.Do("jwks-refresh", func() (any, error) {
		return nil, v.refresh()
	})
	if err != nil {
		return nil, false
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	key, ok = v.keys[kid]
	if !ok {
		// Refresh succeeded but the kid is still absent: negative-cache it.
		now := time.Now()
		for k, until := range v.negative {
			if !now.Before(until) {
				delete(v.negative, k)
			}
		}
		v.negative[kid] = now.Add(negativeCacheTTL)
		return nil, false
	}
	// Success: no negative entry is written; drop any stale one.
	delete(v.negative, kid)
	return key, true
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
			// A malformed act_as is ignored (falls back to sub): only the
			// gateway can sign claims, so this is a gateway bug, not an
			// attack — and sub remains the trustworthy identity.
			if actAs, err := uuid.Parse(claims.ActAs); err == nil && actAs != uuid.Nil {
				ctx = WithActAs(ctx, actAs)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Service tokens no longer exist (no minter): a token without a
		// subject satisfies no endpoint.
		unauthorized(w)
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
