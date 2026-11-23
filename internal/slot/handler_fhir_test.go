package slot_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
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

// stubSearch answers with what it is given and keeps what it was asked.
type stubSearch struct {
	err           error
	from, to      time.Time
	schedule, org string
	out           []slot.Slot
	calls         int
}

func (s *stubSearch) SearchFor(_ context.Context, org, scheduleID string, from, to time.Time) ([]slot.Slot, error) {
	s.org, s.schedule, s.from, s.to = org, scheduleID, from, to
	s.calls++
	return s.out, s.err
}

func searchSlots(t *testing.T, svc *stubSearch, log *bytes.Buffer, query string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	slot.NewHandler(svc, staffClinic, "https://api.example.test/fhir/R4", slog.New(slog.NewJSONHandler(log, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/fhir/R4/Slot?"+query, nil))
	return rec
}

// Anyone, signed in or not, finds the free slots of a schedule of the public directory in a range
// (SPEC.md "FHIR reads and searches").
func TestHandlerSearch_SendsTheFreeSlotsOfTheScheduleInTheRange(t *testing.T) {
	start := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	free := slot.Slot{ID: slotA1, ScheduleID: scheduleA1, OrganizationID: clinicA, Status: slot.StatusFree,
		Period: slot.Period{Start: start, End: start.Add(30 * time.Minute), Mode: schedule.ModeInPerson}, VersionID: 1, LastUpdated: now}
	for _, query := range []string{
		"schedule=Schedule/" + scheduleA1 + "&status=free&start=ge2026-11-05T00:00:00Z&start=lt2026-11-06T00:00:00Z",
		"schedule=" + scheduleA1 + "&start=" + url.QueryEscape("ge2026-11-05T01:00:00+01:00") + "&status=free&start=lt2026-11-06T00:00:00Z",
	} {
		var log bytes.Buffer
		svc := &stubSearch{out: []slot.Slot{free}}

		rec := searchSlots(t, svc, &log, query)

		if rec.Code != http.StatusOK || svc.schedule != scheduleA1 || !svc.from.Equal(start.Add(-8*time.Hour)) || !svc.to.Equal(start.Add(16*time.Hour)) {
			t.Fatalf("?%s: status = %d %s after Search(%q, %v, %v)", query, rec.Code, rec.Body, svc.schedule, svc.from, svc.to)
		}
		assertGolden(t, "search.json", rec.Body.Bytes())
	}
	var log bytes.Buffer
	svc := &stubSearch{}
	rec := searchSlots(t, svc, &log, "schedule="+scheduleA1+"&status=free&start=lt2026-11-06T00:00:00Z")
	if rec.Code != http.StatusOK || !svc.from.IsZero() || !strings.Contains(rec.Body.String(), `"total":0`) {
		t.Errorf("without a start: status = %d %s from %v, want an empty searchset from now (the zero time)", rec.Code, rec.Body, svc.from)
	}
}

func TestHandlerSearch_RejectsAnyOtherSearchWithoutCallingTheService(t *testing.T) {
	const lt = "&start=lt2026-11-06T00:00:00Z"
	for _, query := range []string{
		"", "status=free" + lt, "schedule=" + scheduleA1 + lt, "schedule=" + scheduleA1 + "&status=free",
		"schedule=" + scheduleA1 + "&status=busy" + lt, "schedule=" + scheduleA1 + "&status=free&status=free" + lt,
		"schedule=Practitioner/" + scheduleA1 + "&status=free" + lt, "schedule=" + scheduleA1 + "&schedule=" + scheduleA1 + "&status=free" + lt,
		"schedule=" + scheduleA1 + "&status=free&start=gt2026-11-05T00:00:00Z" + lt, "schedule=" + scheduleA1 + "&status=free&start=lt2026-11-06",
		"schedule=" + scheduleA1 + "&status=free&_count=5" + lt,
	} {
		var log bytes.Buffer
		svc := &stubSearch{}

		rec := searchSlots(t, svc, &log, query)

		if rec.Code != http.StatusBadRequest || svc.calls != 0 {
			t.Errorf("?%s: status = %d after %d calls, want 400 and none", query, rec.Code, svc.calls)
		}
		assertGolden(t, "unsupported_search.json", rec.Body.Bytes())
	}
}

func TestHandlerSearch_SendsAFixedOperationOutcomeOnAnError(t *testing.T) {
	var log bytes.Buffer
	rec := searchSlots(t, &stubSearch{err: errors.New("db down")}, &log, "schedule="+scheduleA1+"&status=free&start=lt2026-11-06T00:00:00Z")
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "db down") || log.Len() == 0 {
		t.Errorf("status = %d %s, log %q; want 500 without the error, the error logged", rec.Code, rec.Body, log.String())
	}
}

// clinicKey marks a request as the signed-in staff's of a clinic, for staffClinic.
type clinicKey struct{}

// staffClinic is the handler's clinic: the one the request's context names, empty for anyone else.
func staffClinic(ctx context.Context) string {
	org, _ := ctx.Value(clinicKey{}).(string)
	return org
}

// The clinic's own staff search its free times in its own scope (the front desk, docs/clinic-cms F3): the handler
// passes the signed-in staff's clinic to the service, and none for anyone else.
func TestHandler_PassesTheStaffsClinic(t *testing.T) {
	query := "schedule=Schedule/" + scheduleA1 + "&status=free&start=lt2026-11-06T00:00:00Z"
	for want, ctx := range map[string]context.Context{clinicA: context.WithValue(t.Context(), clinicKey{}, clinicA), "": t.Context()} {
		svc := &stubSearch{}
		mux := http.NewServeMux()
		slot.NewHandler(svc, staffClinic, "https://api.example.test/fhir/R4", slog.New(slog.DiscardHandler)).Routes(mux)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, http.MethodGet, "/fhir/R4/Slot?"+query, nil))
		if rec.Code != http.StatusOK || svc.org != want {
			t.Errorf("Slot search = %d with clinic %q, want 200 with %q", rec.Code, svc.org, want)
		}
	}
}
