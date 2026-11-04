// Package config is the only package that reads the environment.
package config

import (
	"encoding/base64"
	"errors"
	"net/netip"
	"os"
	"strings"
)

// Config holds the settings the binaries need.
type Config struct {
	DatabaseURL string
	HTTPAddr    string
	RedisURL    string // only the binaries that use Redis require it (redis.Open rejects an empty URL)
	// AppDatabasePassword is set on the salus_app role by `migrate up` when not empty. Production leaves it
	// unset when the role is provisioned outside migrate.
	AppDatabasePassword string
	// SMTPAddr is the SMTP server the notifier sends through (SMTP_ADDR, host:port; Mailpit locally).
	SMTPAddr string
	// WebBaseURL is the web app's origin, for links in e-mails (WEB_BASE_URL).
	WebBaseURL string
	// FHIRBaseURL is the FHIR API's absolute public base, https://…/fhir/R4, for the links and fullUrls of
	// a search's Bundle (FHIR_BASE_URL; the api requires it).
	FHIRBaseURL string
	// TOTPKey seals TOTP secrets (TOTP_KEY, base64, 32 bytes). Local and CI only: production takes it from
	// the KMS, which is not decided yet (docs/accounts/SPEC.md "TOTP").
	TOTPKey []byte
	// RateLimitKey keys the HMAC that names the login rate-limit counters (RATE_LIMIT_KEY, base64).
	RateLimitKey []byte
	// TrustedProxies may set X-Forwarded-For (TRUSTED_PROXY_CIDRS, comma-separated CIDRs).
	TrustedProxies []netip.Prefix
}

// Load reads the configuration from the environment. Errors name the variable, never a value.
func Load() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		HTTPAddr:    os.Getenv("HTTP_ADDR"),
		RedisURL:    os.Getenv("REDIS_URL"),

		AppDatabasePassword: os.Getenv("APP_DATABASE_PASSWORD"),
		SMTPAddr:            os.Getenv("SMTP_ADDR"),
		WebBaseURL:          os.Getenv("WEB_BASE_URL"),
		FHIRBaseURL:         os.Getenv("FHIR_BASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("config: DATABASE_URL is required")
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}
	var err error
	if cfg.TOTPKey, err = base64.StdEncoding.DecodeString(os.Getenv("TOTP_KEY")); err != nil {
		return Config{}, errors.New("config: TOTP_KEY is not base64")
	}
	if cfg.RateLimitKey, err = base64.StdEncoding.DecodeString(os.Getenv("RATE_LIMIT_KEY")); err != nil {
		return Config{}, errors.New("config: RATE_LIMIT_KEY is not base64")
	}
	for _, s := range strings.Split(os.Getenv("TRUSTED_PROXY_CIDRS"), ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return Config{}, errors.New("config: TRUSTED_PROXY_CIDRS holds an entry that is not a CIDR")
		}
		cfg.TrustedProxies = append(cfg.TrustedProxies, p)
	}
	return cfg, nil
}
