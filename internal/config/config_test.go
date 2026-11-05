package config_test

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/config"
)

func TestLoad_ReadsEnvironment(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("HTTP_ADDR", ":9000")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("APP_DATABASE_PASSWORD", "synthetic")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != "postgres://test@localhost/test" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.HTTPAddr != ":9000" {
		t.Errorf("HTTPAddr = %q, want :9000", cfg.HTTPAddr)
	}
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("RedisURL = %q", cfg.RedisURL)
	}
	if cfg.AppDatabasePassword != "synthetic" {
		t.Error("AppDatabasePassword is not read from APP_DATABASE_PASSWORD")
	}
}

func TestLoad_DefaultsHTTPAddr(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("HTTP_ADDR", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
}

func TestLoad_RequiresDatabaseURLWithoutEchoingValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("HTTP_ADDR", "secret-looking-value")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() error = nil, want an error for the missing DATABASE_URL")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error %q does not name DATABASE_URL", err)
	}
	if strings.Contains(err.Error(), "secret-looking-value") {
		t.Errorf("error %q leaks an environment value", err)
	}
}

func TestLoad_ReadsTheAuthKeysAndTrustedProxies(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("TOTP_KEY", base64.StdEncoding.EncodeToString([]byte("synthetic 32-byte key for tests!")))
	t.Setenv("RATE_LIMIT_KEY", base64.StdEncoding.EncodeToString([]byte("synthetic rate-limit key, 32 b.")))
	t.Setenv("TRUSTED_PROXY_CIDRS", " 10.0.0.0/8, fd00::/8 ,")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if string(cfg.TOTPKey) != "synthetic 32-byte key for tests!" || string(cfg.RateLimitKey) != "synthetic rate-limit key, 32 b." {
		t.Error("TOTPKey or RateLimitKey is not the base64-decoded variable")
	}
	want := []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("fd00::/8")}
	if !slices.Equal(cfg.TrustedProxies, want) {
		t.Errorf("TrustedProxies = %v, want %v", cfg.TrustedProxies, want)
	}
}

func TestLoad_RejectsMalformedAuthSettingsWithoutEchoingThem(t *testing.T) {
	for name, value := range map[string]string{
		"TOTP_KEY":            "not*base64*secret",
		"RATE_LIMIT_KEY":      "not*base64*secret",
		"TRUSTED_PROXY_CIDRS": "10.0.0.0/8,not*base64*secret",
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
			t.Setenv(name, value)
			_, err := config.Load()
			if err == nil || !strings.Contains(err.Error(), name) || strings.Contains(err.Error(), "secret") {
				t.Errorf("Load() error = %v, want one naming %s without its value", err, name)
			}
		})
	}
}

func TestLoad_ReadsTheNotifiersSettings(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("SMTP_ADDR", "mailpit:1025")
	t.Setenv("WEB_BASE_URL", "http://localhost:5173")
	t.Setenv("PATIENT_APP_URL", "https://patient.example.test")

	cfg, err := config.Load()
	if err != nil || cfg.SMTPAddr != "mailpit:1025" || cfg.WebBaseURL != "http://localhost:5173" || cfg.PatientAppURL != "https://patient.example.test" {
		t.Errorf("Load() = SMTPAddr %q, WebBaseURL %q, PatientAppURL %q, %v", cfg.SMTPAddr, cfg.WebBaseURL, cfg.PatientAppURL, err)
	}
}

func TestLoad_ReadsTheFHIRBaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("FHIR_BASE_URL", "http://localhost:8080/fhir/R4")

	cfg, err := config.Load()
	if err != nil || cfg.FHIRBaseURL != "http://localhost:8080/fhir/R4" {
		t.Errorf("Load() = FHIRBaseURL %q, %v", cfg.FHIRBaseURL, err)
	}
}

func TestLoadRetention_ReadsTheRetentionRolesCredentials(t *testing.T) {
	t.Setenv("RETENTION_DATABASE_URL", "postgres://salus_retention:synthetic@localhost/test")
	t.Setenv("RETENTION_DATABASE_PASSWORD", "synthetic")

	ret := config.LoadRetention()

	if ret.DatabaseURL != "postgres://salus_retention:synthetic@localhost/test" || ret.Password != "synthetic" {
		t.Error("LoadRetention() does not read RETENTION_DATABASE_URL and RETENTION_DATABASE_PASSWORD")
	}
}

// The api, the notifier and the scheduler load only Load's configuration: with the retention credentials
// in the environment, none of them reaches what they load.
func TestLoad_NeverReadsTheRetentionRolesCredentials(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("RETENTION_DATABASE_URL", "postgres://salus_retention:retention-url-marker@localhost/test")
	t.Setenv("RETENTION_DATABASE_PASSWORD", "retention-password-marker")

	cfg, err := config.Load()

	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%#v", cfg); strings.Contains(got, "retention-url-marker") || strings.Contains(got, "retention-password-marker") {
		t.Errorf("Load() holds the retention credentials: %s", got)
	}
}

// Only cmd/worker (for the retention_apply job's pool) and cmd/migrate (to set the role's password) call
// LoadRetention, and no other Go file reads the variables.
func TestLoadRetention_IsCalledOnlyByTheWorkerAndMigrate(t *testing.T) {
	root := filepath.Join("..", "..")
	var callers, readers []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		if strings.Contains(string(src), "config.LoadRetention(") {
			callers = append(callers, filepath.ToSlash(rel))
		}
		if strings.Contains(string(src), `Getenv("RETENTION_DATABASE_`) {
			readers = append(readers, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"cmd/migrate/main.go", "cmd/worker/main.go"}; !slices.Equal(callers, want) {
		t.Errorf("LoadRetention is called by %v, want %v", callers, want)
	}
	if want := []string{"internal/config/config.go"}; !slices.Equal(readers, want) {
		t.Errorf("the retention variables are named in %v, want only %v", readers, want)
	}
}

func TestLoadLiveKit_ReadsTheFourSettings(t *testing.T) {
	t.Setenv("LIVEKIT_URL", "http://livekit:7880")
	t.Setenv("LIVEKIT_PUBLIC_URL", "ws://localhost:7880")
	t.Setenv("LIVEKIT_API_KEY", "synthetic-key")
	t.Setenv("LIVEKIT_API_SECRET", "synthetic-secret")

	got, err := config.LoadLiveKit()

	want := config.LiveKit{URL: "http://livekit:7880", PublicURL: "ws://localhost:7880", APIKey: "synthetic-key", APISecret: "synthetic-secret"}
	if err != nil || got != want {
		t.Errorf("LoadLiveKit() = %+v, %v; want %+v", got, err, want)
	}
}

// Each setting is required; the error names the variable and never echoes a value.
func TestLoadLiveKit_RequiresEachSettingWithoutEchoingValues(t *testing.T) {
	all := map[string]string{
		"LIVEKIT_URL": "http://livekit:7880", "LIVEKIT_PUBLIC_URL": "ws://localhost:7880",
		"LIVEKIT_API_KEY": "key-marker", "LIVEKIT_API_SECRET": "secret-marker",
	}
	for missing := range all {
		t.Run(missing, func(t *testing.T) {
			for name, v := range all {
				if name == missing {
					v = ""
				}
				t.Setenv(name, v)
			}
			_, err := config.LoadLiveKit()
			if err == nil || !strings.Contains(err.Error(), missing) || strings.Contains(err.Error(), "marker") {
				t.Errorf("LoadLiveKit() without %s: %v; want an error naming it and no value", missing, err)
			}
		})
	}
}

// The api and the worker load the LiveKit settings; the notifier and the scheduler do not (SPEC.md
// "Configuration" in docs/consultation), and Load never reads them.
func TestLoadLiveKit_IsCalledOnlyByTheAPIAndTheWorker(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test@localhost/test")
	t.Setenv("LIVEKIT_API_SECRET", "livekit-secret-marker")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%#v", cfg); strings.Contains(got, "livekit-secret-marker") {
		t.Errorf("Load() holds the LiveKit secret: %s", got)
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
		if strings.Contains(string(src), "config.LoadLiveKit(") {
			callers = append(callers, filepath.ToSlash(rel))
		}
		if strings.Contains(string(src), `Getenv("LIVEKIT_`) {
			readers = append(readers, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"cmd/api/main.go", "cmd/worker/main.go"}; !slices.Equal(callers, want) {
		t.Errorf("LoadLiveKit is called by %v, want %v", callers, want)
	}
	if want := []string{"internal/config/config.go"}; !slices.Equal(readers, want) {
		t.Errorf("the LiveKit variables are named in %v, want only %v", readers, want)
	}
}
