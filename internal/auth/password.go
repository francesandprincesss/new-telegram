package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordCharacters = 8
	maxPasswordCharacters = 72
	argonMemory           = 19 * 1024 // KiB
	argonIterations       = 2
	argonThreads          = 1
	argonSaltSize         = 16
	argonKeySize          = 32
	argonPrefix           = "$argon2id$v=19$m=19456,t=2,p=1$"
)

type Argon2idHasher struct{}

func validatePassword(password string) error {
	if !utf8.ValidString(password) {
		return errors.New("password must be valid UTF-8")
	}
	count := utf8.RuneCountInString(password)
	if count < minPasswordCharacters || count > maxPasswordCharacters {
		return errors.New("password must contain 8 to 72 characters")
	}
	return nil
}

func (Argon2idHasher) Hash(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}
	var salt [argonSaltSize]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return "", err
	}
	digest := passwordDigest(password, salt[:])
	return argonPrefix + base64.RawStdEncoding.EncodeToString(salt[:]) + "$" + base64.RawStdEncoding.EncodeToString(digest), nil
}

func (Argon2idHasher) Compare(hash, password string) error {
	fields, argon := strings.CutPrefix(hash, argonPrefix)
	if !argon {
		// Accounts created before Argon2id used bcrypt with a UTF-8 byte limit.
		if len(password) < 8 || len(password) > 72 {
			return bcrypt.ErrMismatchedHashAndPassword
		}
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	}
	saltField, digestField, ok := strings.Cut(fields, "$")
	if !ok || len(saltField) != base64.RawStdEncoding.EncodedLen(argonSaltSize) ||
		len(digestField) != base64.RawStdEncoding.EncodedLen(argonKeySize) {
		return errors.New("invalid password hash format")
	}
	salt, saltErr := base64.RawStdEncoding.Strict().DecodeString(saltField)
	want, digestErr := base64.RawStdEncoding.Strict().DecodeString(digestField)
	if saltErr != nil || digestErr != nil || len(salt) != argonSaltSize || len(want) != argonKeySize {
		return errors.New("invalid password hash encoding")
	}
	if subtle.ConstantTimeCompare(passwordDigest(password, salt), want) != 1 {
		return errors.New("password does not match")
	}
	return nil
}

func passwordDigest(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonThreads, argonKeySize)
}
