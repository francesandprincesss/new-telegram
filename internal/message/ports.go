package message

import (
	"context"

	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/google/uuid"
)

type Repository interface {
	// CreateAndTouchDialog inserts the message and updates dialogs.updated_at in one transaction.
	CreateAndTouchDialog(ctx context.Context, message NewMessage) (Message, error)
	ListByDialog(ctx context.Context, dialogID uuid.UUID, cursor *Cursor, limit int) ([]Message, error)
	GetByID(ctx context.Context, messageID uuid.UUID) (Message, error)
}

type DialogAccess interface {
	EnsureParticipant(ctx context.Context, dialogID, userID uuid.UUID) error
	Participants(ctx context.Context, dialogID uuid.UUID) ([2]uuid.UUID, error)
}

type Publisher interface {
	PublishTo(userIDs []uuid.UUID, event event.Event)
}
