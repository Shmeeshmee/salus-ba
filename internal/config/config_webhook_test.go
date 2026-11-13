package config_test

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

// synthetic32 is base64 of 32 synthetic bytes.
var synthetic32 = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))

func TestLoadWebhookKey_ReadsTheEnvironmentsKey(t *testing.T) {
	for _, env := range []string{"", "dev", "production"} {
		t.Setenv("SALUS_ENV", env)
		t.Setenv("WEBHOOK_KEY", synthetic32)
		key, err := config.LoadWebhookKey()
		if err != nil || !bytes.Equal(key, bytes.Repeat([]byte{7}, 32)) {
			t.Errorf("SALUS_ENV=%q: %v, %v; want the key set", env, key, err)
		}
	}
	t.Setenv("WEBHOOK_KEY", "")
	path := filepath.Join(t.TempDir(), "webhook_key")
	if err := os.WriteFile(path, []byte(synthetic32+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEBHOOK_KEY_FILE", path)
	if key, err := config.LoadWebhookKey(); err != nil || len(key) != 32 {
		t.Errorf("WEBHOOK_KEY_FILE: %d bytes, %v; want the file's key", len(key), err)
	}
}

// Outside dev a missing key stops the start-up: every environment sets its own (production: the key file).
func TestLoadWebhookKey_IsRequiredOutsideDev(t *testing.T) {
	t.Setenv("WEBHOOK_KEY", "")
	t.Setenv("WEBHOOK_KEY_FILE", "")
	for _, env := range []string{"", "production", "staging", "Dev"} {
		t.Setenv("SALUS_ENV", env)
		if _, err := config.LoadWebhookKey(); err == nil || !strings.Contains(err.Error(), "WEBHOOK_KEY") {
			t.Errorf("SALUS_ENV=%q without a key: %v, want WEBHOOK_KEY required", env, err)
		}
	}
}

// In dev a missing key is generated: 32 random bytes, a new one each time.
func TestLoadWebhookKey_GeneratesOneInDev(t *testing.T) {
	t.Setenv("SALUS_ENV", "dev")
	t.Setenv("WEBHOOK_KEY", "")
	t.Setenv("WEBHOOK_KEY_FILE", "")
	a, err := config.LoadWebhookKey()
	if err != nil || len(a) != 32 {
		t.Fatalf("LoadWebhookKey() = %d bytes, %v; want 32 generated", len(a), err)
	}
	if b, _ := config.LoadWebhookKey(); bytes.Equal(a, b) {
		t.Error("two generated keys are equal")
	}
}

func TestLoadWebhookKey_RefusesAKeyThatIsNot32BytesWithoutTheValue(t *testing.T) {
	t.Setenv("SALUS_ENV", "dev")
	for _, v := range []string{"not base64!", base64.StdEncoding.EncodeToString([]byte("short"))} {
		t.Setenv("WEBHOOK_KEY", v)
		if _, err := config.LoadWebhookKey(); err == nil || strings.Contains(err.Error(), v) {
			t.Errorf("WEBHOOK_KEY=%q: %v, want refused without the value", v, err)
		}
	}
}
