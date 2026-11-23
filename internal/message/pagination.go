package message

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

func decodePageToken(token string) (*Cursor, error) {
	var cursor Cursor

	if token == "" {
		return nil, nil
	}

	data, err := base64.RawURLEncoding.DecodeString(token); if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, err
	}

	if cursor.ID == uuid.Nil || cursor.CreatedAt.IsZero() {
		return nil, fmt.Errorf("invalid page token")
	}

	return &cursor, nil
}

func encodePageToken(cursor *Cursor) (string, error) {
	data, err := json.Marshal(cursor); if err != nil {
		return "", err
	}

	token := base64.RawStdEncoding.EncodeToString(data)

	return token, nil
}