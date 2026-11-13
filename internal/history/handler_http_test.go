package history_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/history"
)

// stubTimelines answers Timeline with out and err, and keeps what it was asked.
type stubTimelines struct {
	err    error
	caller appointment.Caller
	filter history.Filter
	cursor history.Cursor
	out    history.Page
	limit  int
	calls  int
}

func (s *stubTimelines) Timeline(_ context.Context, c appointment.Caller, f history.Filter, cur history.Cursor, limit int) (history.Page, error) {
	s.calls++
	s.caller, s.filter, s.cursor, s.limit = c, f, cur, limit
	return s.out, s.err
}

// serve runs the request on the handler with the caller signed in (none when nil).
func serve(t *testing.T, svc *stubTimelines, c *appointment.Caller, target string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	var logs bytes.Buffer
	caller := func(context.Context) (appointment.Caller, bool, error) {
		if c == nil {
			return appointment.Caller{}, false, nil
		}
		return *c, true, nil
	}
	mux := http.NewServeMux()
	history.NewHTTPHandler(svc, caller, slog.New(slog.NewTextHandler(&logs, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil))
	return rec, logs.String()
}

// GET /me/history gives the page as JSON, not cached, with the default limit and no filter.
func TestHTTP_ListMyHistoryGivesThePage(t *testing.T) {
	svc := &stubTimelines{out: history.Page{Next: "bmV4dA", Items: []history.Item{{
		Date: day, Clinic: history.Clinic{ID: clinicA, Name: "Synthetic clinic A"}, Type: history.TypeVisit,
		Reference: "Encounter/" + visitA, EncounterID: visitA, Specialty: "394579002",
	}}}}
	rec, _ := serve(t, svc, &me, "/me/history")
	want := `{"next":"bmV4dA","items":[{"date":"2030-01-07T09:00:00Z","clinic":{"id":"` + clinicA + `","name":"Synthetic clinic A"},` +
		`"type":"visit","reference":"Encounter/` + visitA + `","encounterId":"` + visitA + `","specialty":"394579002"}]}`
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("GET = %d %s, Cache-Control %q;\nwant 200 %s, no-store", rec.Code, rec.Body, rec.Header().Get("Cache-Control"), want)
	}
	if svc.caller != me || svc.limit != 30 || svc.cursor != (history.Cursor{}) || svc.filter.OrganizationID != "" || svc.filter.Types != nil {
		t.Errorf("asked %+v %+v %+v %d, want the patient's first page of 30, unfiltered", svc.caller, svc.filter, svc.cursor, svc.limit)
	}
}

// The query's cursor, limit, clinic, types and range reach the service; an empty page is an empty list.
func TestHTTP_ListMyHistoryPassesTheQuery(t *testing.T) {
	svc := &stubTimelines{out: history.Page{}}
	cur := history.Cursor{Date: day, Reference: "Encounter/" + visitA}
	rec, _ := serve(t, svc, &me, "/me/history?cursor="+cur.String()+"&limit=5&organization="+clinicB+
		"&type=visit&type=attachment&from=2030-01-01T00:00:00%2B01:00&to=2030-02-01T00:00:00Z")
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"items":[]}` {
		t.Fatalf("GET = %d %s, want 200 and an empty list", rec.Code, rec.Body)
	}
	from := time.Date(2029, 12, 31, 23, 0, 0, 0, time.UTC)
	if svc.cursor != cur || svc.limit != 5 || svc.filter.OrganizationID != clinicB || !svc.filter.From.Equal(from) ||
		!svc.filter.To.Equal(time.Date(2030, 2, 1, 0, 0, 0, 0, time.UTC)) ||
		!slices.Equal(svc.filter.Types, []history.ItemType{history.TypeVisit, history.TypeAttachment}) {
		t.Errorf("asked %+v %+v %d", svc.filter, svc.cursor, svc.limit)
	}
}

// A query this API did not make is 400 invalid_request, before the service is asked.
func TestHTTP_ListMyHistoryRefusesABadQuery(t *testing.T) {
	for _, q := range []string{
		"cursor=x", "cursor=" + strings.Repeat("a", 513), "limit=0", "limit=101", "limit=x", "organization=x", "type=chat",
		"from=2030-01-01", "to=yesterday", "from=2030-01-01T00:00:00", "limit=1&limit=2",
	} {
		svc := &stubTimelines{}
		rec, _ := serve(t, svc, &me, "/me/history?"+q)
		if rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != `{"error":"invalid_request"}` || svc.calls != 0 {
			t.Errorf("%s: GET = %d %s, called %d; want 400 invalid_request before the service", q, rec.Code, rec.Body, svc.calls)
		}
	}
}

// Nobody signed in is 401; staff 403; a failure 500 with a fixed message and nothing of the history logged.
func TestHTTP_ListMyHistoryErrors(t *testing.T) {
	if rec, _ := serve(t, &stubTimelines{}, nil, "/me/history"); rec.Code != http.StatusUnauthorized ||
		strings.TrimSpace(rec.Body.String()) != `{"error":"unauthenticated"}` {
		t.Errorf("signed out: GET = %d %s, want 401", rec.Code, rec.Body)
	}
	if rec, _ := serve(t, &stubTimelines{err: history.ErrForbidden}, &pract, "/me/history"); rec.Code != http.StatusForbidden ||
		strings.TrimSpace(rec.Body.String()) != `{"error":"forbidden"}` {
		t.Errorf("staff: GET = %d %s, want 403", rec.Code, rec.Body)
	}
	if rec, _ := serve(t, &stubTimelines{err: history.ErrLimit}, &me, "/me/history"); rec.Code != http.StatusBadRequest {
		t.Errorf("ErrLimit: GET = %d, want 400", rec.Code)
	}
	rec, logs := serve(t, &stubTimelines{err: errBackend}, &me, "/me/history?organization="+clinicA)
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "database") {
		t.Errorf("failure: GET = %d %s, want 500 with a fixed message", rec.Code, rec.Body)
	}
	if strings.Contains(logs, clinicA) {
		t.Errorf("the log holds the query: %s", logs)
	}
}
