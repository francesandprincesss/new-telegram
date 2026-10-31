package readreceipt

import (
	"time"

	"github.com/google/uuid"
)

type Receipt struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
	ReadAt    time.Time
}

type MarkResult struct {
	MarkedCount int64
	ReadAt      time.Time
}
