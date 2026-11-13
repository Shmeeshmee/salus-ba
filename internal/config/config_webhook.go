package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
)

// EnvDev is the SALUS_ENV of the compose stack (deployments/docker-compose.yml: a developer's machine, and CI's e2e
// stack built on it). Every other value, and none, is an environment that must set its own keys (production).
const EnvDev = "dev"

// webhookKeyLen is the AES-256 key's length.
const webhookKeyLen = 32

// LoadWebhookKey reads the key that seals the clinics' webhook secrets (SAL-29): WEBHOOK_KEY, base64 of 32 bytes, or
// the file WEBHOOK_KEY_FILE names, as production mounts it from the encrypted volume (keySetting). Each environment
// sets its own, so a missing key fails the api's and the worker's start-up, except with SALUS_ENV=dev, where one is
// generated: random, for this process only, so a secret it sealed opens neither in another process nor after a
// restart (the compose stack sets one key for the api and the worker). Errors name the variable, never a value.
func LoadWebhookKey() ([]byte, error) {
	value, err := keySetting("WEBHOOK_KEY")
	if err != nil {
		return nil, err
	}
	if value == "" {
		if os.Getenv("SALUS_ENV") != EnvDev {
			return nil, errors.New("config: WEBHOOK_KEY (or WEBHOOK_KEY_FILE) is required outside SALUS_ENV=dev")
		}
		key := make([]byte, webhookKeyLen)
		_, _ = rand.Read(key) // crypto/rand.Read never returns an error
		return key, nil
	}
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(key) != webhookKeyLen {
		return nil, errors.New("config: WEBHOOK_KEY is not base64 of 32 bytes")
	}
	return key, nil
}
