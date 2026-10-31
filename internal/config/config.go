// Package config owns environment-based application configuration.
package config

import "time"

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

// TODO: Implement strict environment loading and validation. Reject a missing
// DATABASE_URL, JWT issuer, non-positive TTL, and JWT secrets shorter than 32 bytes.
