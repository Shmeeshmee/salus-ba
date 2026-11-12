package api_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

func testConfig(t *testing.T) config.Config {
	return config.Config{
		DatabaseURL: pgtest.NewDatabaseURL(t), RedisURL: redistest.URL(t),
		TOTPKey:      []byte("synthetic 32-byte TOTP key, test"),
		RateLimitKey: []byte("synthetic 32-byte rate-limit key"),
		FHIRBaseURL:  "https://api.example.test/fhir/R4",
	}
}

// testObjectStore is the compose object store (TEST_S3_*, as in internal/binary) and a synthetic
// key-encryption key. Like the databases, the tests fail without it.
func testObjectStore(t *testing.T) config.ObjectStore {
	t.Helper()
	s := config.ObjectStore{
		Endpoint: os.Getenv("TEST_S3_ENDPOINT"), Region: os.Getenv("TEST_S3_REGION"), Bucket: os.Getenv("TEST_S3_BUCKET"),
		AccessKeyID: os.Getenv("TEST_S3_ACCESS_KEY_ID"), SecretAccessKey: os.Getenv("TEST_S3_SECRET_ACCESS_KEY"),
		KEKID: "test-1", KEK: []byte("synthetic 32-byte KEK for tests!"),
	}
	if s.Endpoint == "" || s.Region == "" || s.Bucket == "" || s.AccessKeyID == "" || s.SecretAccessKey == "" {
		t.Fatal("TEST_S3_* are not set; run the tests with `make test`, which points them at the compose object store")
	}
	return s
}

func TestNew_ServesReadinessAgainstTheDatabase(t *testing.T) {
	app, err := api.New(t.Context(), testConfig(t), testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	rec := httptest.NewRecorder()
	app.Handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("GET /readyz = %d %q, want 200", rec.Code, rec.Body)
	}
}

func TestNew_FailsWhenADependencyIsUnreachable(t *testing.T) {
	noDB := testConfig(t)
	noDB.DatabaseURL = "postgres://nobody@127.0.0.1:1/none?connect_timeout=1"
	noRedis := testConfig(t)
	noRedis.RedisURL = "redis://127.0.0.1:1/0?dial_timeout=1s&max_retries=-1"

	shortTOTPKey := testConfig(t)
	shortTOTPKey.TOTPKey = []byte("16 bytes, no more")
	shortRateKey := testConfig(t)
	shortRateKey.RateLimitKey = []byte("short")
	noFHIRBase := testConfig(t)
	noFHIRBase.FHIRBaseURL = ""
	relativeFHIRBase := testConfig(t)
	relativeFHIRBase.FHIRBaseURL = "/fhir/R4"

	shortKEK := testObjectStore(t)
	shortKEK.KEK = []byte("16 bytes, no more")
	if _, err := api.New(t.Context(), testConfig(t), testLiveKit(), shortKEK, testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler)); err == nil || strings.Contains(err.Error(), "16 bytes") {
		t.Errorf("New() with a KEK that is not 32 bytes: error = %v, want one that names no value", err)
	}
	for name, cfg := range map[string]config.Config{
		"postgres unreachable": noDB, "redis unreachable": noRedis,
		"a TOTP key that is not 32 bytes": shortTOTPKey, "a rate-limit key under 32 bytes": shortRateKey,
		"no FHIR base URL": noFHIRBase, "a relative FHIR base URL": relativeFHIRBase,
	} {
		if _, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler)); err == nil {
			t.Errorf("New() with %s: error = nil", name)
		}
	}
}

func TestNew_ServesTheAccountRoutes(t *testing.T) {
	app, err := api.New(t.Context(), testConfig(t), testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"email":"ana@example.test","password":"a long enough secret"}`)
	app.Handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", body))
	if rec.Code != http.StatusCreated {
		t.Errorf("POST /auth/register = %d %q, want 201", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	app.Handler.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/me", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /auth/me without a session = %d %q, want 401", rec.Code, rec.Body)
	}
}

func TestNew_AuditsEveryRegistrationInTheDatabase(t *testing.T) {
	cfg := testConfig(t)
	app, err := api.New(t.Context(), cfg, testLiveKit(), testObjectStore(t), testDrafter(), testSummarizer(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	for range 2 { // the second one finds the e-mail taken
		body := strings.NewReader(`{"email":"ben@example.test","password":"a long enough secret"}`)
		app.Handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/register", body))
	}

	db, err := postgres.Open(t.Context(), cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var created, taken int
	err = db.QueryRow(t.Context(), `SELECT
		count(*) FILTER (WHERE outcome = 0 AND entity_reference = 'Account/' || agent_account_id::text),
		count(*) FILTER (WHERE outcome = 4 AND agent_account_id IS NULL AND entity_reference IS NULL)
		FROM audit_event WHERE action = 'C' AND subtype_code = 'create'`).Scan(&created, &taken)
	if err != nil {
		t.Fatal(err)
	}
	if created != 1 || taken != 1 {
		t.Errorf("audited %d registrations and %d taken e-mails, want 1 and 1", created, taken)
	}
}
