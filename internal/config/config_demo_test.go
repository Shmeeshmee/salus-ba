package config_test

import (
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

// The demo seed (salusctl demo-seed, docs/demo) runs only where the stack says it is a developer's: SALUS_ENV=dev
// exactly. Production and staging set no SALUS_ENV, or another value.
func TestIsDev_IsTrueOnlyForSalusEnvDev(t *testing.T) {
	for env, want := range map[string]bool{"dev": true, "": false, "production": false, "staging": false, "DEV": false, " dev": false} {
		t.Setenv("SALUS_ENV", env)
		if got := config.IsDev(); got != want {
			t.Errorf("SALUS_ENV=%q: IsDev() = %v, want %v", env, got, want)
		}
	}
}
