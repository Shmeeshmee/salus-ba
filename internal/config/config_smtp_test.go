package config_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

// setSMTP sets every SMTP_* variable: the values of a relay with STARTTLS, with any of them replaced by over.
func setSMTP(t *testing.T, over map[string]string) {
	t.Helper()
	env := map[string]string{
		"SMTP_ADDR": "smtp-relay.example.test:587", "SMTP_TLS": "starttls",
		"SMTP_USERNAME": "user-marker", "SMTP_PASSWORD": "password-marker",
		"SMTP_FROM": "Salus <noreply@example.test>",
	}
	for k, v := range over {
		env[k] = v
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
}

func TestLoadSMTP_ReadsTheRelaysSettings(t *testing.T) {
	setSMTP(t, nil)

	s, err := config.LoadSMTP()

	if err != nil {
		t.Fatalf("LoadSMTP() error = %v", err)
	}
	if s.Addr != "smtp-relay.example.test:587" || s.TLS != config.SMTPStartTLS || s.Username != "user-marker" || s.Password != "password-marker" {
		t.Errorf("LoadSMTP() = %+v", s)
	}
	if s.From.Name != "Salus" || s.From.Address != "noreply@example.test" {
		t.Errorf("From = %+v, want Salus <noreply@example.test>", s.From)
	}
}

// Implicit TLS (465), and Mailpit's plaintext without credentials, with a bare from address.
func TestLoadSMTP_ReadsImplicitTLSAndPlaintextWithoutCredentials(t *testing.T) {
	for mode, want := range map[string]config.SMTPTLS{"tls": config.SMTPImplicitTLS, "none": config.SMTPPlaintext} {
		t.Run(mode, func(t *testing.T) {
			setSMTP(t, map[string]string{"SMTP_TLS": mode, "SMTP_USERNAME": "", "SMTP_PASSWORD": "", "SMTP_FROM": "no-reply@salus.invalid"})

			s, err := config.LoadSMTP()

			if err != nil || s.TLS != want || s.Username != "" || s.Password != "" || s.From.Address != "no-reply@salus.invalid" || s.From.Name != "" {
				t.Errorf("LoadSMTP() = %+v, %v", s, err)
			}
		})
	}
}

func TestLoadSMTP_RefusesAMissingOrMalformedSettingWithoutEchoingIt(t *testing.T) {
	for name, tc := range map[string]struct {
		over     map[string]string
		variable string
	}{
		"no address":                {map[string]string{"SMTP_ADDR": ""}, "SMTP_ADDR"},
		"an address without a port": {map[string]string{"SMTP_ADDR": "marker.example.test"}, "SMTP_ADDR"},
		"an address without a host": {map[string]string{"SMTP_ADDR": ":587"}, "SMTP_ADDR"},
		"no TLS mode":               {map[string]string{"SMTP_TLS": ""}, "SMTP_TLS"},
		"an unknown TLS mode":       {map[string]string{"SMTP_TLS": "marker"}, "SMTP_TLS"},
		"a username alone":          {map[string]string{"SMTP_PASSWORD": ""}, "SMTP_PASSWORD"},
		"a password alone":          {map[string]string{"SMTP_USERNAME": ""}, "SMTP_USERNAME"},
		"credentials in plaintext":  {map[string]string{"SMTP_TLS": "none"}, "SMTP_TLS"},
		"a password in plaintext":   {map[string]string{"SMTP_TLS": "none", "SMTP_USERNAME": ""}, "SMTP_TLS"},
		"no from address":           {map[string]string{"SMTP_FROM": ""}, "SMTP_FROM"},
		"a from that is no address": {map[string]string{"SMTP_FROM": "Salus marker"}, "SMTP_FROM"},
		"two from addresses":        {map[string]string{"SMTP_FROM": "a@marker.test, b@marker.test"}, "SMTP_FROM"},
		"a from with a line break":  {map[string]string{"SMTP_FROM": "Salus <a@marker.test>\r\nBcc: b@marker.test"}, "SMTP_FROM"},
	} {
		t.Run(name, func(t *testing.T) {
			setSMTP(t, tc.over)

			_, err := config.LoadSMTP()

			if err == nil || !strings.Contains(err.Error(), tc.variable) || strings.Contains(err.Error(), "marker") {
				t.Errorf("LoadSMTP() = %v, want an error naming %s and no value", err, tc.variable)
			}
		})
	}
}

// Only the notifier loads the SMTP settings, and Load never reads them: the password reaches no other binary.
func TestLoadSMTP_IsCalledOnlyByTheNotifier(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	setSMTP(t, nil)
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%#v", cfg); strings.Contains(got, "password-marker") || strings.Contains(got, "smtp-relay") {
		t.Errorf("Load() holds the SMTP settings: %s", got)
	}

	root := filepath.Join("..", "..")
	var callers, readers []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "tools") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if strings.Contains(string(src), "config.LoadSMTP(") {
			callers = append(callers, filepath.ToSlash(rel))
		}
		if strings.Contains(string(src), `Getenv("SMTP_`) {
			readers = append(readers, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"cmd/notifier/main.go"}; !slices.Equal(callers, want) {
		t.Errorf("LoadSMTP is called by %v, want %v", callers, want)
	}
	if want := []string{"internal/config/config_smtp.go"}; !slices.Equal(readers, want) {
		t.Errorf("the SMTP variables are named in %v, want only %v", readers, want)
	}
}
