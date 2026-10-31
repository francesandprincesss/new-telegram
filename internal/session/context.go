// Package session carries the authenticated identity through request contexts.
package session

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type userIDKey struct{}

var ErrUnauthenticated = errors.New("authenticated user is missing from context")

// WithUserID returns a child context containing a verified user ID.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	if ctx == nil {
		panic("session: nil context")
	}
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserID returns the verified user ID added by the authentication interceptor.
func UserID(ctx context.Context) (uuid.UUID, error) {
	if ctx == nil {
		return uuid.Nil, ErrUnauthenticated
	}
	userID, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, ErrUnauthenticated
	}
	return userID, nil
}
