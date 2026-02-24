// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

/*
Package config handles application-wide settings and environment parsing.

It leverages 'caarlos0/env' to map OS environment variables into a strongly-typed
Go struct, providing early validation and default values.

Usage:

	cfg, err := config.Load()
	if err != nil {
	    log.Fatal(err)
	}

Architecture:

  - Immutability: Once loaded, configuration is read-only.
  - DI-Friendly: Passed to core components (DB, Redis) via constructors.
  - Zero Hidden State: No global variables are used to store config.

This ensures the application is Twelve-Factor compliant by storing config in the env.
*/
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// # Configuration Schema

// Config holds all runtime configuration for the Yomira API server.
type Config struct {

	// Server settings
	ServerPort  string `env:"SERVER_PORT"  envDefault:"8080"`
	Environment string `env:"ENVIRONMENT"  envDefault:"development"`
	Debug       bool   `env:"DEBUG"        envDefault:"false"`

	// Relational Database (PostgreSQL)
	DatabaseURL string `env:"DATABASE_URL,required"`

	// MigrationPath is the filesystem path to the SQL migrations directory.
	MigrationPath string `env:"MIGRATION_PATH" envDefault:"./data/migrations"`

	// Key-Value Cache (Redis)
	RedisURL string `env:"REDIS_URL,required"`

	// Cryptographic keys for session and identity signing
	SessionSecret  string `env:"SESSION_SECRET,required"`
	JWTPrivKeyPath string `env:"JWT_PRIVATE_KEY_PATH,required"`
	JWTPubKeyPath  string `env:"JWT_PUBLIC_KEY_PATH,required"`

	// Object Storage (Cloudflare R2 / S3-compatible)
	S3Bucket   string `env:"S3_BUCKET"`
	S3Region   string `env:"S3_REGION"   envDefault:"auto"`
	S3Endpoint string `env:"S3_ENDPOINT"`

	// Cross-Origin Resource Sharing
	ExtraOrigins string `env:"EXTRA_ORIGINS"`
}

// # Configuration Loading

// Load parses environment variables into a [Config] struct.
func Load() (*Config, error) {

	// Initialize an empty config struct
	cfg := &Config{}

	// Use the 'env' package to map environment variables to struct fields.
	// This will fail if any field marked with 'required' is missing.
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: failed to parse environment variables: %w", err)
	}

	return cfg, nil
}

// IsDevelopment reports whether the server is running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction reports whether the server is running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
