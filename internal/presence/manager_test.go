package presence

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/event"
	"github.com/google/uuid"
)

type memoryRepository struct {
	mu      sync.Mutex
	status  Status
	seenAt  *time.Time
	seenErr error
}

func (r *memoryRepository) Get(_ context.Context, userID uuid.UUID) (State, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return State{UserID: userID, Status: r.status, LastSeenAt: r.seenAt}, nil
}
func (r *memoryRepository) SetManualStatus(_ context.Context, _ uuid.UUID, status Status) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = status
	return nil
}
func (r *memoryRepository) SetLastSeen(_ context.Context, _ uuid.UUID, seenAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seenAt = &seenAt
	return r.seenErr
}

func TestManagerPublishesOfflineWhenLastSeenPersistenceFails(t *testing.T) {
	repository := &memoryRepository{status: Online, seenErr: errors.New("database unavailable")}
	publisher := &recordingPublisher{}
	manager := NewManager(repository, noPeers{}, publisher)
	userID := uuid.New()
	if err := manager.Connected(context.Background(), userID); err != nil {
		t.Fatal(err)
	}
	if err := manager.Disconnected(context.Background(), userID); err == nil {
		t.Fatal("Disconnected() expected a persistence error")
	}
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if len(publisher.events) != 2 {
		t.Fatalf("published %d events, want online and offline", len(publisher.events))
	}
	payload, ok := publisher.events[1].Payload.(event.PresenceChanged)
	if !ok || payload.Status != string(Offline) {
		t.Fatalf("last event = %#v, want OFFLINE", publisher.events[1].Payload)
	}
}

type noPeers struct{}

func (noPeers) PeerIDs(context.Context, uuid.UUID) ([]uuid.UUID, error) { return nil, nil }

type recordingPublisher struct {
	mu     sync.Mutex
	events []event.Event
}

func (p *recordingPublisher) PublishTo(_ []uuid.UUID, value event.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, value)
}

func TestManagerTracksMultipleConnections(t *testing.T) {
	repository := &memoryRepository{status: DND}
	publisher := &recordingPublisher{}
	manager := NewManager(repository, noPeers{}, publisher)
	userID := uuid.New()
	ctx := context.Background()
	if err := manager.Connected(ctx, userID); err != nil {
		t.Fatal(err)
	}
	if err := manager.Connected(ctx, userID); err != nil {
		t.Fatal(err)
	}
	if err := manager.Disconnected(ctx, userID); err != nil {
		t.Fatal(err)
	}
	status, _, err := manager.Resolve(ctx, userID)
	if err != nil || status != string(DND) {
		t.Fatalf("Resolve() = %q, %v", status, err)
	}
	if err := manager.Disconnected(ctx, userID); err != nil {
		t.Fatal(err)
	}
	status, seenAt, err := manager.Resolve(ctx, userID)
	if err != nil || status != string(Offline) || seenAt == nil {
		t.Fatalf("Resolve() = %q, %v, %v", status, seenAt, err)
	}
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if len(publisher.events) != 2 {
		t.Fatalf("published %d events, want online and offline transitions", len(publisher.events))
	}
}

func TestManagerDoesNotExposeManualStatusWhileOffline(t *testing.T) {
	repository := &memoryRepository{status: Online}
	manager := NewManager(repository, noPeers{}, &recordingPublisher{})
	state, err := manager.SetStatus(context.Background(), uuid.New(), Idle)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != Offline {
		t.Fatalf("SetStatus() effective status = %s, want OFFLINE", state.Status)
	}
}
