package main

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/push"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

func TestRun_StartsTheEmailQueueAndStopsCleanly(t *testing.T) {
	t.Setenv("DATABASE_URL", pgtest.NewDatabaseURL(t))
	setMailpit(t)
	t.Setenv("WEB_BASE_URL", "https://app.example.test")
	t.Setenv("PATIENT_APP_URL", "https://patient.example.test")
	t.Setenv("APPOINTMENT_LINK_URL", "https://api.example.test")
	logs, err := runUntilStarted(t)
	if err != nil {
		t.Fatalf("run() error = %v, want nil after the context ends", err)
	}
	for _, msg := range []string{`"msg":"started"`, `"push_android":"fake"`, `"push_ios":"fake"`, `"msg":"stopped"`} {
		if !strings.Contains(logs, msg) {
			t.Errorf("logs lack %s:\n%s", msg, logs)
		}
	}
}

// setMailpit sets the SMTP settings of the compose Mailpit, as deployments/docker-compose.yml does.
func setMailpit(t *testing.T) {
	t.Helper()
	t.Setenv("SMTP_ADDR", "mailpit:1025")
	t.Setenv("SMTP_TLS", "none")
	t.Setenv("SMTP_FROM", "Salus <no-reply@salus.invalid>")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")
}

func TestRun_FailsWithoutConfig(t *testing.T) {
	for name, env := range map[string][4]string{
		"no DATABASE_URL":    {"", "mailpit:1025", "https://app.example.test", "https://patient.example.test"},
		"no SMTP_ADDR":       {pgtest.NewDatabaseURL(t), "", "https://app.example.test", "https://patient.example.test"},
		"no PATIENT_APP_URL": {pgtest.NewDatabaseURL(t), "mailpit:1025", "https://app.example.test", ""},
	} {
		setMailpit(t)
		t.Setenv("DATABASE_URL", env[0])
		t.Setenv("SMTP_ADDR", env[1])
		t.Setenv("WEB_BASE_URL", env[2])
		t.Setenv("PATIENT_APP_URL", env[3])
		if err := run(t.Context(), &bytes.Buffer{}); err == nil {
			t.Errorf("run() with %s: error = nil", name)
		}
	}
	// An invalid SMTP setting stops the notifier before it connects: credentials in plaintext.
	t.Setenv("DATABASE_URL", pgtest.NewDatabaseURL(t))
	t.Setenv("SMTP_ADDR", "mailpit:1025")
	t.Setenv("SMTP_USERNAME", "user")
	t.Setenv("SMTP_PASSWORD", "synthetic")
	if err := run(t.Context(), &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "SMTP_TLS") {
		t.Errorf("run() with credentials and SMTP_TLS=none: error = %v, want one naming SMTP_TLS", err)
	}
}

// syncBuffer collects run's log output; run writes it while the test reads it.
type syncBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// runUntilStarted runs run until it logs "started", then cancels it and returns its error and logs. Startup
// (connecting to Postgres) gets up to 15 s, so a busy test server does not fail the test.
func runUntilStarted(t *testing.T) (string, error) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	logs := &syncBuffer{}
	done := make(chan error, 1)
	go func() { done <- run(ctx, logs) }()
	deadline := time.After(15 * time.Second)
	for !strings.Contains(logs.String(), `"msg":"started"`) {
		select {
		case err := <-done:
			return logs.String(), err
		case <-deadline:
			cancel()
			err := <-done
			t.Fatalf("run did not log started within 15 s (error after cancel: %v):\n%s", err, logs.String())
		case <-time.After(10 * time.Millisecond):
		}
	}
	cancel()
	err := <-done
	return logs.String(), err
}

// Without credentials both platforms run on the fake; credentials that are not a key refuse the start, naming the
// platform's adapter and never the key (docs/deploy/push.md).
func TestNewPusher_IsTheFakeUntilCredentialsAndRefusesBadOnes(t *testing.T) {
	if _, engines, err := newPusher(config.Push{}); err != nil || engines[push.Android] != "fake" || engines[push.IOS] != "fake" {
		t.Errorf("newPusher(none) = %v, %v", engines, err)
	}
	if _, _, err := newPusher(config.Push{FCMCredentials: []byte(`{"project_id":"p","client_email":"e","private_key":"synthetic-secret"}`)}); err == nil || strings.Contains(err.Error(), "synthetic-secret") {
		t.Errorf("newPusher(bad FCM) = %v", err)
	}
	apns := &config.APNs{Key: []byte("synthetic-secret"), KeyID: "K", TeamID: "T", BundleID: "B"}
	if _, _, err := newPusher(config.Push{APNs: apns}); err == nil || strings.Contains(err.Error(), "synthetic-secret") {
		t.Errorf("newPusher(bad APNs) = %v", err)
	}
}
