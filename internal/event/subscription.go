package event

import (
	"sync"

	"github.com/google/uuid"
)

type Subscription struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Events <-chan Event
	Done   <-chan struct{}
	cancel func()
	once   sync.Once
}

func (s *Subscription) Cancel() {
	if s == nil {
		return
	}
	s.once.Do(s.cancel)
}
