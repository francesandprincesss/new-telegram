package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	CreateAccount(ctx context.Context, account NewAccount) (Account, error)
	FindCredentialsByNormalizedUsername(ctx context.Context, username string) (Credentials, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenIssuer interface {
	Issue(userID uuid.UUID) (token string, expiresAt time.Time, err error)
}
