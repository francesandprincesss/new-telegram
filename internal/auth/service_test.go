package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/google/uuid"
)

type repositoryStub struct {
	created     NewAccount
	credentials Credentials
	findErr     error
}

func (r *repositoryStub) CreateAccount(_ context.Context, account NewAccount) (Account, error) {
	r.created = account
	return Account{ID: account.ID, Username: account.Username, CreatedAt: account.CreatedAt}, nil
}

func (r *repositoryStub) FindCredentialsByNormalizedUsername(_ context.Context, _ string) (Credentials, error) {
	return r.credentials, r.findErr
}

type passwordStub struct {
	compareErr error
	compares   int
}

func (passwordStub) Hash(string) (string, error) { return "dummy-hash", nil }
func (p *passwordStub) Compare(string, string) error {
	p.compares++
	return p.compareErr
}

type tokenStub struct{}

func (tokenStub) Issue(uuid.UUID) (string, time.Time, error) {
	return "token", time.Unix(2_000_000_000, 0), nil
}

func TestServiceRegisterNormalizesUsername(t *testing.T) {
	repository := &repositoryStub{}
	passwords := &passwordStub{}
	service := NewService(repository, passwords, tokenStub{})
	result, err := service.Register(context.Background(), "Alice_1", "password123")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if repository.created.UsernameNormalized != "alice_1" {
		t.Fatalf("normalized username = %q", repository.created.UsernameNormalized)
	}
	if result.Token != "token" || result.UserID == uuid.Nil {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestServiceLoginUsesUniformCredentialErrorAndDummyCompare(t *testing.T) {
	cases := []struct {
		name       string
		repository *repositoryStub
		compareErr error
	}{
		{name: "unknown user", repository: &repositoryStub{findErr: apperror.New(apperror.CodeNotFound, "user not found")}},
		{name: "wrong password", repository: &repositoryStub{credentials: Credentials{UserID: uuid.New(), PasswordHash: "hash"}}, compareErr: errors.New("mismatch")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			passwords := &passwordStub{compareErr: tc.compareErr}
			service := NewService(tc.repository, passwords, tokenStub{})
			_, err := service.Login(context.Background(), "Alice_1", "password123")
			if apperror.CodeOf(err) != apperror.CodeUnauthenticated || err.Error() != "unauthenticated: invalid username or password" {
				t.Fatalf("Login() error = %v", err)
			}
			if passwords.compares != 1 {
				t.Fatalf("password comparisons = %d, want 1", passwords.compares)
			}
		})
	}
}
