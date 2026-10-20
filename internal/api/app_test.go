package api_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

func TestMain(m *testing.M) { pgtest.Main(m, nil) }

func testConfig(t *testing.T) config.Config {
	return config.Config{DatabaseURL: pgtest.NewDatabaseURL(t), RedisURL: redistest.URL(t)}
}

func TestNew_ServesReadinessAgainstTheDatabase(t *testing.T) {
	app, err := api.New(t.Context(), testConfig(t), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer app.Close()

	rec := httptest.NewRecorder()
	app.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("GET /readyz = %d %q, want 200", rec.Code, rec.Body)
	}
}

func TestNew_FailsWhenADependencyIsUnreachable(t *testing.T) {
	noDB := testConfig(t)
	noDB.DatabaseURL = "postgres://nobody@127.0.0.1:1/none?connect_timeout=1"
	noRedis := testConfig(t)
	noRedis.RedisURL = "redis://127.0.0.1:1/0?dial_timeout=1s&max_retries=-1"

	for name, cfg := range map[string]config.Config{"postgres": noDB, "redis": noRedis} {
		if _, err := api.New(t.Context(), cfg, slog.New(slog.DiscardHandler)); err == nil {
			t.Errorf("New() with %s unreachable: error = nil, want a connection error", name)
		}
	}
}
