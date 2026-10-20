package httpx_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/httpx"
)

func TestJSON_WritesStatusContentTypeAndBody(t *testing.T) {
	rec := httptest.NewRecorder()

	httpx.JSON(rec, http.StatusCreated, map[string]string{"status": "ok"})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if body := rec.Body.String(); body != "{\"status\":\"ok\"}\n" {
		t.Errorf("body = %q", body)
	}
}

func TestInternalError_LogsErrorAndSendsFixedMessage(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/things", nil)

	httpx.InternalError(rec, req, logger, errors.New("connection refused to 10.0.0.7"))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if body := rec.Body.String(); body != "{\"error\":\"internal error\"}\n" {
		t.Errorf("body = %q, want the fixed message", body)
	}
	if !strings.Contains(logs.String(), "connection refused to 10.0.0.7") {
		t.Errorf("log %q does not carry the error", logs.String())
	}
}
