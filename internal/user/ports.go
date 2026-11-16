package user

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	GetByID(ctx context.Context, userID uuid.UUID) (User, error)
	FindByNormalizedUsername(ctx context.Context, username string) (User, error)
	Exists(ctx context.Context, userID uuid.UUID) (bool, error)
}

type PresenceResolver interface {
	Resolve(ctx context.Context, userID uuid.UUID) (status string, lastSeenAt *time.Time, err error)
}
