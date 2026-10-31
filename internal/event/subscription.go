package event

import "github.com/google/uuid"

type Subscription struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Events <-chan Event
}

// TODO: Add an idempotent cancellation function owned by Hub. Never close a
// channel concurrently with a publisher writing to it.
