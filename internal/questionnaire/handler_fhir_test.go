package questionnaire_test

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
	"time"

	"github.com/Shmeeshmee/salus/internal/questionnaire"
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

// stubReader answers with what it is given and keeps what it was asked.
type stubReader struct {
	err    error
	caller questionnaire.Caller
	id     string
	filter questionnaire.Filter
	out    []questionnaire.Questionnaire
	calls  int
}

func (s *stubReader) Read(_ context.Context, c questionnaire.Caller, id string) (questionnaire.Questionnaire, error) {
	s.caller, s.id = c, id
	s.calls++
	if s.err != nil || len(s.out) == 0 {
		return questionnaire.Questionnaire{}, s.err
	}
	return s.out[0], nil
}

func (s *stubReader) Search(_ context.Context, c questionnaire.Caller, f questionnaire.Filter) ([]questionnaire.Questionnaire, error) {
	s.caller, s.filter = c, f
	s.calls++
	return s.out, s.err
}

func as(c questionnaire.Caller) func(context.Context) (questionnaire.Caller, bool, error) {
	return func(context.Context) (questionnaire.Caller, bool, error) { return c, true, nil }
}

func nobody(context.Context) (questionnaire.Caller, bool, error) {
	return questionnaire.Caller{}, false, nil
}

const base = "https://api.example.test/fhir/R4"

func getFHIR(t testing.TB, svc *stubReader, caller func(context.Context) (questionnaire.Caller, bool, error), log *bytes.Buffer, path string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	questionnaire.NewHandler(svc, caller, base, slog.New(slog.NewJSONHandler(log, nil))).Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
	return rec
}

// everyType is a synthetic active version (placeholder wording) with every item type, a section with help
// text and a condition hiding it, required questions, help text on questions, conditions on a boolean and on
// a single choice, and a multiple choice.
var everyType = questionnaire.Questionnaire{
	ID: activeA, SeriesID: seriesA, OrganizationID: clinicA, Version: 1, Status: questionnaire.StatusActive,
	Title: placeholderTitle, Description: "Placeholder description, not clinical content",
	PublishedAt: now, LastUpdated: now, VersionID: 4,
	Items: []questionnaire.Item{
		{LinkID: "intro", Text: "Placeholder instructions", Type: questionnaire.ItemDisplay},
		{LinkID: "smoker", Text: "Placeholder yes or no question", Type: questionnaire.ItemBoolean, Required: true, Help: "Placeholder help text"},
		{
			LinkID: "freq", Text: "Placeholder single choice", Type: questionnaire.ItemChoice,
			Options:    []questionnaire.Option{{Value: "Placeholder option A"}, {Value: "Placeholder option B"}},
			EnableWhen: &questionnaire.Condition{Question: "smoker", Boolean: yes()},
		},
		{
			LinkID: "details", Text: "Placeholder section", Type: questionnaire.ItemGroup, Help: "Placeholder section help",
			EnableWhen: &questionnaire.Condition{Question: "freq", String: "Placeholder option B"},
			Items: []questionnaire.Item{
				{LinkID: "name", Text: "Placeholder short answer", Type: questionnaire.ItemString, Required: true},
				{LinkID: "notes", Text: "Placeholder long answer", Type: questionnaire.ItemText},
				{
					LinkID: "many", Text: "Placeholder multiple choice", Type: questionnaire.ItemChoice, Repeats: true,
					Options: []questionnaire.Option{{Value: "Placeholder option C"}, {Value: "Placeholder option D"}, {Value: "Placeholder option E"}},
				},
				{LinkID: "count", Text: "Placeholder whole number", Type: questionnaire.ItemInteger},
				{LinkID: "weight", Text: "Placeholder decimal number", Type: questionnaire.ItemDecimal, Help: "Placeholder unit hint"},
				{LinkID: "since", Text: "Placeholder date", Type: questionnaire.ItemDate},
			},
		},
	},
}

// A draft: the next version of the same series, changed a day later, one question so far.
var draft = questionnaire.Questionnaire{
	ID: draftA, SeriesID: seriesA, OrganizationID: clinicA, Version: 2, Status: questionnaire.StatusDraft,
	Title: placeholderTitle, LastUpdated: now.Add(24 * time.Hour), VersionID: 1,
	Items: []questionnaire.Item{{LinkID: "smoker", Text: "Placeholder yes or no question", Type: questionnaire.ItemBoolean}},
}

// attachments is a synthetic active version with an attachment item for each combination of kinds, of 1 and
// of 10 files, one of them required, with help text and under a condition.
var attachments = questionnaire.Questionnaire{
	ID: activeA, SeriesID: seriesA, OrganizationID: clinicA, Version: 1, Status: questionnaire.StatusActive,
	Title: placeholderTitle, PublishedAt: now, LastUpdated: now, VersionID: 2,
	Items: []questionnaire.Item{
		{LinkID: "has-files", Text: "Placeholder yes or no question", Type: questionnaire.ItemBoolean},
		with(attachment("p1", 1, questionnaire.FilePhoto), func(i *questionnaire.Item) {
			i.Required, i.Help = true, "Placeholder help text"
			i.EnableWhen = &questionnaire.Condition{Question: "has-files", Boolean: yes()}
		}),
		attachment("d10", 10, questionnaire.FileDocument),
		attachment("v1", 1, questionnaire.FileVideo),
		attachment("pd10", 10, questionnaire.FilePhoto, questionnaire.FileDocument),
		attachment("pv1", 1, questionnaire.FilePhoto, questionnaire.FileVideo),
		attachment("dv10", 10, questionnaire.FileDocument, questionnaire.FileVideo),
		{LinkID: "s", Text: "Placeholder section", Type: questionnaire.ItemGroup, Items: []questionnaire.Item{
			attachment("pdv1", 1, questionnaire.FilePhoto, questionnaire.FileDocument, questionnaire.FileVideo),
			attachment("pdv10", 10, questionnaire.FileVideo, questionnaire.FilePhoto, questionnaire.FileDocument),
		}},
	},
}

// The read sends the Questionnaire, its canonical url and version, with its version as the ETag.
func TestHandlerRead_SendsTheQuestionnaire(t *testing.T) {
	for _, tt := range []struct {
		golden, etag string
		q            questionnaire.Questionnaire
	}{
		{golden: "questionnaire_active.json", etag: `W/"4"`, q: everyType},
		{golden: "questionnaire_draft.json", etag: `W/"1"`, q: draft},
		{golden: "questionnaire_attachments.json", etag: `W/"2"`, q: attachments},
	} {
		svc := &stubReader{out: []questionnaire.Questionnaire{tt.q}}
		rec := getFHIR(t, svc, as(practitionerA1), &bytes.Buffer{}, "/fhir/R4/Questionnaire/"+tt.q.ID)
		if rec.Code != http.StatusOK || rec.Header().Get("ETag") != tt.etag || rec.Header().Get("Content-Type") != "application/fhir+json" ||
			rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("status = %d %s %v, want 200 with %s", rec.Code, rec.Body, rec.Header(), tt.etag)
		}
		if svc.caller != practitionerA1 || svc.id != tt.q.ID {
			t.Errorf("Read(%+v, %q), want the caller and the id of the URL", svc.caller, svc.id)
		}
		assertGolden(t, tt.golden, rec.Body.Bytes())
	}
}

// The search passes the series of the url, the version and the status, and sends a searchset with its self
// link; no parameters search all the clinic's versions.
func TestHandlerSearch_PassesTheFilterAndSendsASearchset(t *testing.T) {
	svc := &stubReader{out: []questionnaire.Questionnaire{everyType, draft}}
	rec := getFHIR(t, svc, as(adminA), &bytes.Buffer{},
		"/fhir/R4/Questionnaire?url=https://salus.invalid/fhir/Questionnaire/"+seriesA+"&status=draft&version=2")
	want := questionnaire.Filter{SeriesID: seriesA, Status: questionnaire.StatusDraft, Version: 2}
	if rec.Code != http.StatusOK || svc.caller != adminA || svc.filter != want || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status = %d %s %v after Search(%+v, %+v), want 200, no-store and %+v", rec.Code, rec.Body, rec.Header(), svc.caller, svc.filter, want)
	}
	assertGolden(t, "search.json", rec.Body.Bytes())

	for status, want := range map[string]questionnaire.Status{"draft": questionnaire.StatusDraft, "active": questionnaire.StatusActive, "retired": questionnaire.StatusRetired} {
		svc = &stubReader{}
		if rec := getFHIR(t, svc, as(adminA), &bytes.Buffer{}, "/fhir/R4/Questionnaire?status="+status); rec.Code != http.StatusOK || svc.filter.Status != want {
			t.Errorf("status=%s = %d after %+v, want the status %s", status, rec.Code, svc.filter, want)
		}
	}
	svc = &stubReader{}
	rec = getFHIR(t, svc, as(practitionerA1), &bytes.Buffer{}, "/fhir/R4/Questionnaire")
	if rec.Code != http.StatusOK || svc.filter != (questionnaire.Filter{}) || svc.calls != 1 ||
		!strings.Contains(rec.Body.String(), `"total":0`) || !strings.Contains(rec.Body.String(), `"url":"`+base+`/Questionnaire"`) {
		t.Errorf("no parameters = %d %s after %+v, want an empty filter and searchset", rec.Code, rec.Body, svc.filter)
	}
}

func TestHandlerSearch_RejectsAParameterItCannotReadWithoutCallingTheService(t *testing.T) {
	for i, query := range []string{
		"name=x",
		"_include=Questionnaire:x",
		"status=draft&status=active",
		"status=unknown",
		"version=one",
		"version=0",
		"version=99999999999",
		"url=https://other.example.test/fhir/Questionnaire/" + seriesA,
		"url=https://salus.invalid/fhir/Questionnaire/",
		"url=https://salus.invalid/fhir/Questionnaire/" + seriesA + "|1",
		"url=https://salus.invalid/fhir/Questionnaire/" + seriesA + "/x",
		"url=https://salus.invalid/fhir/Questionnaire/not-a-uuid",
		"url=https://salus.invalid/fhir/Questionnaire/" + seriesA + "0",
		"url=https://salus.invalid/fhir/Questionnaire/" + strings.ReplaceAll(seriesA, "-", "x"),
		"status=%zz",
	} {
		svc := &stubReader{}
		rec := getFHIR(t, svc, as(adminA), &bytes.Buffer{}, "/fhir/R4/Questionnaire?"+query)
		if rec.Code != http.StatusBadRequest || svc.calls != 0 {
			t.Errorf("?%s = %d %s (service called %d times), want 400", query, rec.Code, rec.Body, svc.calls)
		}
		if i == 0 {
			assertGolden(t, "unsupported_search.json", rec.Body.Bytes())
		}
	}
}

func TestHandler_MapsTheServiceErrors(t *testing.T) {
	read, search := "/fhir/R4/Questionnaire/"+missing, "/fhir/R4/Questionnaire?status=active"
	rec := getFHIR(t, &stubReader{err: questionnaire.ErrNotFound}, as(adminB), &bytes.Buffer{}, read)
	if rec.Code != http.StatusNotFound {
		t.Errorf("read of another clinic's = %d, want 404", rec.Code)
	}
	assertGolden(t, "not_found.json", rec.Body.Bytes())

	// A patient reads no questionnaire yet (docs/questionnaire/SPEC.md §2 "FHIR": 404, as for one they may
	// not see) and may not search the library.
	if rec := getFHIR(t, &stubReader{err: questionnaire.ErrForbidden}, as(patient), &bytes.Buffer{}, read); rec.Code != http.StatusNotFound {
		t.Errorf("a patient's read = %d %s, want 404", rec.Code, rec.Body)
	} else {
		assertGolden(t, "not_found.json", rec.Body.Bytes())
	}
	rec = getFHIR(t, &stubReader{err: questionnaire.ErrForbidden}, as(patient), &bytes.Buffer{}, search)
	if rec.Code != http.StatusForbidden {
		t.Errorf("a patient's search = %d %s, want 403", rec.Code, rec.Body)
	}
	assertGolden(t, "forbidden.json", rec.Body.Bytes())

	for _, path := range []string{read, search} {
		svc := &stubReader{}
		rec := getFHIR(t, svc, nobody, &bytes.Buffer{}, path)
		if rec.Code != http.StatusUnauthorized || svc.calls != 0 {
			t.Errorf("GET %s signed out = %d (service called %d times), want 401", path, rec.Code, svc.calls)
		}
		assertGolden(t, "unauthenticated.json", rec.Body.Bytes())
	}
}

// The log holds no title, item text, option or help text: nothing on success, the error alone on a failure.
func TestHandler_LogsNoQuestionnaireContent(t *testing.T) {
	for _, path := range []string{"/fhir/R4/Questionnaire/" + activeA, "/fhir/R4/Questionnaire?status=active"} {
		var logs bytes.Buffer
		getFHIR(t, &stubReader{out: []questionnaire.Questionnaire{everyType}}, as(adminA), &logs, path)
		if logs.Len() != 0 {
			t.Errorf("GET %s logged %q on success, want nothing", path, logs.String())
		}
		rec := getFHIR(t, &stubReader{err: errors.New("synthetic failure")}, as(adminA), &logs, path)
		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "synthetic") {
			t.Errorf("GET %s with a failing service = %d %s, want 500 with a fixed message", path, rec.Code, rec.Body)
		}
		for _, s := range []string{placeholderTitle, "Placeholder"} {
			if strings.Contains(logs.String(), s) {
				t.Errorf("GET %s: the log holds %q: %s", path, s, logs.String())
			}
		}
		failing := func(context.Context) (questionnaire.Caller, bool, error) {
			return questionnaire.Caller{}, false, errors.New("synthetic caller failure")
		}
		if rec := getFHIR(t, &stubReader{}, failing, &logs, path); rec.Code != http.StatusInternalServerError {
			t.Errorf("GET %s when the caller cannot be resolved = %d, want 500", path, rec.Code)
		}
	}
}

// BenchmarkFHIRReadVersion encodes a 200-item version as FHIR JSON through the read (the service stubbed):
// with BenchmarkReadVersion, the whole read of SPEC.md §2 "Performance".
func BenchmarkFHIRReadVersion(b *testing.B) {
	q := questionnaire.Questionnaire{
		ID: activeA, SeriesID: seriesA, OrganizationID: clinicA, Version: 1, Status: questionnaire.StatusActive,
		Title: placeholderTitle, Items: large().Items, VersionID: 1, LastUpdated: now,
	}
	svc := &stubReader{out: []questionnaire.Questionnaire{q}}
	for b.Loop() {
		if rec := getFHIR(b, svc, as(adminA), &bytes.Buffer{}, "/fhir/R4/Questionnaire/"+activeA); rec.Code != http.StatusOK {
			b.Fatal(rec.Code)
		}
	}
}
