package session

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestUserIDRoundTrip(t *testing.T) {
	want := uuid.New()
	ctx := WithUserID(context.Background(), want)

	got, err := UserID(ctx)
	if err != nil {
		t.Fatalf("UserID() error = %v", err)
	}
	if got != want {
		t.Fatalf("UserID() = %s, want %s", got, want)
	}
}

func TestUserIDRejectsMissingIdentity(t *testing.T) {
	if _, err := UserID(context.Background()); err == nil {
		t.Fatal("UserID() error = nil, want an error")
	}
}

func TestWithUserIDRejectsNilContext(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("WithUserID(nil, ...) did not panic")
		}
	}()

	WithUserID(nil, uuid.New())
}
