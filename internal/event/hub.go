package event

import (
	"sync"

	"github.com/google/uuid"
)

const defaultQueueSize = 64

type Hub struct {
	mu            sync.RWMutex
	subscriptions map[uuid.UUID]map[uuid.UUID]*subscriber
	queueSize     int
	closed        bool
}

type subscriber struct {
	mu     sync.Mutex
	events chan Event
	done   chan struct{}
	closed bool
}

func NewHub(queueSize int) *Hub {
	if queueSize <= 0 {
		queueSize = defaultQueueSize
	}
	return &Hub{subscriptions: make(map[uuid.UUID]map[uuid.UUID]*subscriber), queueSize: queueSize}
}

func (h *Hub) Subscribe(userID uuid.UUID) *Subscription {
	id := uuid.New()
	sub := &subscriber{events: make(chan Event, h.queueSize), done: make(chan struct{})}
	h.mu.Lock()
	if h.closed {
		sub.close()
	} else {
		if h.subscriptions[userID] == nil {
			h.subscriptions[userID] = make(map[uuid.UUID]*subscriber)
		}
		h.subscriptions[userID][id] = sub
	}
	h.mu.Unlock()
	return &Subscription{ID: id, UserID: userID, Events: sub.events, Done: sub.done, cancel: func() { h.unsubscribe(userID, id, sub) }}
}

func (h *Hub) PublishTo(userIDs []uuid.UUID, event Event) {
	seen := make(map[uuid.UUID]struct{}, len(userIDs))
	var targets []*subscriber
	h.mu.RLock()
	for _, userID := range userIDs {
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		for _, sub := range h.subscriptions[userID] {
			targets = append(targets, sub)
		}
	}
	h.mu.RUnlock()
	for _, sub := range targets {
		if !sub.send(event) {
			h.removeSubscriber(sub)
		}
	}
}

func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	var targets []*subscriber
	for _, byID := range h.subscriptions {
		for _, sub := range byID {
			targets = append(targets, sub)
		}
	}
	h.subscriptions = make(map[uuid.UUID]map[uuid.UUID]*subscriber)
	h.mu.Unlock()
	for _, sub := range targets {
		sub.close()
	}
}

func (h *Hub) unsubscribe(userID, id uuid.UUID, sub *subscriber) {
	h.mu.Lock()
	if current := h.subscriptions[userID][id]; current == sub {
		delete(h.subscriptions[userID], id)
		if len(h.subscriptions[userID]) == 0 {
			delete(h.subscriptions, userID)
		}
	}
	h.mu.Unlock()
	sub.close()
}

func (h *Hub) removeSubscriber(target *subscriber) {
	h.mu.Lock()
	for userID, byID := range h.subscriptions {
		for id, sub := range byID {
			if sub == target {
				delete(byID, id)
			}
		}
		if len(byID) == 0 {
			delete(h.subscriptions, userID)
		}
	}
	h.mu.Unlock()
	target.close()
}

func (s *subscriber) send(event Event) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return true
	}
	select {
	case s.events <- event:
		return true
	default:
		return false
	}
}

func (s *subscriber) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.done)
	}
}
