package message

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/francesandprincesss/new-telegram/internal/pagination"
	"github.com/google/uuid"
)

const MaxBodyLength = 4000

type Service struct {
	repository Repository
	dialogs    DialogAccess
	events     Publisher
}

func NewService(repository Repository, dialogs DialogAccess, events Publisher) *Service {
	return &Service{repository: repository, dialogs: dialogs, events: events}
}

func (s *Service) SendMessage(ctx context.Context, actorID, dialogID uuid.UUID, body string, replyToID *uuid.UUID) (Message, error) {
	if !utf8.ValidString(body) || strings.ContainsRune(body, 0) || strings.TrimSpace(body) == "" || utf8.RuneCountInString(body) > MaxBodyLength {
		return Message{}, apperror.New(apperror.CodeInvalidArgument, "body must have 1–4000 characters and cannot be blank")
	}
	if err := s.dialogs.EnsureParticipant(ctx, dialogID, actorID); err != nil {
		return Message{}, err
	}
	participants, err := s.dialogs.Participants(ctx, dialogID)
	if err != nil {
		return Message{}, err
	}
	result, err := s.repository.CreateAndTouchDialog(ctx, NewMessage{
		ID: uuid.New(), DialogID: dialogID, SenderID: actorID, Body: body, CreatedAt: time.Now().UTC(), ReplyToID: replyToID,
	})
	if err != nil {
		return Message{}, err
	}
	s.events.PublishTo(participants[:], event.Event{
		ID: uuid.New(), OccurredAt: result.CreatedAt,
		Payload: event.MessageCreated{
			MessageID: result.ID, DialogID: result.DialogID,
			SenderID: result.SenderID, Body: result.Body, CreatedAt: result.CreatedAt, ReplyToID: replyToID,
		},
	})
	return result, nil
}

func (s *Service) ListMessages(ctx context.Context, actorID, dialogID uuid.UUID, pageSize int, pageToken string) ([]Message, string, error) {
	size, err := pagination.Size(pageSize)
	if err != nil {
		return nil, "", err
	}
	var cursor *Cursor
	if pageToken != "" {
		timestamp, id, err := pagination.Decode(pageToken)
		if err != nil {
			return nil, "", err
		}
		cursor = &Cursor{CreatedAt: timestamp, ID: id}
	}
	if err := s.dialogs.EnsureParticipant(ctx, dialogID, actorID); err != nil {
		return nil, "", err
	}
	messages, err := s.repository.ListByDialog(ctx, dialogID, cursor, size+1)
	if err != nil {
		return nil, "", err
	}
	if len(messages) <= size {
		return messages, "", nil
	}
	last := messages[size-1]
	return messages[:size], pagination.Encode(last.CreatedAt, last.ID), nil
}

func (s *Service) DialogSummary(ctx context.Context, actorID, dialogID uuid.UUID) (*Message, int64, error) {
	if err := s.dialogs.EnsureParticipant(ctx, dialogID, actorID); err != nil {
		return nil, 0, err
	}
	messages, err := s.repository.ListByDialog(ctx, dialogID, nil, 1)
	if err != nil {
		return nil, 0, err
	}
	if len(messages) == 0 {
		return nil, 0, nil
	}
	count, err := s.repository.CountUnread(ctx, dialogID, actorID)
	if err != nil {
		return nil, 0, err
	}
	return &messages[0], count, nil
}

// EnsureMessageParticipant exposes only neutral UUID values so receipt and
// reaction modules can depend on it without cyclic package imports.
func (s *Service) EnsureMessageParticipant(
	ctx context.Context,
	messageID uuid.UUID,
	userID uuid.UUID,
) (dialogID uuid.UUID, senderID uuid.UUID, participants [2]uuid.UUID, err error) {
	message, err := s.repository.GetMetadataByID(ctx, messageID)
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
