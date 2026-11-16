package event

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHubDeliversAndCancelsSubscription(t *testing.T) {
	hub := NewHub(2)
	userID := uuid.New()
	sub := hub.Subscribe(userID)
	want := Event{ID: uuid.New(), OccurredAt: time.Now()}
	hub.PublishTo([]uuid.UUID{userID, userID}, want)
	select {
	case got := <-sub.Events:
		if got.ID != want.ID {
			t.Fatalf("event ID = %s, want %s", got.ID, want.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("event was not delivered")
	}
	sub.Cancel()
	sub.Cancel()
	select {
	case <-sub.Done:
	case <-time.After(time.Second):
		t.Fatal("subscription was not cancelled")
	}
}

func TestHubDisconnectsSlowConsumer(t *testing.T) {
	hub := NewHub(1)
	userID := uuid.New()
	sub := hub.Subscribe(userID)
	hub.PublishTo([]uuid.UUID{userID}, Event{ID: uuid.New()})
	hub.PublishTo([]uuid.UUID{userID}, Event{ID: uuid.New()})
	select {
	case <-sub.Done:
	case <-time.After(time.Second):
		t.Fatal("slow consumer was not disconnected")
	}
}
