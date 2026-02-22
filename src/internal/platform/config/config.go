// Copyright (c) 2026 Yomira. All rights reserved.
// Author: tai.buivan.jp@gmail.com

// Package config defines the application configuration struct and environment
// variable parsing.
//
// # Architecture
//
// Config is loaded exactly once at startup and then passed to every component
// via dependency injection — no package-level state is maintained.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds all runtime configuration for the Yomira API server.
//
// # Population
//
// Values are populated from environment variables at startup via [Load].
// Every field with `env:"...,required"` will cause [Load] to fail fast if absent.
type Config struct {
	// --- Server ---
	ServerPort  string `env:"SERVER_PORT"  envDefault:"8080"`
	Environment string `env:"ENVIRONMENT"  envDefault:"development"`
	Debug       bool   `env:"DEBUG"        envDefault:"false"`

	// --- Database ---
	DatabaseURL string `env:"DATABASE_URL,required"`
	// MigrationPath is the filesystem path to the SQL migrations directory.
	// Relative to the working directory of the process (i.e. src/).
	MigrationPath string `env:"MIGRATION_PATH" envDefault:"./data/migrations"`

	// --- Redis ---
	RedisURL string `env:"REDIS_URL,required"`

	// --- Authentication ---
	SessionSecret  string `env:"SESSION_SECRET,required"`
	JWTPrivKeyPath string `env:"JWT_PRIVATE_KEY_PATH,required"`
	JWTPubKeyPath  string `env:"JWT_PUBLIC_KEY_PATH,required"`

	// --- Storage (Cloudflare R2 / S3-compatible) ---
	S3Bucket   string `env:"S3_BUCKET"`
	S3Region   string `env:"S3_REGION"   envDefault:"auto"`
	S3Endpoint string `env:"S3_ENDPOINT"`

	// --- CORS ---
	// ExtraOrigins is a comma-separated list of allowed origins beyond defaults.
	ExtraOrigins string `env:"EXTRA_ORIGINS"`
}

// Load parses environment variables into a [Config] struct.
//
// # Lifecycle
//
// Returns an error if any required variable is missing.
// This function must be called once at program startup (inside main.go).
func Load() (*Config, error) {
	cfg := &Config{}
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
