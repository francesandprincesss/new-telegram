package event

import (
	"context"

	"github.com/google/uuid"
)

// ConnectionObserver lets presence track stream lifecycle without event
// importing the presence package.
type ConnectionObserver interface {
	Connected(ctx context.Context, userID uuid.UUID) error
	Disconnected(ctx context.Context, userID uuid.UUID) error
}
