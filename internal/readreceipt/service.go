package readreceipt

import (
	"context"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/google/uuid"
)

type Service struct {
	repository Repository
	messages   MessageAccess
	events     Publisher
}

func NewService(repository Repository, messages MessageAccess, events Publisher) *Service {
	return &Service{repository: repository, messages: messages, events: events}
}

func (s *Service) MarkRead(ctx context.Context, actorID, dialogID, upToMessageID uuid.UUID) (MarkResult, error) {
	messageDialogID, senderID, participants, err := s.messages.EnsureMessageParticipant(ctx, upToMessageID, actorID)
	if err != nil {
		return MarkResult{}, err
	}
	if messageDialogID != dialogID {
		return MarkResult{}, apperror.New(apperror.CodeInvalidArgument, "message does not belong to the dialog")
	}
	if senderID == actorID {
		return MarkResult{}, apperror.New(apperror.CodeInvalidArgument, "cannot mark your own message as read")
	}
	result, err := s.repository.MarkThrough(ctx, dialogID, actorID, upToMessageID, time.Now().UTC())
	if err != nil {
		return MarkResult{}, err
	}
	if result.MarkedCount > 0 {
		s.events.PublishTo(participants[:], event.Event{
			ID: uuid.New(), OccurredAt: result.ReadAt,
			Payload: event.MessagesRead{
				DialogID: dialogID, ReaderID: actorID, UpToMessageID: upToMessageID, ReadAt: result.ReadAt,
			},
		})
	}
	return result, nil
}
