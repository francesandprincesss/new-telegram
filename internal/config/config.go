// Package config owns environment-based application configuration.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config contains process-level configuration. Secrets must be supplied via
// environment variables and must never be committed to the repository.
type Config struct {
	GRPCAddress string
	DatabaseURL string
	JWT         JWTConfig
}

type JWTConfig struct {
	Secret []byte
	Issuer string
	TTL    time.Duration
}

func Load() (Config, error) {
	ttl, err := time.ParseDuration(os.Getenv("JWT_TTL"))
	if err != nil || ttl <= 0 {
		return Config{}, fmt.Errorf("JWT_TTL must be a positive duration")
	}
	cfg := Config{
		GRPCAddress: strings.TrimSpace(os.Getenv("GRPC_ADDRESS")),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWT:         JWTConfig{Secret: []byte(os.Getenv("JWT_SECRET")), Issuer: strings.TrimSpace(os.Getenv("JWT_ISSUER")), TTL: ttl},
	}
	if cfg.GRPCAddress == "" {
		return Config{}, fmt.Errorf("GRPC_ADDRESS is required")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWT.Issuer == "" {
		return Config{}, fmt.Errorf("JWT_ISSUER is required")
	}
	if len(cfg.JWT.Secret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	return cfg, nil
}
