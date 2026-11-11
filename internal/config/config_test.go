package config

import "testing"

func validEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("GRPC_ADDRESS", ":50051")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_ISSUER", "test")
	t.Setenv("JWT_TTL", "1h")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
}

func TestLoadValidatesRequiredConfiguration(t *testing.T) {
	validEnvironment(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GRPCAddress != ":50051" || cfg.JWT.TTL.String() != "1h0m0s" {
		t.Fatalf("unexpected config: %+v", cfg)
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
}
