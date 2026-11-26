package chat

import (
	"time"

	"github.com/google/uuid"
)

type Dialog struct {
	ID         uuid.UUID
	UserLowID  uuid.UUID
	UserHighID uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Cursor struct {
	UpdatedAt time.Time
	ID        uuid.UUID
}
