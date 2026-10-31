package presence

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	Online  Status = "ONLINE"
	Idle    Status = "IDLE"
	DND     Status = "DND"
	Offline Status = "OFFLINE"
)

type State struct {
	UserID     uuid.UUID
	Status     Status
	LastSeenAt *time.Time
}
