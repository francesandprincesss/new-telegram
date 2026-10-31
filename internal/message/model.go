package message

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID        uuid.UUID
	DialogID  uuid.UUID
	SenderID  uuid.UUID
	Body      string
	CreatedAt time.Time
}

type NewMessage struct {
	ID        uuid.UUID
	DialogID  uuid.UUID
	SenderID  uuid.UUID
	Body      string
	CreatedAt time.Time
}

type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}
