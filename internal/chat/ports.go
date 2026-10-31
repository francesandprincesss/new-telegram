package chat

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	CreateOrGetDirect(ctx context.Context, firstUserID, secondUserID uuid.UUID) (Dialog, error)
	ListForUser(ctx context.Context, userID uuid.UUID, cursor *Cursor, limit int) ([]Dialog, error)
	EnsureParticipant(ctx context.Context, dialogID, userID uuid.UUID) error
	Participants(ctx context.Context, dialogID uuid.UUID) ([2]uuid.UUID, error)
	PeerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type UserLookup interface {
	UserExists(ctx context.Context, userID uuid.UUID) (bool, error)
}
