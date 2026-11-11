package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTManager struct {
	secret []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time
}

func NewJWTManager(secret []byte, issuer string, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: append([]byte(nil), secret...), issuer: issuer, ttl: ttl, now: time.Now}
}

func (m *JWTManager) Issue(userID uuid.UUID) (string, time.Time, error) {
	now := m.now().UTC()
	expiresAt := now.Add(m.ttl)
	claims := jwt.RegisteredClaims{
		Issuer: m.issuer, Subject: userID.String(),
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expiresAt),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	return token, expiresAt, err
}

func (m *JWTManager) Verify(raw string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithTimeFunc(m.now))
	if err != nil || !token.Valid {
		return uuid.Nil, fmt.Errorf("invalid token")
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil || userID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("invalid token subject")
	}
	return userID, nil
}
