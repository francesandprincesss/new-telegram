package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTManagerIssueAndVerify(t *testing.T) {
	manager := NewJWTManager([]byte("01234567890123456789012345678901"), "test", time.Hour)
	manager.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	userID := uuid.New()
	token, expiresAt, err := manager.Issue(userID)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if !expiresAt.Equal(manager.now().Add(time.Hour)) {
		t.Fatalf("expiresAt = %v", expiresAt)
	}
	verified, err := manager.Verify(token)
	if err != nil || verified != userID {
		t.Fatalf("Verify() = %v, %v", verified, err)
	}

	manager.now = func() time.Time { return time.Unix(1_700_000_000, 0).Add(2 * time.Hour) }
	if _, err := manager.Verify(token); err == nil {
		t.Fatal("Verify() accepted an expired token")
	}
}
