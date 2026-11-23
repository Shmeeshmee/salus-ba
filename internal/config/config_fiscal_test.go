package config_test

import (
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

func TestLoadFiscal_IsTheFakeOnly(t *testing.T) {
	for _, engine := range []string{"", "fake"} {
		t.Setenv("FISCAL_ENGINE", engine)
		if f, err := config.LoadFiscal(); err != nil || f.Engine != config.FiscalFake {
			t.Errorf("FISCAL_ENGINE=%q: %+v, %v; want the fake", engine, f, err)
		}
	}
	// ESET is the Tax Administration's system the adapter is planned for: named, but refused until package G.
	t.Setenv("FISCAL_ENGINE", "eset")
	if _, err := config.LoadFiscal(); err == nil || !strings.Contains(err.Error(), "package G") {
		t.Errorf("FISCAL_ENGINE=eset: %v, want refused until its adapter exists", err)
	}
	for _, engine := range []string{"http", "synthetic-engine"} {
		t.Setenv("FISCAL_ENGINE", engine)
		if _, err := config.LoadFiscal(); err == nil || strings.Contains(err.Error(), engine) {
			t.Errorf("FISCAL_ENGINE=%q: %v, want refused without the value", engine, err)
		}
	}
}
