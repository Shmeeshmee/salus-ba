package fhir_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

func TestWriteError_SendsOperationOutcomeWithOneErrorIssue(t *testing.T) {
	rec := httptest.NewRecorder()

	fhir.WriteError(rec, http.StatusNotFound, "not-found", "No such resource.")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/fhir+json" {
		t.Errorf("Content-Type = %q, want application/fhir+json", ct)
	}
	want := `{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"not-found","diagnostics":"No such resource."}]}` + "\n"
	if body := rec.Body.String(); body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestWriteError_OmitsEmptyDiagnostics(t *testing.T) {
	rec := httptest.NewRecorder()

	fhir.WriteError(rec, http.StatusBadRequest, "invalid", "")

	want := `{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"invalid"}]}` + "\n"
	if body := rec.Body.String(); body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestETag_IsAWeakTagOfTheVersion(t *testing.T) {
	if got := fhir.ETag("3"); got != `W/"3"` {
		t.Errorf(`ETag("3") = %q, want W/"3"`, got)
	}
}

func TestIfMatch_ReadsTheVersionOfAWeakOrStrongTag(t *testing.T) {
	for header, want := range map[string]string{`W/"3"`: "3", `"12"`: "12", ` W/"7" `: "7"} {
		got, ok := fhir.IfMatch(header)
		if !ok || got != want {
			t.Errorf("IfMatch(%q) = %q, %v, want %q, true", header, got, ok, want)
		}
	}
	for _, header := range []string{"", "3", `W/3`, `W/""`, `"3`, `*`} {
		if got, ok := fhir.IfMatch(header); ok {
			t.Errorf("IfMatch(%q) = %q, true, want false", header, got)
		}
	}
}

func TestNewMeta_GivesTheVersionAsTextAndTheTimeInUTC(t *testing.T) {
	at := time.Date(2026, 11, 4, 11, 30, 0, 123456000, time.FixedZone("CET", 3600))

	m := fhir.NewMeta(4, at)

	got, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"lastUpdated":"2026-11-04T10:30:00.123456Z","versionId":"4"}`; string(got) != want {
		t.Errorf("meta = %s, want %s", got, want)
	}
}

func TestNewSearchset_CountsTheMatchesAndMarksEachEntrysMode(t *testing.T) {
	base := "https://api.example.test/fhir/R4"
	b := fhir.NewSearchset(base+"/A?x=1",
		[]fhir.BundleEntry{{FullURL: base + "/A/1", Resource: map[string]string{"resourceType": "A"}}},
		[]fhir.BundleEntry{{FullURL: base + "/B/2", Resource: map[string]string{"resourceType": "B"}}})

	got, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"resourceType":"Bundle","type":"searchset","link":[{"relation":"self","url":"https://api.example.test/fhir/R4/A?x=1"}],"entry":[` +
		`{"fullUrl":"https://api.example.test/fhir/R4/A/1","resource":{"resourceType":"A"},"search":{"mode":"match"}},` +
		`{"fullUrl":"https://api.example.test/fhir/R4/B/2","resource":{"resourceType":"B"},"search":{"mode":"include"}}],"total":1}`
	if string(got) != want {
		t.Errorf("bundle = %s\nwant     %s", got, want)
	}
}

func TestNewSearchset_HasATotalOfZeroAndNoEntriesForNoMatch(t *testing.T) {
	got, err := json.Marshal(fhir.NewSearchset("https://api.example.test/fhir/R4/A", nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"resourceType":"Bundle","type":"searchset","link":[{"relation":"self","url":"https://api.example.test/fhir/R4/A"}],"total":0}`
	if string(got) != want {
		t.Errorf("bundle = %s, want %s", got, want)
	}
}

func TestInternalError_LogsTheErrorAndSendsAFixedOperationOutcome(t *testing.T) {
	var logs bytes.Buffer
	rec := httptest.NewRecorder()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/fhir/R4/Practitioner/x", nil)

	fhir.InternalError(rec, r, slog.New(slog.NewJSONHandler(&logs, nil)), errors.New("dial tcp 10.0.0.7:5432: refused"))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	want := `{"resourceType":"OperationOutcome","issue":[{"severity":"error","code":"exception","diagnostics":"Internal error."}]}` + "\n"
	if rec.Body.String() != want {
		t.Errorf("body = %q, want %q", rec.Body, want)
	}
	if !strings.Contains(logs.String(), "10.0.0.7") {
		t.Errorf("the log does not hold the error: %s", logs.String())
	}
}

func TestWrite_SendsURLsWithoutHTMLEscaping(t *testing.T) {
	rec := httptest.NewRecorder()

	fhir.Write(rec, http.StatusOK, map[string]string{"url": "https://api.example.test/fhir/R4/A?x=1&y=<2>"})

	if want := `{"url":"https://api.example.test/fhir/R4/A?x=1&y=<2>"}` + "\n"; rec.Body.String() != want {
		t.Errorf("body = %q, want %q", rec.Body, want)
	}
}

func TestTypes_EncodeTheElementsTheySetOnly(t *testing.T) {
	yes := true
	for name, tc := range map[string]struct {
		v    any
		want string
	}{
		"a reference":                {fhir.Reference{Reference: "Patient/1"}, `{"reference":"Patient/1"}`},
		"a reference with a display": {fhir.Reference{Reference: "Patient/1", Display: "Ana Example"}, `{"reference":"Patient/1","display":"Ana Example"}`},
		"a codeable concept": {
			fhir.CodeableConcept{Coding: []fhir.Coding{{System: "http://snomed.info/sct", Code: "394579002", Display: "Cardiology"}}, Text: "Cardiology"},
			`{"text":"Cardiology","coding":[{"system":"http://snomed.info/sct","code":"394579002","display":"Cardiology"}]}`,
		},
		"a coding without display": {fhir.Coding{System: "s", Code: "c"}, `{"system":"s","code":"c"}`},
		"a boolean extension":      {fhir.Extension{URL: "u", ValueBoolean: &yes}, `{"valueBoolean":true,"url":"u"}`},
		"a code extension":         {fhir.Extension{URL: "u", ValueCode: "Europe/Sarajevo"}, `{"url":"u","valueCode":"Europe/Sarajevo"}`},
	} {
		got, err := json.Marshal(tc.v)
		if err != nil || string(got) != tc.want {
			t.Errorf("%s = %s (%v), want %s", name, got, err, tc.want)
		}
	}
}

func TestReferenceID_TakesTheIDOfAReferenceOrABareID(t *testing.T) {
	for _, tc := range []struct {
		value, typ, want string
		ok               bool
	}{
		{"Patient/abc", "Patient", "abc", true},
		{"abc", "Patient", "abc", true},
		{"Practitioner/abc", "Patient", "", false},
		{"Patient/", "Patient", "", false},
		{"", "Patient", "", false},
		{"Patient/a/b", "Patient", "", false},
		{"a,b", "Patient", "", false},
		{"https://elsewhere.test/fhir/Patient/abc", "Patient", "", false},
	} {
		got, ok := fhir.ReferenceID(tc.value, tc.typ)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ReferenceID(%q, %q) = %q, %v; want %q, %v", tc.value, tc.typ, got, ok, tc.want, tc.ok)
		}
	}
}

func TestDateRange_ReadsOneGeAndOneLtInstant(t *testing.T) {
	from := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		from, to time.Time
		name     string
		values   []string
		ok       bool
	}{
		{from, to, "both", []string{"ge2026-11-02T01:00:00+01:00", "lt2026-11-09T00:00:00Z"}, true},
		{from, time.Time{}, "only ge", []string{"ge2026-11-02T00:00:00Z"}, true},
		{time.Time{}, to, "only lt", []string{"lt2026-11-09T00:00:00Z"}, true},
		{time.Time{}, time.Time{}, "none", nil, true},
		{time.Time{}, time.Time{}, "two ge", []string{"ge2026-11-02T00:00:00Z", "ge2026-11-03T00:00:00Z"}, false},
		{time.Time{}, time.Time{}, "another prefix", []string{"gt2026-11-02T00:00:00Z"}, false},
		{time.Time{}, time.Time{}, "no prefix", []string{"2026-11-02T00:00:00Z"}, false},
		{time.Time{}, time.Time{}, "a date without a zone", []string{"ge2026-11-02"}, false},
		{time.Time{}, time.Time{}, "not a date", []string{"lttomorrow"}, false},
	} {
		gotFrom, gotTo, ok := fhir.DateRange(tc.values)
		if !gotFrom.Equal(tc.from) || !gotTo.Equal(tc.to) || ok != tc.ok {
			t.Errorf("DateRange(%s) = %v, %v, %v; want %v, %v, %v", tc.name, gotFrom, gotTo, ok, tc.from, tc.to, tc.ok)
		}
	}
}

func TestIsUUID_AcceptsOnlyTheHyphenatedHexadecimalForm(t *testing.T) {
	t.Parallel()
	for s, want := range map[string]bool{
		"0b6c3a52-6a0c-4c52-9e1b-6f1c2a3b4c5d": true,
		"0B6C3A52-6A0C-4C52-9E1B-6F1C2A3B4C5D": true,
		"":                                     false,
		"0b6c3a526a0c4c529e1b6f1c2a3b4c5d":     false,
		"0b6c3a52-6a0c-4c52-9e1b-6f1c2a3b4c5":  false,
		"0b6c3a52-6a0c-4c52-9e1b-6f1c2a3b4c5g": false,
		"0b6c3a52_6a0c-4c52-9e1b-6f1c2a3b4c5d": false,
	} {
		if got := fhir.IsUUID(s); got != want {
			t.Errorf("IsUUID(%q) = %v, want %v", s, got, want)
		}
	}
}
