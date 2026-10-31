package reaction

import (
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	Like  Type = "LIKE"
	Love  Type = "LOVE"
	Laugh Type = "LAUGH"
	Wow   Type = "WOW"
)

type Reaction struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
	Type      Type
	CreatedAt time.Time
	UpdatedAt time.Time
}
