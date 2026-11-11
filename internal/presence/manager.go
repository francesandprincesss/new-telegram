package presence

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/google/uuid"
)

type Manager struct {
	mu          sync.Mutex
	connections map[uuid.UUID]int
	repository  Repository
	peers       PeerLookup
	events      Publisher
	now         func() time.Time
}

func NewManager(repository Repository, peers PeerLookup, events Publisher) *Manager {
	return &Manager{connections: make(map[uuid.UUID]int), repository: repository, peers: peers, events: events, now: time.Now}
}

func (m *Manager) Connected(ctx context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.connections[userID] > 0 {
		m.connections[userID]++
		return nil
	}
	state, err := m.repository.Get(ctx, userID)
	if err != nil {
		return err
	}
	m.connections[userID] = 1
	m.publish(ctx, State{UserID: userID, Status: state.Status})
	return nil
}

func (m *Manager) Disconnected(ctx context.Context, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := m.connections[userID]
	if count > 1 {
		m.connections[userID] = count - 1
		return nil
	}
	if count == 0 {
		return nil
	}
	delete(m.connections, userID)
	seenAt := m.now().UTC()
	persistErr := m.repository.SetLastSeen(ctx, userID, seenAt)
	m.publish(ctx, State{UserID: userID, Status: Offline, LastSeenAt: &seenAt})
	return persistErr
}

func (m *Manager) SetStatus(ctx context.Context, userID uuid.UUID, status Status) (State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.repository.SetManualStatus(ctx, userID, status); err != nil {
		return State{}, err
	}
	connected := m.connections[userID] > 0
	state, err := m.repository.Get(ctx, userID)
	if err != nil {
		return State{}, err
	}
	if connected {
		state.Status = status
		m.publish(ctx, state)
	} else {
		state.Status = Offline
	}
	return state, nil
}

func (m *Manager) Resolve(ctx context.Context, userID uuid.UUID) (string, *time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, err := m.repository.Get(ctx, userID)
	if err != nil {
		return "", nil, err
	}
	connected := m.connections[userID] > 0
	if !connected {
		state.Status = Offline
	}
	return string(state.Status), state.LastSeenAt, nil
}

func (m *Manager) publish(ctx context.Context, state State) {
	recipients, err := m.peers.PeerIDs(ctx, state.UserID)
	if err != nil {
		log.Printf("load presence recipients for %s: %v", state.UserID, err)
		recipients = nil
	}
	recipients = append(recipients, state.UserID)
	m.events.PublishTo(recipients, event.Event{ID: uuid.New(), OccurredAt: m.now().UTC(), Payload: event.PresenceChanged{UserID: state.UserID, Status: string(state.Status), LastSeenAt: state.LastSeenAt}})
}
