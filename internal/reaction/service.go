package reaction

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

func (s *Service) SetReaction(ctx context.Context, actorID, messageID uuid.UUID, kind Type) (Reaction, error) {
	switch kind {
	case Like, Love, Laugh, Wow:
	default:
		return Reaction{}, apperror.New(apperror.CodeInvalidArgument, "reaction must be LIKE, LOVE, LAUGH or WOW")
	}
	_, _, participants, err := s.messages.EnsureMessageParticipant(ctx, messageID, actorID)
	if err != nil {
		return Reaction{}, err
	}
	now := time.Now().UTC()
	result, err := s.repository.Upsert(ctx, Reaction{
		MessageID: messageID, UserID: actorID, Type: kind, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return Reaction{}, err
	}
	s.events.PublishTo(participants[:], event.Event{
		ID: uuid.New(), OccurredAt: result.UpdatedAt,
		Payload: event.ReactionChanged{
			MessageID: messageID, UserID: actorID, Reaction: string(result.Type), UpdatedAt: result.UpdatedAt,
		},
	})
	return result, nil
}

func (s *Service) RemoveReaction(ctx context.Context, actorID, messageID uuid.UUID) (bool, error) {
	_, _, participants, err := s.messages.EnsureMessageParticipant(ctx, messageID, actorID)
	if err != nil {
		return false, err
	}
	removed, err := s.repository.Delete(ctx, messageID, actorID)
	if err != nil {
		return false, err
	}
	if removed {
		now := time.Now().UTC()
		s.events.PublishTo(participants[:], event.Event{
			ID: uuid.New(), OccurredAt: now,
			Payload: event.ReactionChanged{MessageID: messageID, UserID: actorID, Removed: true, UpdatedAt: now},
		})
	}
	return removed, nil
}
