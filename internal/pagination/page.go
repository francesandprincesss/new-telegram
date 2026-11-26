// Package pagination handles keyset cursors shared by dialog and message history.
package pagination

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/francesandprincesss/new-telegram/internal/apperror"
	"github.com/google/uuid"
)

const DefaultSize = 30
const MaxSize = 100

func Size(size int) (int, error) {
	switch {
	case size < 0 || size > MaxSize:
		return 0, apperror.New(apperror.CodeInvalidArgument, "page_size must be between 1 and 100")
	case size == 0:
		return DefaultSize, nil
	default:
		return size, nil
	}
}

func Encode(timestamp time.Time, id uuid.UUID) string {
	value := timestamp.UTC().Format(time.RFC3339Nano) + "/" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func Decode(token string) (time.Time, uuid.UUID, error) {
	invalid := apperror.New(apperror.CodeInvalidArgument, "invalid page_token")
	if len(token) > 128 {
		return time.Time{}, uuid.Nil, invalid
	}
	value, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, uuid.Nil, invalid
	}
	rawTime, rawID, ok := strings.Cut(string(value), "/")
	if !ok {
		return time.Time{}, uuid.Nil, invalid
	}
	timestamp, err := time.Parse(time.RFC3339Nano, rawTime)
	if err != nil || timestamp.IsZero() {
		return time.Time{}, uuid.Nil, invalid
	}
	id, err := uuid.Parse(rawID)
	if err != nil || id == uuid.Nil {
		return time.Time{}, uuid.Nil, invalid
	}
	return timestamp, id, nil
}
