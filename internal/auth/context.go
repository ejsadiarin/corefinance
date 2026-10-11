package auth

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	userKey   contextKey = "user_id"
	scopesKey contextKey = "scopes"
	actAsKey  contextKey = "act_as"
)

// WithUserID stores the user ID in the context.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userKey, id)
}

// UserIDFromContext extracts the user ID from the context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userKey).(uuid.UUID)
	return id, ok
}

// WithScopes stores the verified scope list (space-delimited scope claim,
// split by the verifier) in the context.
func WithScopes(ctx context.Context, scopes []string) context.Context {
	return context.WithValue(ctx, scopesKey, scopes)
}

// ScopesFromContext extracts the verified scope list; empty when none.
func ScopesFromContext(ctx context.Context) []string {
	scopes, _ := ctx.Value(scopesKey).([]string)
	return scopes
}

// HasScope reports whether the verified scope list contains scope.
func HasScope(ctx context.Context, scope string) bool {
	for _, s := range ScopesFromContext(ctx) {
		if s == scope {
			return true
		}
	}
	return false
}

// WithActAs stores the gateway-asserted target user (the signed act_as
// claim) in the context. It is honored only with admin scope.
func WithActAs(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, actAsKey, id)
}

// ActAsFromContext extracts the gateway-asserted target user; false when
// the token carries no act_as claim.
func ActAsFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(actAsKey).(uuid.UUID)
	return id, ok
}
