package datarequest_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/datarequest"
)

type stubCloser struct {
	err      error
	patient  datarequest.Patient
	password string
}

func (s *stubCloser) CloseAccount(_ context.Context, p datarequest.Patient, password string) (datarequest.Request, error) {
	s.patient, s.password = p, password
	return datarequest.Request{}, s.err
}

func TestCloseHandler_ClosesTheSignedInAccount(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		err      error
		body     string
		code     string
		status   int
		signedIn bool
	}{
		"closed":            {body: `{"password":"` + password + `"}`, signedIn: true, status: http.StatusNoContent},
		"a wrong password":  {body: `{"password":"x"}`, signedIn: true, err: datarequest.ErrWrongPassword, status: http.StatusForbidden, code: "wrong_credentials"},
		"a staff account":   {body: `{"password":"x"}`, signedIn: true, err: datarequest.ErrNotFound, status: http.StatusForbidden, code: "forbidden"},
		"no password":       {body: `{}`, signedIn: true, status: http.StatusBadRequest, code: "invalid_request"},
		"an unknown field":  {body: `{"password":"x","more":1}`, signedIn: true, status: http.StatusBadRequest, code: "invalid_request"},
		"signed out":        {body: `{"password":"x"}`, status: http.StatusUnauthorized, code: "unauthenticated"},
		"an internal error": {body: `{"password":"x"}`, signedIn: true, err: errors.New("synthetic"), status: http.StatusInternalServerError},
	} {
		t.Run(name, func(t *testing.T) {
			svc := &stubCloser{err: tc.err}
			h := datarequest.NewCloseHandler(svc, func(context.Context) (datarequest.Patient, bool, error) {
				return datarequest.Patient{AccountID: accountID}, tc.signedIn, nil
			}, slog.New(slog.DiscardHandler))
			mux := http.NewServeMux()
			h.Routes(mux)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/me/account/close", strings.NewReader(tc.body)))
			if rec.Code != tc.status || tc.code != "" && !strings.Contains(rec.Body.String(), `"`+tc.code+`"`) {
				t.Errorf("%d %s, want %d %s", rec.Code, rec.Body, tc.status, tc.code)
			}
			if tc.status == http.StatusNoContent && (svc.patient.AccountID != accountID || svc.password != password) {
				t.Errorf("closed %+v with %q", svc.patient, svc.password)
			}
		})
	}
}

func TestCloseHandler_OverTheLimitIs429WithRetryAfter(t *testing.T) {
	t.Parallel()
	svc := &stubCloser{err: fmt.Errorf("datarequest: close account: %w", &auth.RateLimitedError{RetryAfter: 90 * time.Second})}
	h := datarequest.NewCloseHandler(svc, func(context.Context) (datarequest.Patient, bool, error) {
		return datarequest.Patient{AccountID: accountID}, true, nil
	}, slog.New(slog.DiscardHandler))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/me/account/close", strings.NewReader(`{"password":"x"}`)))
	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), `"rate_limited"`) || rec.Header().Get("Retry-After") != "90" {
		t.Errorf("%d %s Retry-After %q, want 429 rate_limited 90", rec.Code, rec.Body, rec.Header().Get("Retry-After"))
	}
}
