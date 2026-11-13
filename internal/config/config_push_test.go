package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

// pushVars are every variable LoadPush reads.
var pushVars = []string{"FCM_CREDENTIALS_FILE", "APNS_KEY_FILE", "APNS_KEY_ID", "APNS_TEAM_ID", "APNS_BUNDLE_ID", "APNS_SANDBOX"}

func clearPush(t *testing.T) {
	t.Helper()
	for _, v := range pushVars {
		t.Setenv(v, "")
	}
}

func keyFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(p, []byte(content), 0o400); err != nil {
		t.Fatal(err)
	}
	return p
}

// Without credentials both platforms are off: the binaries send through the fake (pkg/push/pushtest).
func TestLoadPush_IsTheFakeWithoutCredentials(t *testing.T) {
	clearPush(t)

	p, err := config.LoadPush()

	if err != nil || p.FCMCredentials != nil || p.APNs != nil {
		t.Errorf("LoadPush() = %+v, %v; want both off", p, err)
	}
}

// The credentials come from files (the root-only key files of ADR 0011), never from a variable's value.
func TestLoadPush_ReadsBothPlatformsFromTheirFiles(t *testing.T) {
	clearPush(t)
	t.Setenv("FCM_CREDENTIALS_FILE", keyFile(t, `{"type":"service_account"}`))
	t.Setenv("APNS_KEY_FILE", keyFile(t, "-----BEGIN PRIVATE KEY-----\nsynthetic\n-----END PRIVATE KEY-----\n"))
	t.Setenv("APNS_KEY_ID", "SYNTHKEY01")
	t.Setenv("APNS_TEAM_ID", "SYNTHTEAM1")
	t.Setenv("APNS_BUNDLE_ID", "ba.salus.synthetic")
	t.Setenv("APNS_SANDBOX", "true")

	p, err := config.LoadPush()

	if err != nil {
		t.Fatalf("LoadPush() = %v", err)
	}
	if string(p.FCMCredentials) != `{"type":"service_account"}` {
		t.Errorf("FCM credentials %q", p.FCMCredentials)
	}
	if p.APNs == nil || !strings.Contains(string(p.APNs.Key), "synthetic") || p.APNs.KeyID != "SYNTHKEY01" ||
		p.APNs.TeamID != "SYNTHTEAM1" || p.APNs.BundleID != "ba.salus.synthetic" || !p.APNs.Sandbox {
		t.Errorf("APNs %+v", p.APNs)
	}
}

// A half configuration is refused, naming the variable that is missing, never a value or a path.
func TestLoadPush_RefusesAHalfConfiguration(t *testing.T) {
	apns := map[string]string{
		"APNS_KEY_FILE": "", "APNS_KEY_ID": "SYNTHKEY01", "APNS_TEAM_ID": "SYNTHTEAM1", "APNS_BUNDLE_ID": "ba.salus.synthetic",
	}
	for missing := range apns {
		t.Run(missing, func(t *testing.T) {
			clearPush(t)
			for k, v := range apns {
				if k == "APNS_KEY_FILE" {
					v = keyFile(t, "synthetic-p8")
				}
				if k != missing {
					t.Setenv(k, v)
				}
			}
			_, err := config.LoadPush()
			if err == nil || !strings.Contains(err.Error(), missing) || strings.Contains(err.Error(), "SYNTH") {
				t.Errorf("LoadPush() without %s = %v", missing, err)
			}
		})
	}
	for name, set := range map[string]map[string]string{
		"an unreadable FCM file":  {"FCM_CREDENTIALS_FILE": "/nonexistent/fcm.json"},
		"an empty FCM file":       {"FCM_CREDENTIALS_FILE": keyFile(t, " \n")},
		"APNS_SANDBOX not a bool": {"APNS_SANDBOX": "maybe"},
		"APNS_SANDBOX alone":      {"APNS_SANDBOX": "true"},
	} {
		t.Run(name, func(t *testing.T) {
			clearPush(t)
			for k, v := range set {
				t.Setenv(k, v)
			}
			if _, err := config.LoadPush(); err == nil || strings.Contains(err.Error(), "/nonexistent") {
				t.Errorf("LoadPush() = %v, want an error without the path", err)
			}
		})
	}
}

// WEBHOOK_KEY seals the clinics' webhook secrets: base64, from the variable or from the file WEBHOOK_KEY_FILE names.
func TestLoad_ReadsTheWebhookKeyFromTheVariableOrItsFile(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://synthetic")
	t.Setenv("WEBHOOK_KEY", "c3ludGhldGljIDMyLWJ5dGUgd2ViaG9vayBrZXkhISE=")
	cfg, err := config.Load()
	if err != nil || string(cfg.WebhookKey) != "synthetic 32-byte webhook key!!!" {
		t.Fatalf("Load() = %q, %v", cfg.WebhookKey, err)
	}
	t.Setenv("WEBHOOK_KEY", "")
	t.Setenv("WEBHOOK_KEY_FILE", keyFile(t, "c3ludGhldGljIDMyLWJ5dGUgd2ViaG9vayBrZXkhISE=\n"))
	if cfg, err = config.Load(); err != nil || string(cfg.WebhookKey) != "synthetic 32-byte webhook key!!!" {
		t.Errorf("Load() from the file = %q, %v", cfg.WebhookKey, err)
	}
	t.Setenv("WEBHOOK_KEY_FILE", "")
	t.Setenv("WEBHOOK_KEY", "not base64!")
	if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), "WEBHOOK_KEY") || strings.Contains(err.Error(), "not base64!") {
		t.Errorf("Load(bad key) = %v", err)
	}
}
