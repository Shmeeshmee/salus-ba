package schedule_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/schedule"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

// assertGolden compares got with testdata/<name>; -update rewrites the file.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Errorf("%s differs from the response:\n got: %s\nwant: %s", path, got, want)
	}
}

// stubPublic answers with what it is given and keeps what it was asked.
type stubPublic struct {
	err   error
	asked []string // "read:<id>" or "actor:<role>"
	out   schedule.Schedule
}

func (s *stubPublic) Read(_ context.Context, id string) (schedule.Schedule, error) {
	s.asked = append(s.asked, "read:"+id)
	return s.out, s.err
}

func (s *stubPublic) ByActor(_ context.Context, roleID string) (schedule.Schedule, error) {
	s.asked = append(s.asked, "actor:"+roleID)
	return s.out, s.err
}

func getSchedule(t *testing.T, svc *stubPublic, log *bytes.Buffer, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	schedule.NewHandler(svc, "https://api.example.test/fhir/R4", slog.New(slog.NewJSONHandler(log, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
	return rec
}

var scheduleOfA2 = schedule.Schedule{ID: scheduleA2, OrganizationID: clinicA, PractitionerRoleID: roleA2, SlotMinutes: 30,
	Rules: []schedule.Rule{{Day: 1, Start: "08:00", End: "12:00"}}, VersionID: 2, LastUpdated: now}

// Anyone, signed in or not, reads the schedule of an active role of a listed clinic (SPEC.md "Public browse").
func TestHandlerRead_SendsTheScheduleWithItsActorMetaAndETag(t *testing.T) {
	var log bytes.Buffer
	svc := &stubPublic{out: scheduleOfA2}

	rec := getSchedule(t, svc, &log, "/fhir/R4/Schedule/"+scheduleA2)

	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"2"` || len(svc.asked) != 1 || svc.asked[0] != "read:"+scheduleA2 {
		t.Fatalf("status = %d %s after %v, want 200 at version 2 from Read", rec.Code, rec.Body, svc.asked)
	}
	assertGolden(t, "schedule.json", rec.Body.Bytes())

	rec = getSchedule(t, &stubPublic{err: schedule.ErrNotFound}, &log, "/fhir/R4/Schedule/"+scheduleB)
	if rec.Code != http.StatusNotFound {
		t.Errorf("a schedule outside the directory = %d, want 404", rec.Code)
	}
	assertGolden(t, "not_found.json", rec.Body.Bytes())
}

func TestHandlerSearch_FindsTheScheduleOfTheActor(t *testing.T) {
	var log bytes.Buffer
	for _, actor := range []string{"PractitionerRole/" + roleA2, roleA2} {
		svc := &stubPublic{out: scheduleOfA2}

		rec := getSchedule(t, svc, &log, "/fhir/R4/Schedule?actor="+actor)

		if rec.Code != http.StatusOK || len(svc.asked) != 1 || svc.asked[0] != "actor:"+roleA2 {
			t.Fatalf("actor=%s: status = %d %s after %v, want 200 from ByActor", actor, rec.Code, rec.Body, svc.asked)
		}
		assertGolden(t, "search.json", rec.Body.Bytes())
	}
	rec := getSchedule(t, &stubPublic{err: schedule.ErrNotFound}, &log, "/fhir/R4/Schedule?actor=PractitionerRole/"+roleB)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"total":0`) {
		t.Errorf("a role outside the directory = %d %s, want an empty searchset", rec.Code, rec.Body)
	}
	assertGolden(t, "search_empty.json", rec.Body.Bytes())
}

func TestHandlerSearch_RejectsAnyOtherSearchWithoutCallingTheService(t *testing.T) {
	for _, query := range []string{"", "actor=", "actor=Practitioner/" + practitioner1, "actor=" + roleA1 + "&actor=" + roleA2,
		"actor=" + roleA1 + "," + roleA2, "actor=" + roleA1 + "&active=true", "date=ge2026-11-04T00:00:00Z"} {
		var log bytes.Buffer
		svc := &stubPublic{}

		rec := getSchedule(t, svc, &log, "/fhir/R4/Schedule?"+query)

		if rec.Code != http.StatusBadRequest || len(svc.asked) != 0 {
			t.Errorf("?%s: status = %d after %v, want 400 and no call", query, rec.Code, svc.asked)
		}
		assertGolden(t, "unsupported_search.json", rec.Body.Bytes())
	}
}

func TestHandler_SendsAFixedOperationOutcomeOnAnError(t *testing.T) {
	for _, path := range []string{"/fhir/R4/Schedule/" + scheduleA2, "/fhir/R4/Schedule?actor=" + roleA2} {
		var log bytes.Buffer
		rec := getSchedule(t, &stubPublic{err: errors.New("db down")}, &log, path)
		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") || log.Len() == 0 {
			t.Errorf("GET %s = %d %s, log %q; want 500 without the error, the error logged", path, rec.Code, rec.Body, log.String())
		}
	}
}
