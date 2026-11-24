package config_test

import (
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

func TestLoadOperator_ReadsTheAddressAndTheLanguage(t *testing.T) {
	t.Setenv("SALUS_ENV", "production")
	t.Setenv("SALUS_OPERATOR_EMAIL", " operator@example.test ")
	t.Setenv("SALUS_OPERATOR_LANGUAGE", "hr")
	got, err := config.LoadOperator()
	if err != nil || got != (config.Operator{Email: "operator@example.test", Language: "hr"}) {
		t.Errorf("LoadOperator() = %+v, %v", got, err)
	}
	t.Setenv("SALUS_OPERATOR_LANGUAGE", "")
	if got, err := config.LoadOperator(); err != nil || got.Language != "bs" {
		t.Errorf("no language: %+v, %v; want bs", got, err)
	}
	t.Setenv("SALUS_OPERATOR_LANGUAGE", "xx")
	if _, err := config.LoadOperator(); err == nil || !strings.Contains(err.Error(), "SALUS_OPERATOR_LANGUAGE") {
		t.Errorf("an unknown language: error = %v, want one naming the variable", err)
	}
}

// Outside dev the operator's deadline warnings need an address; the .env.example placeholder is none. In dev there
// may be none (the worker logs the warning instead).
func TestLoadOperator_TheAddressIsRequiredOutsideDev(t *testing.T) {
	for _, email := range []string{"", "[OPERATOR EMAIL]"} {
		t.Setenv("SALUS_OPERATOR_EMAIL", email)
		t.Setenv("SALUS_OPERATOR_LANGUAGE", "")
		t.Setenv("SALUS_ENV", "production")
		if _, err := config.LoadOperator(); err == nil || !strings.Contains(err.Error(), "SALUS_OPERATOR_EMAIL") {
			t.Errorf("%q outside dev: error = %v, want one naming the variable", email, err)
		}
		t.Setenv("SALUS_ENV", config.EnvDev)
		if got, err := config.LoadOperator(); err != nil || got.Email != "" || got.Language != "bs" {
			t.Errorf("%q in dev: %+v, %v; want none", email, got, err)
		}
	}
}
