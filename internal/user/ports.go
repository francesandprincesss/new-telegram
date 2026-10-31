package user

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	GetByID(ctx context.Context, userID uuid.UUID) (User, error)
	FindByNormalizedUsername(ctx context.Context, username string) (User, error)
	Exists(ctx context.Context, userID uuid.UUID) (bool, error)
}
