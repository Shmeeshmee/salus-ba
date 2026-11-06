package questionnaire_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

// stubLibrary answers every call with out (or def) and err, and keeps what it was asked.
type stubLibrary struct {
	err     error
	called  string
	caller  questionnaire.Caller
	id      string
	other   string // the questionnaire of SetDefault
	in      questionnaire.Content
	def     questionnaire.Default
	out     questionnaire.Questionnaire
	version int32
}

func (s *stubLibrary) Create(_ context.Context, c questionnaire.Caller, in questionnaire.Content) (questionnaire.Questionnaire, error) {
	s.called, s.caller, s.in = "Create", c, in
	return s.out, s.err
}

func (s *stubLibrary) UpdateDraft(_ context.Context, c questionnaire.Caller, id string, in questionnaire.Content, version int32) (questionnaire.Questionnaire, error) {
	s.called, s.caller, s.id, s.in, s.version = "UpdateDraft", c, id, in, version
	return s.out, s.err
}

func (s *stubLibrary) DeleteDraft(_ context.Context, c questionnaire.Caller, id string) error {
	s.called, s.caller, s.id = "DeleteDraft", c, id
	return s.err
}

func (s *stubLibrary) Publish(_ context.Context, c questionnaire.Caller, id string, version int32) (questionnaire.Questionnaire, error) {
	s.called, s.caller, s.id, s.version = "Publish", c, id, version
	return s.out, s.err
}

func (s *stubLibrary) Retire(_ context.Context, c questionnaire.Caller, id string) (questionnaire.Questionnaire, error) {
	s.called, s.caller, s.id = "Retire", c, id
	return s.out, s.err
}

func (s *stubLibrary) Revise(_ context.Context, c questionnaire.Caller, id string) (questionnaire.Questionnaire, error) {
	s.called, s.caller, s.id = "Revise", c, id
	return s.out, s.err
}

func (s *stubLibrary) Default(_ context.Context, c questionnaire.Caller, roleID string) (questionnaire.Default, error) {
	s.called, s.caller, s.id = "Default", c, roleID
	return s.def, s.err
}

func (s *stubLibrary) SetDefault(_ context.Context, c questionnaire.Caller, roleID, questionnaireID string) (questionnaire.Default, error) {
	s.called, s.caller, s.id, s.other = "SetDefault", c, roleID, questionnaireID
	return s.def, s.err
}

func (s *stubLibrary) ClearDefault(_ context.Context, c questionnaire.Caller, roleID string) error {
	s.called, s.caller, s.id = "ClearDefault", c, roleID
	return s.err
}

// request is one JSON API call.
type request struct {
	method, path, ifMatch, body string
}

// serveHTTP sends r as practitionerA1, or as nobody when signedIn is false, logging to log.
func serveHTTP(t *testing.T, svc *stubLibrary, signedIn bool, log *bytes.Buffer, r request) *httptest.ResponseRecorder {
	t.Helper()
	h := questionnaire.NewHTTPHandler(svc, func(context.Context) (questionnaire.Caller, bool, error) { return practitionerA1, signedIn, nil },
		slog.New(slog.NewTextHandler(log, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), r.method, r.path, strings.NewReader(r.body))
	if r.ifMatch != "" {
		req.Header.Set("If-Match", r.ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// A synthetic draft: a section with a yes or no question and a single choice shown after "yes", with help
// text. Placeholder wording.
const draftBody = `{"title":"` + placeholderTitle + `","description":"Placeholder description","item":[` +
	`{"linkId":"s1","text":"Placeholder section","type":"group","item":[` +
	`{"linkId":"q1","text":"` + placeholderText + `","type":"boolean","required":true,"help":"Placeholder help text"},` +
	`{"linkId":"q2","text":"Placeholder choice","type":"choice","repeats":true,"answerOption":[{"valueString":"Placeholder option A"},{"valueString":"Placeholder option B"}],` +
	`"enableWhen":{"question":"q1","answerBoolean":true}}]}]}`

var (
	draftContent = questionnaire.Content{Title: placeholderTitle, Description: "Placeholder description", Items: []questionnaire.Item{{
		LinkID: "s1", Text: "Placeholder section", Type: questionnaire.ItemGroup, Items: []questionnaire.Item{
			{LinkID: "q1", Text: placeholderText, Type: questionnaire.ItemBoolean, Required: true, Help: "Placeholder help text"},
			{
				LinkID: "q2", Text: "Placeholder choice", Type: questionnaire.ItemChoice, Repeats: true,
				Options:    []questionnaire.Option{{Value: "Placeholder option A"}, {Value: "Placeholder option B"}},
				EnableWhen: &questionnaire.Condition{Question: "q1", Boolean: yes()},
			},
		},
	}}}
	savedDraft = questionnaire.Questionnaire{
		ID: draftA, SeriesID: seriesA, OrganizationID: clinicA, Version: 2, Status: questionnaire.StatusDraft, Title: draftContent.Title,
		Description: draftContent.Description, Items: draftContent.Items, VersionID: 3, LastUpdated: now,
	}
)

const savedDraftJSON = `{"id":"` + draftA + `","url":"https://salus.invalid/fhir/Questionnaire/` + seriesA + `","status":"draft",` +
	`"title":"` + placeholderTitle + `","description":"Placeholder description","lastUpdated":"2026-11-06T09:30:00Z","item":[` +
	`{"linkId":"s1","text":"Placeholder section","type":"group","item":[` +
	`{"linkId":"q1","text":"` + placeholderText + `","type":"boolean","help":"Placeholder help text","required":true},` +
	`{"enableWhen":{"answerBoolean":true,"question":"q1"},"linkId":"q2","text":"Placeholder choice","type":"choice",` +
	`"answerOption":[{"valueString":"Placeholder option A"},{"valueString":"Placeholder option B"}],"repeats":true}]}],` +
	`"version":2,"versionId":3}` + "\n"

func sameContent(a, b questionnaire.Content) bool {
	return a.Title == b.Title && a.Description == b.Description && sameItems(a.Items, b.Items)
}

func sameItems(a, b []questionnaire.Item) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if x.LinkID != y.LinkID || x.Text != y.Text || x.Type != y.Type || x.Help != y.Help || x.Required != y.Required ||
			x.Repeats != y.Repeats || len(x.Options) != len(y.Options) || !sameItems(x.Items, y.Items) || (x.EnableWhen == nil) != (y.EnableWhen == nil) {
			return false
		}
		for j := range x.Options {
			if x.Options[j] != y.Options[j] {
				return false
			}
		}
		if x.EnableWhen != nil && (x.EnableWhen.Question != y.EnableWhen.Question || x.EnableWhen.String != y.EnableWhen.String ||
			(x.EnableWhen.Boolean == nil) != (y.EnableWhen.Boolean == nil) || (x.EnableWhen.Boolean != nil && *x.EnableWhen.Boolean != *y.EnableWhen.Boolean)) {
			return false
		}
	}
	return true
}

func TestHTTPHandler_CreatesADraftAndSendsItWithItsVersion(t *testing.T) {
	svc := &stubLibrary{out: savedDraft}
	rec := serveHTTP(t, svc, true, &bytes.Buffer{}, request{http.MethodPost, "/questionnaires", "", draftBody})

	if rec.Code != http.StatusCreated || rec.Body.String() != savedDraftJSON || rec.Header().Get("ETag") != `W/"3"` {
		t.Errorf("POST = %d %s ETag %q, want 201 %s ETag W/\"3\"", rec.Code, rec.Body, rec.Header().Get("ETag"), savedDraftJSON)
	}
	if svc.called != "Create" || svc.caller != practitionerA1 || !sameContent(svc.in, draftContent) {
		t.Errorf("%s(%+v, %+v), want Create of the body for the caller", svc.called, svc.caller, svc.in)
	}
	if rec := serveHTTP(t, &stubLibrary{out: savedDraft}, true, &bytes.Buffer{}, request{http.MethodPost, "/questionnaires", "", `{"title":"x","item":[]}`}); rec.Code != http.StatusCreated {
		t.Errorf("POST without a description = %d %s, want 201", rec.Code, rec.Body)
	}
}

func TestHTTPHandler_CallsTheServiceForEachAction(t *testing.T) {
	q := "/questionnaires/" + draftA
	tests := []struct {
		r       request
		called  string
		content bool  // the body is the draft's content
		version int32 // from If-Match
		status  int
	}{
		{request{http.MethodPut, q, `W/"3"`, draftBody}, "UpdateDraft", true, 3, http.StatusOK},
		{request{http.MethodPost, q + "/publish", `W/"3"`, ""}, "Publish", false, 3, http.StatusOK},
		{request{http.MethodPost, q + "/retire", "", ""}, "Retire", false, 0, http.StatusOK},
		{request{http.MethodPost, q + "/revise", "", ""}, "Revise", false, 0, http.StatusCreated},
	}
	for _, tt := range tests {
		svc := &stubLibrary{out: savedDraft}
		rec := serveHTTP(t, svc, true, &bytes.Buffer{}, tt.r)
		if rec.Code != tt.status || rec.Body.String() != savedDraftJSON || rec.Header().Get("ETag") != `W/"3"` {
			t.Errorf("%s %s = %d %s, want %d with the questionnaire", tt.r.method, tt.r.path, rec.Code, rec.Body, tt.status)
		}
		if svc.called != tt.called || svc.caller != practitionerA1 || svc.id != draftA || svc.version != tt.version ||
			(tt.content && !sameContent(svc.in, draftContent)) {
			t.Errorf("%s %s called %s(%+v, %s, %+v, %d), want %s", tt.r.method, tt.r.path, svc.called, svc.caller, svc.id, svc.in, svc.version, tt.called)
		}
	}

	svc := &stubLibrary{}
	if rec := serveHTTP(t, svc, true, &bytes.Buffer{}, request{http.MethodDelete, q, "", ""}); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 ||
		svc.called != "DeleteDraft" || svc.id != draftA {
		t.Errorf("DELETE = %d %s after %s(%s), want 204 after DeleteDraft", rec.Code, rec.Body, svc.called, svc.id)
	}
}

func TestHTTPHandler_ReadsSetsAndClearsARolesDefault(t *testing.T) {
	path := "/practitioner-roles/" + roleA1 + "/questionnaire"
	set := questionnaire.Default{PractitionerRoleID: roleA1, OrganizationID: clinicA, SeriesID: seriesA, VersionID: 2, LastUpdated: now}
	want := `{"practitionerRoleId":"` + roleA1 + `","url":"https://salus.invalid/fhir/Questionnaire/` + seriesA + `","versionId":2}` + "\n"

	svc := &stubLibrary{def: set}
	if rec := serveHTTP(t, svc, true, &bytes.Buffer{}, request{http.MethodGet, path, "", ""}); rec.Code != http.StatusOK || rec.Body.String() != want ||
		svc.called != "Default" || svc.id != roleA1 || svc.caller != practitionerA1 {
		t.Errorf("GET = %d %s after %s(%s), want 200 %s", rec.Code, rec.Body, svc.called, svc.id, want)
	}
	none := `{"practitionerRoleId":"` + roleA1 + `","versionId":0}` + "\n"
	if rec := serveHTTP(t, &stubLibrary{def: questionnaire.Default{PractitionerRoleID: roleA1}}, true, &bytes.Buffer{}, request{http.MethodGet, path, "", ""}); rec.Body.String() != none {
		t.Errorf("GET of a role without one = %s, want %s", rec.Body, none)
	}
	svc = &stubLibrary{def: set}
	if rec := serveHTTP(t, svc, true, &bytes.Buffer{}, request{http.MethodPut, path, "", `{"questionnaireId":"` + activeA + `"}`}); rec.Code != http.StatusOK ||
		rec.Body.String() != want || svc.called != "SetDefault" || svc.id != roleA1 || svc.other != activeA {
		t.Errorf("PUT = %d %s after %s(%s, %s), want 200 %s", rec.Code, rec.Body, svc.called, svc.id, svc.other, want)
	}
	svc = &stubLibrary{}
	if rec := serveHTTP(t, svc, true, &bytes.Buffer{}, request{http.MethodDelete, path, "", ""}); rec.Code != http.StatusNoContent || svc.called != "ClearDefault" || svc.id != roleA1 {
		t.Errorf("DELETE = %d %s after %s(%s), want 204 after ClearDefault", rec.Code, rec.Body, svc.called, svc.id)
	}
}

// pad is body followed by spaces up to n bytes.
func pad(body string, n int) string { return body + strings.Repeat(" ", n-len(body)) }

func TestHTTPHandler_ReadsABodyOfUpTo256KiB(t *testing.T) {
	for _, r := range []request{
		{http.MethodPost, "/questionnaires", "", pad(draftBody, 256<<10)},
		{http.MethodPut, "/questionnaires/" + draftA, `W/"3"`, pad(draftBody, 256<<10)},
	} {
		svc := &stubLibrary{out: savedDraft}
		if rec := serveHTTP(t, svc, true, &bytes.Buffer{}, r); rec.Code/100 != 2 || svc.called == "" {
			t.Errorf("%s of 256 KiB = %d %s, want it read", r.method, rec.Code, rec.Body)
		}
		r.body += " "
		svc = &stubLibrary{out: savedDraft}
		if rec := serveHTTP(t, svc, true, &bytes.Buffer{}, r); rec.Code != http.StatusBadRequest || svc.called != "" {
			t.Errorf("%s of 256 KiB and a byte = %d %s (service called %q), want 400", r.method, rec.Code, rec.Body, svc.called)
		}
	}
}

func TestHTTPHandler_RefusesARequestItCannotReadBeforeCallingTheService(t *testing.T) {
	q := "/questionnaires/" + draftA
	def := "/practitioner-roles/" + roleA1 + "/questionnaire"
	tests := []struct {
		name string
		r    request
	}{
		{"no body", request{http.MethodPost, "/questionnaires", "", ""}},
		{"no title", request{http.MethodPost, "/questionnaires", "", `{"item":[]}`}},
		{"no items", request{http.MethodPost, "/questionnaires", "", `{"title":"x"}`}},
		{"an unknown field", request{http.MethodPost, "/questionnaires", "", `{"title":"x","item":[],"status":"active"}`}},
		{"an unknown item field", request{http.MethodPost, "/questionnaires", "", `{"title":"x","item":[{"linkId":"a","text":"x","type":"string","code":"x"}]}`}},
		{"an unknown condition field", request{http.MethodPost, "/questionnaires", "", `{"title":"x","item":[{"linkId":"a","text":"x","type":"string","enableWhen":{"question":"b","operator":"!="}}]}`}},
		{"two objects", request{http.MethodPost, "/questionnaires", "", draftBody + draftBody}},
		{"an update without If-Match", request{http.MethodPut, q, "", draftBody}},
		{"an If-Match that is not a version", request{http.MethodPut, q, `W/"three"`, draftBody}},
		{"a publish without If-Match", request{http.MethodPost, q + "/publish", "", ""}},
		{"a default without a questionnaire", request{http.MethodPut, def, "", `{}`}},
		{"a default with an unknown field", request{http.MethodPut, def, "", `{"questionnaireId":"` + activeA + `","seriesId":"x"}`}},
	}
	for _, tt := range tests {
		svc := &stubLibrary{}
		rec := serveHTTP(t, svc, true, &bytes.Buffer{}, tt.r)
		if rec.Code != http.StatusBadRequest || rec.Body.String() != `{"error":"invalid_request"}`+"\n" || svc.called != "" {
			t.Errorf("%s: %s = %d %s (service called %q), want 400 invalid_request", tt.name, tt.r.method, rec.Code, rec.Body, svc.called)
		}
	}
}

// every is one request per route, each readable.
func every() []request {
	q := "/questionnaires/" + draftA
	def := "/practitioner-roles/" + roleA1 + "/questionnaire"
	return []request{
		{http.MethodPost, "/questionnaires", "", draftBody},
		{http.MethodPut, q, `W/"3"`, draftBody},
		{http.MethodDelete, q, "", ""},
		{http.MethodPost, q + "/publish", `W/"3"`, ""},
		{http.MethodPost, q + "/retire", "", ""},
		{http.MethodPost, q + "/revise", "", ""},
		{http.MethodGet, def, "", ""},
		{http.MethodPut, def, "", `{"questionnaireId":"` + activeA + `"}`},
		{http.MethodDelete, def, "", ""},
	}
}

func TestHTTPHandler_MapsTheServiceErrors(t *testing.T) {
	tests := []struct {
		err    error
		body   string
		status int
	}{
		{questionnaire.ErrForbidden, `{"error":"forbidden"}`, http.StatusForbidden},
		{questionnaire.ErrNotFound, `{"error":"not_found"}`, http.StatusNotFound},
		{questionnaire.ErrConflict, `{"error":"version_conflict"}`, http.StatusPreconditionFailed},
		{questionnaire.ErrNotDraft, `{"error":"not_draft"}`, http.StatusConflict},
		{questionnaire.ErrNotActive, `{"error":"not_active"}`, http.StatusConflict},
		{questionnaire.ErrDraftExists, `{"error":"draft_exists"}`, http.StatusConflict},
		{&questionnaire.InvalidError{LinkIDs: []string{"q1", "q2"}}, `{"error":"invalid_questionnaire","linkIds":["q1","q2"]}`, http.StatusUnprocessableEntity},
		{&questionnaire.InvalidError{}, `{"error":"invalid_questionnaire","linkIds":[]}`, http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		for _, r := range every() {
			rec := serveHTTP(t, &stubLibrary{err: tt.err}, true, &bytes.Buffer{}, r)
			if rec.Code != tt.status || rec.Body.String() != tt.body+"\n" {
				t.Errorf("%s %s with %v = %d %s, want %d %s", r.method, r.path, tt.err, rec.Code, rec.Body, tt.status, tt.body)
			}
		}
	}
}

func TestHTTPHandler_AsksForASignInWithoutCallingTheService(t *testing.T) {
	for _, r := range every() {
		svc := &stubLibrary{}
		rec := serveHTTP(t, svc, false, &bytes.Buffer{}, r)
		if rec.Code != http.StatusUnauthorized || rec.Body.String() != `{"error":"unauthenticated"}`+"\n" || svc.called != "" {
			t.Errorf("%s %s signed out = %d %s (service called %q), want 401 unauthenticated", r.method, r.path, rec.Code, rec.Body, svc.called)
		}
	}
}

// The log holds no title, item text, option or help text: nothing on success, and the error alone on a
// failure, with a fixed message to the client.
func TestHTTPHandler_LogsNoQuestionnaireContent(t *testing.T) {
	for _, r := range every() {
		var logs bytes.Buffer
		serveHTTP(t, &stubLibrary{out: savedDraft}, true, &logs, r)
		if logs.Len() != 0 {
			t.Errorf("%s %s logged %q on success, want nothing", r.method, r.path, logs.String())
		}
		rec := serveHTTP(t, &stubLibrary{err: errors.New("synthetic failure")}, true, &logs, r)
		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "synthetic") {
			t.Errorf("%s %s with a failing service = %d %s, want 500 with a fixed message", r.method, r.path, rec.Code, rec.Body)
		}
		for _, s := range []string{placeholderTitle, placeholderText, "Placeholder option", "Placeholder help"} {
			if strings.Contains(logs.String(), s) {
				t.Errorf("%s %s: the log holds %q: %s", r.method, r.path, s, logs.String())
			}
		}
		if !strings.Contains(logs.String(), "synthetic failure") {
			t.Errorf("%s %s: log = %q, want the error", r.method, r.path, logs.String())
		}
	}

	h := questionnaire.NewHTTPHandler(&stubLibrary{}, func(context.Context) (questionnaire.Caller, bool, error) {
		return questionnaire.Caller{}, false, errors.New("synthetic caller failure")
	}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/questionnaires", strings.NewReader(draftBody)))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("POST when the caller cannot be resolved = %d, want 500", rec.Code)
	}
}
