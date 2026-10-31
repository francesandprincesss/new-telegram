package event

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID         uuid.UUID
	OccurredAt time.Time
	Payload    Payload
}

type Payload interface {
	isEventPayload()
}

type MessageCreated struct {
	MessageID uuid.UUID
	DialogID  uuid.UUID
	SenderID  uuid.UUID
	Body      string
	CreatedAt time.Time
}

func (MessageCreated) isEventPayload() {}

type MessagesRead struct {
	DialogID      uuid.UUID
	ReaderID      uuid.UUID
	UpToMessageID uuid.UUID
	ReadAt        time.Time
}

func (MessagesRead) isEventPayload() {}

type ReactionChanged struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
	Reaction  string
	Removed   bool
	UpdatedAt time.Time
}

func (ReactionChanged) isEventPayload() {}

type PresenceChanged struct {
	UserID     uuid.UUID
	Status     string
	LastSeenAt *time.Time
}

func (PresenceChanged) isEventPayload() {}
