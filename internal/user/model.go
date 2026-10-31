package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Username     string
	ManualStatus string
	LastSeenAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
