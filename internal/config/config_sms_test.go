package config_test

import (
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

func TestLoadSMS_IsTheFakeOnly(t *testing.T) {
	for _, engine := range []string{"", "fake"} {
		t.Setenv("SMS_ENGINE", engine)
		if s, err := config.LoadSMS(); err != nil || s.Engine != config.SMSFake {
			t.Errorf("SMS_ENGINE=%q: %+v, %v; want the fake", engine, s, err)
		}
	}
	// Infobip is the provider the adapter is planned for: named, but refused until its DPA is recorded.
	t.Setenv("SMS_ENGINE", "infobip")
	if _, err := config.LoadSMS(); err == nil || !strings.Contains(err.Error(), "processors.md") {
		t.Errorf("SMS_ENGINE=infobip: %v, want refused until its DPA is recorded", err)
	}
	for _, engine := range []string{"http", "synthetic-provider"} {
		t.Setenv("SMS_ENGINE", engine)
		if _, err := config.LoadSMS(); err == nil || strings.Contains(err.Error(), engine) {
			t.Errorf("SMS_ENGINE=%q: %v, want refused without the value", engine, err)
		}
	}
}
