package message

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/google/uuid"
)

type sendRepository struct {
	Repository
	calls int
	err   error
}

func (r *sendRepository) CreateAndTouchDialog(_ context.Context, input NewMessage) (Message, error) {
	r.calls++
	return Message{
		ID: input.ID, DialogID: input.DialogID, SenderID: input.SenderID,
		Body: input.Body, CreatedAt: input.CreatedAt,
	}, r.err
}

type dialogAccessStub struct {
	participants [2]uuid.UUID
}

func (s dialogAccessStub) EnsureParticipant(_ context.Context, _, userID uuid.UUID) error {
	if userID != s.participants[0] && userID != s.participants[1] {
		return apperror.New(apperror.CodeForbidden, "dialog access denied")
	}
	return nil
}

func (s dialogAccessStub) Participants(_ context.Context, _ uuid.UUID) ([2]uuid.UUID, error) {
	return s.participants, nil
}

type recordedEvent struct {
	recipients []uuid.UUID
	values     []event.Event
}

func (s *recordedEvent) PublishTo(recipients []uuid.UUID, value event.Event) {
	s.recipients = recipients
	s.values = append(s.values, value)
}

func TestSendDoesNotPublishBeforePersistence(t *testing.T) {
	ctx := context.Background()
	sender, peer, dialogID := uuid.New(), uuid.New(), uuid.New()
	persistenceFailure := errors.New("transaction rolled back")
	repo := &sendRepository{err: persistenceFailure}
	publisher := &recordedEvent{}
	service := NewService(repo, dialogAccessStub{participants: [2]uuid.UUID{sender, peer}}, publisher)

	for _, body := range []string{" \t\n", "\x00", "\xff", strings.Repeat("Ж", MaxBodyLength+1)} {
		_, err := service.SendMessage(ctx, sender, dialogID, body, nil)
		if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
			t.Fatalf("invalid message error = %v", err)
		}
	}
	if _, err := service.SendMessage(ctx, uuid.New(), dialogID, "hello", nil); apperror.CodeOf(err) != apperror.CodeForbidden {
		t.Fatalf("outside participant error = %v", err)
	}
	if repo.calls != 0 {
		t.Fatalf("rejected messages reached persistence %d times", repo.calls)
	}
	if _, err := service.SendMessage(ctx, sender, dialogID, "hello", nil); !errors.Is(err, persistenceFailure) {
		t.Fatalf("persistence error = %v", err)
	}
	if len(publisher.values) != 0 {
		t.Fatal("an unpublished message reached subscribers")
	}

	repo.err = nil
	body := strings.Repeat("Ж", MaxBodyLength)
	message, err := service.SendMessage(ctx, sender, dialogID, body, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(publisher.values) != 1 || len(publisher.recipients) != 2 || publisher.recipients[0] != sender || publisher.recipients[1] != peer {
		t.Fatalf("wrong event delivery: %d event(s) for %v", len(publisher.values), publisher.recipients)
	}
	payload, ok := publisher.values[0].Payload.(event.MessageCreated)
	if !ok || payload.MessageID != message.ID || payload.DialogID != dialogID || payload.Body != body {
		t.Fatal("event does not describe the persisted message")
	}
}

type historyRepository struct {
	Repository
	messages []Message
	cursor   *Cursor
	limit    int
}

func (r *historyRepository) ListByDialog(_ context.Context, _ uuid.UUID, cursor *Cursor, limit int) ([]Message, error) {
	r.cursor, r.limit = cursor, limit
	return r.messages, nil
}

func TestListMessagesUsesLastReturnedMessageAsCursor(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	sender, peer, dialogID := uuid.New(), uuid.New(), uuid.New()
	latest, next := Message{ID: uuid.New(), CreatedAt: now}, Message{ID: uuid.New(), CreatedAt: now}
	repo := &historyRepository{messages: []Message{latest, next}}
	service := NewService(repo, dialogAccessStub{participants: [2]uuid.UUID{sender, peer}}, &recordedEvent{})

	messages, token, err := service.ListMessages(ctx, sender, dialogID, 1, "")
	if err != nil || len(messages) != 1 || messages[0].ID != latest.ID || token == "" || repo.limit != 2 {
		t.Fatalf("first page = %v, %q, %v (limit %d)", messages, token, err, repo.limit)
	}
	repo.messages = []Message{next}
	messages, finalToken, err := service.ListMessages(ctx, sender, dialogID, 1, token)
	if err != nil || len(messages) != 1 || messages[0].ID != next.ID || finalToken != "" {
		t.Fatalf("last page = %v, %q, %v", messages, finalToken, err)
	}
	if repo.cursor == nil || repo.cursor.ID != latest.ID || !repo.cursor.CreatedAt.Equal(now) {
		t.Fatalf("cursor did not point at last returned message: %+v", repo.cursor)
	}
}
