package trends_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/trends"
)

func serve(t *testing.T, svc trends.View, caller func(context.Context) (trends.Caller, bool, error), path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	trends.NewHandler(svc, caller, slog.New(slog.DiscardHandler)).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

func as(c trends.Caller) func(context.Context) (trends.Caller, bool, error) {
	return func(context.Context) (trends.Caller, bool, error) { return c, true, nil }
}

// allowed are every key a response of the platform view may have (A1): codes and counts, no identifier.
var allowed = map[string]bool{
	"months": true, "quarters": true, "minCount": true, "rows": true, "key": true, "count": true, "partial": true,
	"cells": true, "weekday": true, "hour": true, "rises": true, "week": true, "region": true, "group": true,
}

// walk fails on any key of the JSON value outside allowed.
func walk(t *testing.T, path string, v any) {
	t.Helper()
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			if !allowed[k] {
				t.Errorf("%s: key %q is not in the allow-list", path, k)
			}
			walk(t, path, child)
		}
	case []any:
		for _, child := range x {
			walk(t, path, child)
		}
	}
}

// A1: a test walks the response JSON and fails on any key outside the allow-list.
func TestHandler_ResponsesHoldOnlyAllowedKeys(t *testing.T) {
	svc := newView(published(), &fakeAudit{})
	for _, path := range []string{
		"/analytics/platform/months",
		"/analytics/platform/cells?table=gd&from=2026-09&to=2026-10&by=region",
		"/analytics/platform/cells?table=gd&from=2026-09&to=2026-10&by=code",
		"/analytics/platform/cells?table=gd&from=2026-09&to=2026-10&by=month&region=BA-09",
		"/analytics/platform/heatmap?quarter=2026-Q3",
		"/analytics/platform/rises?weeks=8",
	} {
		rec := serve(t, svc, as(staff), path)
		if rec.Code != http.StatusOK {
			t.Errorf("%s = %d %s, want 200", path, rec.Code, rec.Body)
			continue
		}
		var v any
		if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		walk(t, path, v)
	}
}

// A count of 0 and a suppressed one are both null.
func TestHandler_SendsNullForZeroAndSuppressedAlike(t *testing.T) {
	rec := serve(t, newView(published(), &fakeAudit{}), as(staff), "/analytics/platform/cells?table=gd&from=2026-10&to=2026-10&by=region")
	want := `{"rows":[{"count":null,"key":"BA-01"},{"count":45,"key":"BA-09"},{"count":null,"key":"BA-SRP"}],"partial":false}`
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
		t.Errorf("= %d %s, want %s", rec.Code, rec.Body, want)
	}
}

// stubView answers with err.
type stubView struct{ err error }

func (s stubView) Overview(context.Context, trends.Caller) (trends.Overview, error) {
	return trends.Overview{}, s.err
}

func (s stubView) Cells(context.Context, trends.Caller, trends.Query) (trends.Breakdown, error) {
	return trends.Breakdown{}, s.err
}

func (s stubView) Heatmap(context.Context, trends.Caller, string, string) ([]trends.HeatCell, error) {
	return nil, s.err
}

func (s stubView) Rises(context.Context, trends.Caller, string, int) ([]trends.Flag, error) {
	return nil, s.err
}

func TestHandler_Errors(t *testing.T) {
	nobody := func(context.Context) (trends.Caller, bool, error) { return trends.Caller{}, false, nil }
	broken := func(context.Context) (trends.Caller, bool, error) { return trends.Caller{}, false, errors.New("down") }
	for _, tt := range []struct {
		svc    trends.View
		caller func(context.Context) (trends.Caller, bool, error)
		path   string
		status int
		body   string
	}{
		{stubView{}, nobody, "/analytics/platform/months", 401, `{"error":"unauthenticated"}`},
		{stubView{}, broken, "/analytics/platform/months", 500, ""},
		{stubView{err: trends.ErrNotContributing}, as(optedOut), "/analytics/platform/rises", 403, `{"error":"analytics_not_contributing"}`},
		{stubView{err: trends.ErrForbidden}, as(patientOne), "/analytics/platform/heatmap?quarter=2026-Q3", 403, `{"error":"forbidden"}`},
		{stubView{err: trends.ErrInvalidQuery}, as(staff), "/analytics/platform/cells?table=gd", 400, `{"error":"invalid_request"}`},
		{stubView{}, as(staff), "/analytics/platform/cells?table=gd&clinic=x", 400, `{"error":"invalid_request"}`},
		{stubView{}, as(staff), "/analytics/platform/cells?table=gd&table=gs", 400, `{"error":"invalid_request"}`},
		{stubView{}, as(staff), "/analytics/platform/heatmap?quarter=2026-Q3&age=40", 400, `{"error":"invalid_request"}`},
		{stubView{}, as(staff), "/analytics/platform/rises?weeks=x", 400, `{"error":"invalid_request"}`},
		{stubView{err: errors.New("down")}, as(staff), "/analytics/platform/months", 500, ""},
	} {
		rec := serve(t, tt.svc, tt.caller, tt.path)
		if rec.Code != tt.status || (tt.body != "" && strings.TrimSpace(rec.Body.String()) != tt.body) {
			t.Errorf("%s = %d %s, want %d %s", tt.path, rec.Code, rec.Body, tt.status, tt.body)
		}
	}
}
