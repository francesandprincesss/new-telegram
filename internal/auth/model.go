package auth

import (
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID                 uuid.UUID
	Username           string
	UsernameNormalized string
	PasswordHash       string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type NewAccount struct {
	ID                 uuid.UUID
	Username           string
	UsernameNormalized string
	PasswordHash       string
	CreatedAt          time.Time
}

type Credentials struct {
	UserID       uuid.UUID
	Username     string
	PasswordHash string
}
