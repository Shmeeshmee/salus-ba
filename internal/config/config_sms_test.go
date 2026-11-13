package config_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

func clearSMS(t *testing.T) {
	t.Helper()
	for _, v := range []string{"SMS_ENGINE", "SMS_HTTP_URL", "SMS_HTTP_TOKEN_FILE"} {
		t.Setenv(v, "")
	}
}

func TestLoadSMS_IsTheFakeByDefault(t *testing.T) {
	clearSMS(t)
	if s, err := config.LoadSMS(); err != nil || s.Engine != config.SMSFake {
		t.Errorf("LoadSMS() = %+v, %v; want the fake", s, err)
	}
	t.Setenv("SMS_ENGINE", "fake")
	if s, err := config.LoadSMS(); err != nil || s.Engine != config.SMSFake {
		t.Errorf("SMS_ENGINE=fake: %+v, %v", s, err)
	}
}

// The adapter needs both its endpoint and its token file; nothing half configured, no other engine.
func TestLoadSMS_RefusesTheAdapterWithoutItsCredentials(t *testing.T) {
	token := keyFile(t, "synthetic-sms-token\n")
	for name, env := range map[string]map[string]string{
		"another engine":        {"SMS_ENGINE": "twilio"},
		"http without a URL":    {"SMS_ENGINE": "http", "SMS_HTTP_TOKEN_FILE": token},
		"http without a token":  {"SMS_ENGINE": "http", "SMS_HTTP_URL": "https://sms.example.test/send"},
		"plain http":            {"SMS_ENGINE": "http", "SMS_HTTP_URL": "http://sms.example.test/send", "SMS_HTTP_TOKEN_FILE": token},
		"credentials in URL":    {"SMS_ENGINE": "http", "SMS_HTTP_URL": "https://u:p@sms.example.test/send", "SMS_HTTP_TOKEN_FILE": token},
		"unreadable token":      {"SMS_ENGINE": "http", "SMS_HTTP_URL": "https://sms.example.test/send", "SMS_HTTP_TOKEN_FILE": filepath.Join(t.TempDir(), "none")},
		"credentials with fake": {"SMS_HTTP_URL": "https://sms.example.test/send"},
	} {
		clearSMS(t)
		for k, v := range env {
			t.Setenv(k, v)
		}
		_, err := config.LoadSMS()
		if err == nil {
			t.Errorf("%s: LoadSMS() succeeded", name)
			continue
		}
		if strings.Contains(err.Error(), "example.test") || strings.Contains(err.Error(), token) {
			t.Errorf("%s: the error %q carries a value", name, err)
		}
	}
}

func TestLoadSMS_ReadsTheAdaptersCredentials(t *testing.T) {
	clearSMS(t)
	t.Setenv("SMS_ENGINE", "http")
	t.Setenv("SMS_HTTP_URL", "https://sms.example.test/send")
	t.Setenv("SMS_HTTP_TOKEN_FILE", keyFile(t, "synthetic-sms-token\n"))
	s, err := config.LoadSMS()
	if err != nil || s.Engine != config.SMSHTTP || s.URL != "https://sms.example.test/send" || s.Token != "synthetic-sms-token" {
		t.Errorf("LoadSMS() = %+v, %v", s, err)
	}
}
