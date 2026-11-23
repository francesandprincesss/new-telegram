package message

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/event"
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

func (s *Service) SendMessage(ctx context.Context, dialogID, sendlerID uuid.UUID, body string) (Message, error) {
	if dialogID == uuid.Nil || sendlerID == uuid.Nil {
		return Message{}, apperror.New(apperror.CodeInvalidArgument, "invalid dialog or sendler")
	}

	if utf8.RuneCountInString(body) > 4000 || strings.TrimSpace(body) == "" {
		return Message{}, apperror.New(apperror.CodeInvalidArgument, "invalid message body")
	}

	if err := s.dialogs.EnsureParticipant(ctx, dialogID, sendlerID); err != nil {
		return Message{}, err
	}

	msg := NewMessage{
		ID: uuid.New(),
		DialogID: dialogID,
		SenderID: sendlerID,
		Body: body,
		CreatedAt: time.Now().UTC(),
	}

	msgData, err := s.dialogs.Participants(ctx, msg.DialogID); if err != nil {
		return Message{}, apperror.Wrap(apperror.CodeInternal, "cannot have participant dialog", err)
	}

	newMsg, err := s.repository.CreateAndTouchDialog(ctx, msg); if err != nil {
		return Message{}, apperror.Wrap(apperror.CodeInternal, "invalid create message", err)
	}

	evt := event.Event{
		ID: uuid.New(),
		OccurredAt: time.Now().UTC(),
		Payload: event.MessageCreated{
			MessageID: newMsg.ID,
			DialogID: newMsg.DialogID,
			SenderID: newMsg.SenderID,
			Body: newMsg.Body,
			CreatedAt: newMsg.CreatedAt,
		},
	}

	reciverID := msgData[0]
	if reciverID == sendlerID {
		reciverID = msgData[1]
	}

	s.events.PublishTo([]uuid.UUID{reciverID}, evt)

	return newMsg, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, dialogID uuid.UUID, cursor *Cursor, limit int) ([]Message, error) {
	if userID == uuid.Nil || dialogID == uuid.Nil {
		return nil, apperror.New(apperror.CodeInvalidArgument, "invalid userId or dialogID")
	}

	if limit == 0 {
		limit = 15
	}

	if limit < 1 || limit > 15 {
		return nil, apperror.New(apperror.CodeInvalidArgument, "invalid limit message")
	}

	if err := s.dialogs.EnsureParticipant(ctx, dialogID, userID); err != nil {
		return nil, err
	}

	messages, err := s.repository.ListByDialog(ctx, dialogID, cursor, limit+1); if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "invalid get messages", err)
	}

	return messages, nil
}