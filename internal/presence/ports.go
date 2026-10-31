package presence

import (
	"context"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/google/uuid"
)

type Repository interface {
	ManualStatus(ctx context.Context, userID uuid.UUID) (Status, error)
	SetManualStatus(ctx context.Context, userID uuid.UUID, status Status) error
	SetLastSeen(ctx context.Context, userID uuid.UUID, seenAt time.Time) error
}

type PeerLookup interface {
	PeerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type Publisher interface {
	PublishTo(userIDs []uuid.UUID, event event.Event)
}
