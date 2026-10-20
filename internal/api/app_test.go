package api_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shmeeshmee/salus/internal/api"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, nil) }

func TestNew_ServesReadinessAgainstTheDatabase(t *testing.T) {
	app, err := api.New(t.Context(), config.Config{DatabaseURL: pgtest.NewDatabaseURL(t)}, slog.New(slog.DiscardHandler))
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

func TestNew_FailsWhenTheDatabaseIsUnreachable(t *testing.T) {
	cfg := config.Config{DatabaseURL: "postgres://nobody@127.0.0.1:1/none?connect_timeout=1"}

	if _, err := api.New(t.Context(), cfg, slog.New(slog.DiscardHandler)); err == nil {
		t.Fatal("New() error = nil, want a connection error")
	}
}
