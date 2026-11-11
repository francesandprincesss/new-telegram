package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const minPasswordBytes = 8

type BcryptHasher struct{ Cost int }

func NewBcryptHasher(cost int) *BcryptHasher {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{Cost: cost}
}

func validatePassword(password string) error {
	if len(password) < minPasswordBytes {
		return fmt.Errorf("password must contain at least %d bytes", minPasswordBytes)
	}
	if len(password) > 72 {
		return fmt.Errorf("password must contain at most 72 bytes")
	}
	return nil
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.Cost)
	return string(hash), err
}

func (h *BcryptHasher) Compare(hash, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
