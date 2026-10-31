package event

import "github.com/google/uuid"

// Publisher is the stable event-delivery contract used by domain modules.
type Publisher interface {
	PublishTo(userIDs []uuid.UUID, event Event)
}
