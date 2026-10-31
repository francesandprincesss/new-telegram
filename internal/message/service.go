package message

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repository Repository
	dialogs    DialogAccess
	events     Publisher
}

func NewService(repository Repository, dialogs DialogAccess, events Publisher) *Service {
	return &Service{repository: repository, dialogs: dialogs, events: events}
}

// EnsureMessageParticipant exposes only neutral UUID values so receipt and
// reaction modules can depend on it without cyclic package imports.
func (s *Service) EnsureMessageParticipant(
	ctx context.Context,
	messageID uuid.UUID,
	userID uuid.UUID,
) (dialogID uuid.UUID, senderID uuid.UUID, participants [2]uuid.UUID, err error) {
	message, err := s.repository.GetByID(ctx, messageID)
	if err != nil {
		return uuid.Nil, uuid.Nil, participants, err
	}
	if err := s.dialogs.EnsureParticipant(ctx, message.DialogID, userID); err != nil {
		return uuid.Nil, uuid.Nil, participants, err
	}
	participants, err = s.dialogs.Participants(ctx, message.DialogID)
	if err != nil {
		return uuid.Nil, uuid.Nil, participants, err
	}
	return message.DialogID, message.SenderID, participants, nil
}

// TODO: Implement sending, body validation, history pagination, and event
// publication strictly after a successful database commit.
