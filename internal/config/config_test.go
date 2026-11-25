package config

import (
	"encoding/base64"
	"testing"
)

func validEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("GRPC_ADDRESS", ":50051")
	t.Setenv("HTTP_ADDRESS", ":8080")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_ISSUER", "test")
	t.Setenv("JWT_TTL", "1h")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("MESSAGE_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(make([]byte, 32)))
}

func TestLoadValidatesRequiredConfiguration(t *testing.T) {
	validEnvironment(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GRPCAddress != ":50051" || cfg.JWT.TTL.String() != "1h0m0s" || len(cfg.MessageEncryptionKey) != 32 {
		t.Fatal("unexpected configuration")
	}

	t.Run("short secret", func(t *testing.T) {
		validEnvironment(t)
		t.Setenv("JWT_SECRET", "short")
		if _, err := Load(); err == nil {
			t.Fatal("Load() expected an error")
		}
	})

	t.Run("invalid ttl", func(t *testing.T) {
		validEnvironment(t)
		t.Setenv("JWT_TTL", "0s")
		if _, err := Load(); err == nil {
			t.Fatal("Load() expected an error")
		}
	})

	t.Run("missing http address", func(t *testing.T) {
		validEnvironment(t)
		t.Setenv("HTTP_ADDRESS", "")
		if _, err := Load(); err == nil {
			t.Fatal("Load() expected an error")
		}
	})
}

func TestLoadRejectsInvalidMessageEncryptionKeys(t *testing.T) {
	for _, key := range []string{"", "not base64", base64.StdEncoding.EncodeToString(make([]byte, 31))} {
		validEnvironment(t)
		t.Setenv("MESSAGE_ENCRYPTION_KEY", key)
		if _, err := Load(); err == nil {
			t.Fatal("Load() accepted an invalid message encryption key")
		}
	}
}
