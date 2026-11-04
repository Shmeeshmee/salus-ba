package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

func TestRun_StartsTheEmailQueueAndStopsCleanly(t *testing.T) {
	t.Setenv("DATABASE_URL", pgtest.NewDatabaseURL(t))
	t.Setenv("SMTP_ADDR", "mailpit:1025")
	t.Setenv("WEB_BASE_URL", "https://app.example.test")
	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()
	var logs bytes.Buffer

	if err := run(ctx, &logs); err != nil {
		t.Fatalf("run() error = %v, want nil after the context ends", err)
	}
	for _, msg := range []string{`"msg":"started"`, `"msg":"stopped"`} {
		if !strings.Contains(logs.String(), msg) {
			t.Errorf("logs lack %s:\n%s", msg, logs.String())
		}
	}
}

func TestRun_FailsWithoutConfig(t *testing.T) {
	for name, env := range map[string][3]string{
		"no DATABASE_URL": {"", "mailpit:1025", "https://app.example.test"},
		"no SMTP_ADDR":    {pgtest.NewDatabaseURL(t), "", "https://app.example.test"},
	} {
		t.Setenv("DATABASE_URL", env[0])
		t.Setenv("SMTP_ADDR", env[1])
		t.Setenv("WEB_BASE_URL", env[2])
		if err := run(t.Context(), &bytes.Buffer{}); err == nil {
			t.Errorf("run() with %s: error = nil", name)
		}
	}
}
