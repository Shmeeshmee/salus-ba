package config_test

import (
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

// The platform view is off unless ANALYTICS_PLATFORM_ENABLED is true (docs/plans/analytics.md §2a); k is 10 unless
// ANALYTICS_MIN_COUNT sets a larger one, and below 10 is refused (A4).
func TestLoad_ReadsTheAnalyticsSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("ANALYTICS_PLATFORM_ENABLED", "")
	t.Setenv("ANALYTICS_MIN_COUNT", "")
	cfg, err := config.Load()
	if err != nil || cfg.Analytics.PlatformEnabled || cfg.Analytics.MinCount != 10 {
		t.Errorf("defaults = %+v, %v; want off and 10", cfg.Analytics, err)
	}
	t.Setenv("ANALYTICS_PLATFORM_ENABLED", "true")
	t.Setenv("ANALYTICS_MIN_COUNT", "25")
	if cfg, err = config.Load(); err != nil || !cfg.Analytics.PlatformEnabled || cfg.Analytics.MinCount != 25 {
		t.Errorf("set = %+v, %v; want on and 25", cfg.Analytics, err)
	}
	t.Setenv("ANALYTICS_PLATFORM_ENABLED", "false")
	if cfg, err = config.Load(); err != nil || cfg.Analytics.PlatformEnabled {
		t.Errorf("false = %+v, %v; want off", cfg.Analytics, err)
	}
	for name, value := range map[string]string{
		"ANALYTICS_PLATFORM_ENABLED": "yes", "ANALYTICS_MIN_COUNT": "9",
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(name, value)
			if _, err := config.Load(); err == nil || !strings.Contains(err.Error(), name) || strings.Contains(err.Error(), value) {
				t.Errorf("%s=%s: %v, want an error naming the variable, not the value", name, value, err)
			}
		})
	}
	t.Setenv("ANALYTICS_MIN_COUNT", "ten")
	if _, err := config.Load(); err == nil {
		t.Error("ANALYTICS_MIN_COUNT=ten loads")
	}
}
