package reaction

import (
	"context"

	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/google/uuid"
)

type Repository interface {
	Upsert(ctx context.Context, reaction Reaction) (Reaction, error)
	Delete(ctx context.Context, messageID, userID uuid.UUID) (bool, error)
}

type MessageAccess interface {
	EnsureMessageParticipant(
		ctx context.Context,
		messageID, userID uuid.UUID,
	) (dialogID, senderID uuid.UUID, participants [2]uuid.UUID, err error)
}

type Publisher interface {
	PublishTo(userIDs []uuid.UUID, event event.Event)
}
