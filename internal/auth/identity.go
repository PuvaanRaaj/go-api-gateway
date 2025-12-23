package auth

import (
	"context"

	"github.com/google/uuid"
)

// Identity captures the authenticated user details.
type Identity struct {
	UserID uuid.UUID
	Email  string
	Method string
}

type contextKey string

const identityContextKey contextKey = "gateway.identity"

// WithIdentity stores the identity in context.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityContextKey, id)
}

// FromContext extracts the identity from context when present.
func FromContext(ctx context.Context) (Identity, bool) {
	val := ctx.Value(identityContextKey)
	if val == nil {
		return Identity{}, false
	}
	id, ok := val.(Identity)
	return id, ok
}
