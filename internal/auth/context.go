package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const userKey contextKey = "user_id"

// WithUserID stores the user ID in the context.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userKey, id)
}

// UserIDFromContext extracts the user ID from the context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userKey).(uuid.UUID)
	return id, ok
}

// GetUserID is a convenience handler that extracts the user ID from the
// X-User-ID header and returns it. This is used when coregateway passes
// the user ID via header (inter-service communication).
func GetUserID(r *http.Request) (uuid.UUID, bool) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, false
	}
	return userID, true
}
