// Package config is the only package that reads the environment.
package config

import (
	"errors"
	"os"
)

// Config holds the settings the binaries need.
type Config struct {
	DatabaseURL string
	HTTPAddr    string
}

// Load reads the configuration from the environment. Errors name the missing variable, never a value.
func Load() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		HTTPAddr:    os.Getenv("HTTP_ADDR"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("config: DATABASE_URL is required")
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	return cfg, nil
}
