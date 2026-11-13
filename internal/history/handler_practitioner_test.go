package history_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/history"
)

// stubViews answers with what the test set and keeps what it was asked.
type stubViews struct {
	err     error
	page    history.Page
	asked   []string
	cursor  history.Cursor
	summary history.Summary
	limit   int
}

func (s *stubViews) ForPractitioner(_ context.Context, c appointment.Caller, patientID string, at history.Cursor, limit int) (history.Page, error) {
	s.asked = append(s.asked, c.AccountID+" history "+patientID)
	s.cursor, s.limit = at, limit
	return s.page, s.err
}

func (s *stubViews) Summary(_ context.Context, c appointment.Caller, patientID, appointmentID string) (history.Summary, error) {
	s.asked = append(s.asked, c.AccountID+" summary "+patientID+" "+appointmentID)
	return s.summary, s.err
}

func servePract(t *testing.T, views *stubViews, signedIn bool, target string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	logs := &bytes.Buffer{}
	caller := func(context.Context) (appointment.Caller, bool, error) { return pvPract, signedIn, nil }
	mux := http.NewServeMux()
	history.NewPractitionerHandler(views, caller, slog.New(slog.NewTextHandler(logs, nil))).Routes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil))
	return w, logs.String()
}

func TestPractitionerHandler_ServesAPageOfThePatientsHistory(t *testing.T) {
	t.Parallel()
	views := &stubViews{page: history.Page{Next: "abc", Items: []history.Item{{Date: pvDay(1, 9), Clinic: history.Clinic{ID: pvClinicB, Name: "Synthetic clinic B"},
		Type: history.TypeVisit, Reference: "Encounter/" + visit2, EncounterID: visit2, Shared: true}}}}
	at := history.Cursor{Date: pvDay(1, 10), Reference: "Encounter/" + visit1}
	w, _ := servePract(t, views, true, "/patients/"+pvPatientA+"/history?limit=5&cursor="+at.String())
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status %d, Cache-Control %q", w.Code, w.Header().Get("Cache-Control"))
	}
	want := `{"next":"abc","items":[{"date":"2026-10-01T09:00:00Z","clinic":{"id":"` + pvClinicB + `","name":"Synthetic clinic B"},"type":"visit",` +
		`"reference":"Encounter/` + visit2 + `","encounterId":"` + visit2 + `","shared":true}]}`
	if got := strings.TrimSpace(w.Body.String()); got != want {
		t.Errorf("body %s,\nwant %s", got, want)
	}
	if views.cursor != at || views.limit != 5 || views.asked[0] != practAcc+" history "+pvPatientA {
		t.Errorf("asked %v at %+v limit %d", views.asked, views.cursor, views.limit)
	}
}

func TestPractitionerHandler_AnEmptyHistoryIsAnEmptyList(t *testing.T) {
	t.Parallel()
	views := &stubViews{}
	w, _ := servePract(t, views, true, "/patients/"+pvPatientA+"/history")
	if got := strings.TrimSpace(w.Body.String()); w.Code != http.StatusOK || got != `{"items":[]}` {
		t.Errorf("status %d, body %s; want an empty list", w.Code, got)
	}
	if views.limit != 30 {
		t.Errorf("limit %d, want the default 30", views.limit)
	}
}

func TestPractitionerHandler_RefusesABadRequestBeforeTheService(t *testing.T) {
	t.Parallel()
	for _, q := range []string{"?limit=0", "?limit=101", "?limit=x", "?cursor=nope", "?limit=1&limit=2"} {
		views := &stubViews{}
		w, _ := servePract(t, views, true, "/patients/"+pvPatientA+"/history"+q)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"invalid_request"`) || len(views.asked) != 0 {
			t.Errorf("%s: status %d, body %s, asked %v; want invalid_request", q, w.Code, w.Body.String(), views.asked)
		}
	}
}

func TestPractitionerHandler_MapsErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		err    error
		code   string
		status int
	}{
		{history.ErrNotFound, "not_found", http.StatusNotFound},
		{io.ErrUnexpectedEOF, "internal", http.StatusInternalServerError},
	} {
		for _, target := range []string{"/patients/" + pvPatientA + "/history", "/patients/" + pvPatientA + "/history/summary?appointment=" + apptNow} {
			w, logs := servePract(t, &stubViews{err: tt.err}, true, target)
			if w.Code != tt.status || (tt.status == http.StatusNotFound && !strings.Contains(w.Body.String(), `"not_found"`)) {
				t.Errorf("%s with %v: status %d, body %s", target, tt.err, w.Code, w.Body.String())
			}
			if tt.status == http.StatusInternalServerError && (strings.Contains(w.Body.String(), "unexpected EOF") || !strings.Contains(logs, "unexpected EOF")) {
				t.Errorf("%s: the internal error reached the client or missed the log", target)
			}
		}
	}
	for _, target := range []string{"/patients/" + pvPatientA + "/history", "/patients/" + pvPatientA + "/history/summary?appointment=" + apptNow} {
		if w, _ := servePract(t, &stubViews{}, false, target); w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), `"unauthenticated"`) {
			t.Errorf("%s signed out: status %d", target, w.Code)
		}
	}
}

func TestPractitionerHandler_ServesTheSummary(t *testing.T) {
	t.Parallel()
	views := &stubViews{summary: history.Summary{
		Allergies:   []history.SummaryAllergy{{Reference: "AllergyIntolerance/" + allergy1, Text: "Placeholder substance", Confirmed: true}},
		Medications: []history.SummaryMedication{},
		OpenItems:   []history.OpenItem{{Type: history.OpenDraft, Reference: "Composition/" + draftNow}},
	}}
	w, _ := servePract(t, views, true, "/patients/"+pvPatientA+"/history/summary?appointment="+apptNow)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status %d", w.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["lastVisit"]; ok || len(got["allergies"].([]any)) != 1 || len(got["openItems"].([]any)) != 1 {
		t.Errorf("body %s", w.Body.String())
	}
	if views.asked[0] != practAcc+" summary "+pvPatientA+" "+apptNow {
		t.Errorf("asked %v", views.asked)
	}
	if w, _ := servePract(t, &stubViews{}, true, "/patients/"+pvPatientA+"/history/summary"); w.Code != http.StatusNotFound {
		t.Errorf("without an appointment: status %d, want 404", w.Code)
	}
}

// A summary with a section unavailable is served, the section marked, and the failure logged, not sent.
func TestPractitionerHandler_ServesAPartialSummaryAndLogsTheFailure(t *testing.T) {
	t.Parallel()
	views := &stubViews{err: fmt.Errorf("%w: allergies: %w", history.ErrPartial, io.ErrUnexpectedEOF), summary: history.Summary{
		Allergies: []history.SummaryAllergy{}, Medications: []history.SummaryMedication{}, OpenItems: []history.OpenItem{}, AllergiesUnavailable: true,
	}}
	w, logs := servePract(t, views, true, "/patients/"+pvPatientA+"/history/summary?appointment="+apptNow)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"allergiesUnavailable":true`) || strings.Contains(w.Body.String(), "medicationsUnavailable") {
		t.Errorf("status %d, body %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "unexpected EOF") || !strings.Contains(logs, "unexpected EOF") {
		t.Errorf("the failure reached the client or missed the log: %s", logs)
	}
}
