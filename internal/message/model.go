package message

import (
	"time"

	"github.com/francesandprincesss/new-telegram/internal/reaction"
	"github.com/google/uuid"
)

type Message struct {
	ID        uuid.UUID
	DialogID  uuid.UUID
	SenderID  uuid.UUID
	Body      string
	CreatedAt time.Time
	ReadAt    *time.Time          `gorm:"->"`
	Reactions []reaction.Reaction `gorm:"-"`
	ReplyTo   *ReplyPreview       `gorm:"-"`
}

type ReplyPreview struct {
	ID       uuid.UUID
	SenderID uuid.UUID
	Excerpt  string
}

type NewMessage struct {
	ReplyToID *uuid.UUID
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
