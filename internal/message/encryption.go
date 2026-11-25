package message

import (
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/chacha20poly1305"
)

const bodyCipherVersion byte = 1

type bodyCipher struct {
	aead     cipher.AEAD
	verifier []byte
}

func newBodyCipher(key []byte) (bodyCipher, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return bodyCipher{}, err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("new-telegram/message-encryption-key/v1"))
	return bodyCipher{aead: aead, verifier: mac.Sum(nil)}, nil
}

func bodyAAD(id, dialogID, senderID uuid.UUID) [49]byte {
	aad := [49]byte{bodyCipherVersion}
	copy(aad[1:17], id[:])
	copy(aad[17:33], dialogID[:])
	copy(aad[33:], senderID[:])
	return aad
}

func (c bodyCipher) encrypt(body string, id, dialogID, senderID uuid.UUID) ([]byte, error) {
	header := make([]byte, 1+chacha20poly1305.NonceSizeX)
	header[0] = bodyCipherVersion
	nonce := header[1:]
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	aad := bodyAAD(id, dialogID, senderID)
	return c.aead.Seal(header, nonce, []byte(body), aad[:]), nil
}

func (c bodyCipher) decrypt(encrypted []byte, id, dialogID, senderID uuid.UUID) (string, error) {
	const headerSize = 1 + chacha20poly1305.NonceSizeX
	if len(encrypted) < headerSize+chacha20poly1305.Overhead || encrypted[0] != bodyCipherVersion {
		return "", errors.New("invalid message ciphertext format")
	}
	aad := bodyAAD(id, dialogID, senderID)
	body, err := c.aead.Open(nil, encrypted[1:headerSize], encrypted[headerSize:], aad[:])
	return string(body), err
}
