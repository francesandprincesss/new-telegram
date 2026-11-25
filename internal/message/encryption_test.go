package message

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestMessageCipherRoundTripAndIntegrity(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	c, err := newBodyCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	other, err := newBodyCipher(bytes.Repeat([]byte{2}, 32))
	if err != nil {
		t.Fatal(err)
	}
	id, dialog, sender := uuid.New(), uuid.New(), uuid.New()
	body := strings.Repeat("🔒", MaxBodyLength)
	sealed, err := c.encrypt(body, id, dialog, sender)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.encrypt(body, id, dialog, sender)
	if err != nil || bytes.Equal(sealed, second) {
		t.Fatalf("expected distinct ciphertexts, error = %v", err)
	}
	if got, err := c.decrypt(sealed, id, dialog, sender); err != nil || got != body {
		t.Fatalf("message round trip failed: %v", err)
	}
	tampered := bytes.Clone(sealed)
	tampered[len(tampered)-1] ^= 1
	for _, tc := range []struct {
		name               string
		key                bodyCipher
		data               []byte
		id, dialog, sender uuid.UUID
	}{
		{"wrong key", other, sealed, id, dialog, sender},
		{"wrong message", c, sealed, uuid.New(), dialog, sender},
		{"wrong dialog", c, sealed, id, uuid.New(), sender},
		{"wrong sender", c, sealed, id, dialog, uuid.New()},
		{"tampered", c, tampered, id, dialog, sender},
		{"truncated", c, sealed[:10], id, dialog, sender},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.key.decrypt(tc.data, tc.id, tc.dialog, tc.sender); err == nil {
				t.Fatal("decrypt accepted an invalid ciphertext")
			}
		})
	}
}
