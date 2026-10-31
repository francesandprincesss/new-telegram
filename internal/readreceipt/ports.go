package readreceipt

import (
	"context"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/google/uuid"
)

type Repository interface {
	MarkThrough(
		ctx context.Context,
		dialogID, readerID, upToMessageID uuid.UUID,
		readAt time.Time,
	) (MarkResult, error)
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
