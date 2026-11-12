package composition_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/composition"
)

// stubDocuments answers every call with its fields and records the last call's arguments.
type stubDocuments struct {
	err       error
	templates []composition.Template
	pdf       []byte
	terms     []composition.SectionTerms
	got       struct {
		id      string
		content composition.TemplateContent
		gen     composition.Generate
		values  []composition.Value
		version int32
	}
	draft composition.Draft
}

func (s *stubDocuments) Terms(_ context.Context, _ appointment.Caller, id string) ([]composition.SectionTerms, error) {
	s.got.id = id
	return s.terms, s.err
}

func (s *stubDocuments) Templates(context.Context, appointment.Caller) ([]composition.Template, error) {
	return s.templates, s.err
}

func (s *stubDocuments) CreateTemplate(_ context.Context, _ appointment.Caller, in composition.TemplateContent) (composition.Template, error) {
	s.got.content = in
	return s.template(), s.err
}

func (s *stubDocuments) UpdateTemplate(_ context.Context, _ appointment.Caller, id string, in composition.TemplateContent, v int32) (composition.Template, error) {
	s.got.id, s.got.content, s.got.version = id, in, v
	return s.template(), s.err
}

func (s *stubDocuments) Generate(_ context.Context, _ appointment.Caller, id string, in composition.Generate) (composition.Draft, error) {
	s.got.id, s.got.gen = id, in
	return s.draft, s.err
}

func (s *stubDocuments) Drafts(_ context.Context, _ appointment.Caller, id string) ([]composition.Draft, error) {
	s.got.id = id
	return []composition.Draft{s.draft}, s.err
}

func (s *stubDocuments) Sign(_ context.Context, _ appointment.Caller, id string, values []composition.Value, v int32) (composition.Draft, error) {
	s.got.id, s.got.values, s.got.version = id, values, v
	return s.draft, s.err
}

func (s *stubDocuments) PDF(_ context.Context, _ appointment.Caller, id string) (io.ReadCloser, int64, error) {
	s.got.id = id
	return io.NopCloser(bytes.NewReader(s.pdf)), int64(len(s.pdf)), s.err
}

func (s *stubDocuments) template() composition.Template {
	if len(s.templates) == 0 {
		return composition.Template{}
	}
	return s.templates[0]
}

const tplID = "70000000-0000-4000-8000-000000000001"

func stubbed() *stubDocuments {
	c := content()
	return &stubDocuments{
		templates: []composition.Template{{
			ID: tplID, OrganizationID: clinicA, Title: c.Title, Language: c.Language, Status: c.Status, TypeCode: c.TypeCode,
			Fields: c.Fields, Translations: c.Translations, VersionID: 2, LastUpdated: now,
		}},
		draft: composition.Draft{
			ID: "d0000000-0000-4000-8000-000000000001", EncounterID: encA, TemplateID: tplID, Status: composition.StatusPreliminary, VersionID: 3,
			Fields: []composition.DraftField{{Field: composition.Field{Key: "findings", Value: "Placeholder", SourceSegmentIDs: []string{seg1}}}},
		},
		pdf: []byte("%PDF-1.4 placeholder"),
	}
}

func serve(t *testing.T, svc *stubDocuments, signedIn bool, method, target, body string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	h := composition.NewHTTPHandler(svc, func(context.Context) (appointment.Caller, bool, error) { return pract, signedIn, nil }, slog.New(slog.DiscardHandler))
	mux := http.NewServeMux()
	h.Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	for k, v := range header {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestHTTP_ListsTheTemplates(t *testing.T) {
	t.Parallel()
	w := serve(t, stubbed(), true, "GET", "/document-templates", "", nil)
	var got struct {
		Templates []struct {
			ID, Title, Language, Status, TypeCode, LastUpdated string
			Fields                                             []composition.TemplateField
			Translations                                       []struct {
				Fields   map[string]composition.FieldTexts
				Language string
				Title    string
			}
			VersionID int32
		}
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &got) != nil || len(got.Templates) != 1 {
		t.Fatalf("GET = %d %s", w.Code, w.Body)
	}
	tpl := got.Templates[0]
	if tpl.ID != tplID || tpl.TypeCode != "11488-4" || tpl.Status != "draft" || tpl.VersionID != 2 || tpl.LastUpdated != "2030-01-07T09:30:00Z" ||
		len(tpl.Fields) != 4 || !tpl.Fields[0].Required || tpl.Translations[0].Language != "en" || tpl.Translations[0].Fields["tests"].Label == "" {
		t.Errorf("template = %+v", tpl)
	}
	if !strings.Contains(w.Body.String(), `"typeCode":"11488-4"`) || w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("body %s", w.Body)
	}
}

func TestHTTP_WritesATemplate(t *testing.T) {
	t.Parallel()
	body := `{"title":"T","language":"bs","status":"active","typeCode":"57133-1",
		"fields":[{"key":"a","type":"date","label":"L","instruction":"I","required":true}],
		"translations":[{"language":"en","title":"T","fields":{"a":{"label":"L","instruction":"I"}}}]}`
	svc := stubbed()
	w := serve(t, svc, true, "POST", "/document-templates", body, nil)
	c := svc.got.content
	if w.Code != 201 || w.Header().Get("ETag") != `W/"2"` || c.Title != "T" || c.Status != composition.TemplateActive || c.TypeCode != "57133-1" ||
		len(c.Fields) != 1 || c.Fields[0].Type != composition.FieldDate || !c.Fields[0].Required || c.Translations[0].Fields["a"].Instruction != "I" {
		t.Errorf("POST = %d %s, content %+v", w.Code, w.Body, c)
	}
	w = serve(t, svc, true, "PUT", "/document-templates/"+tplID, body, map[string]string{"If-Match": `W/"2"`})
	if w.Code != 200 || svc.got.id != tplID || svc.got.version != 2 {
		t.Errorf("PUT = %d %s, got %+v", w.Code, w.Body, svc.got)
	}
	for name, tt := range map[string]struct {
		header map[string]string
		method string
		body   string
	}{
		"no If-Match":     {nil, "PUT", body},
		"a broken body":   {nil, "POST", "{"},
		"an unknown key":  {nil, "POST", `{"title":"T","other":1}`},
		"trailing tokens": {nil, "POST", body + "{}"},
	} {
		target := "/document-templates"
		if tt.method == "PUT" {
			target += "/" + tplID
		}
		if w := serve(t, svc, true, tt.method, target, tt.body, tt.header); w.Code != 400 || !strings.Contains(w.Body.String(), "invalid_request") {
			t.Errorf("%s: %d %s, want 400 invalid_request", name, w.Code, w.Body)
		}
	}
}

func TestHTTP_DraftsAndSigns(t *testing.T) {
	t.Parallel()
	svc := stubbed()
	w := serve(t, svc, true, "POST", "/encounters/"+encA+"/drafts", `{"templateId":"`+tplID+`","language":"en"}`, nil)
	if w.Code != 200 || svc.got.id != encA || svc.got.gen != (composition.Generate{TemplateID: tplID, Language: "en"}) ||
		!strings.Contains(w.Body.String(), `"draft_id":"d0000000-0000-4000-8000-000000000001"`) || !strings.Contains(w.Body.String(), `"version_id":3`) ||
		w.Header().Get("ETag") != `W/"3"` {
		t.Errorf("POST drafts = %d %s", w.Code, w.Body)
	}
	w = serve(t, svc, true, "GET", "/encounters/"+encA+"/drafts", "", nil)
	if w.Code != 200 || !strings.HasPrefix(w.Body.String(), `{"drafts":[{"draft_id"`) {
		t.Errorf("GET drafts = %d %s", w.Code, w.Body)
	}
	w = serve(t, svc, true, "POST", "/drafts/"+svc.draft.ID+"/sign", `{"fields":[{"key":"findings","value":"x"}]}`, map[string]string{"If-Match": `W/"3"`})
	if w.Code != 200 || svc.got.id != svc.draft.ID || svc.got.version != 3 || len(svc.got.values) != 1 || svc.got.values[0] != (composition.Value{Key: "findings", Value: "x"}) {
		t.Errorf("sign = %d %s, got %+v", w.Code, w.Body, svc.got)
	}
	w = serve(t, svc, true, "POST", "/drafts/"+svc.draft.ID+"/sign", `{"fields":[{"key":"diagnosis","value":"J06.9","confirmed":true}]}`, map[string]string{"If-Match": `W/"3"`})
	if w.Code != 200 || len(svc.got.values) != 1 || svc.got.values[0] != (composition.Value{Key: "diagnosis", Value: "J06.9", Confirmed: true}) {
		t.Errorf("sign with a confirmed diagnosis = %d %s, got %+v", w.Code, w.Body, svc.got)
	}
	if w := serve(t, svc, true, "POST", "/drafts/"+svc.draft.ID+"/sign", `{"fields":[]}`, nil); w.Code != 400 {
		t.Errorf("sign without If-Match = %d, want 400", w.Code)
	}
	if w := serve(t, svc, true, "POST", "/encounters/"+encA+"/drafts", `{"templateId":1}`, nil); w.Code != 400 {
		t.Errorf("drafts with a broken body = %d, want 400", w.Code)
	}
}

func TestHTTP_ServesThePDF(t *testing.T) {
	t.Parallel()
	svc := stubbed()
	w := serve(t, svc, true, "GET", "/document-references/r1/content", "", nil)
	if w.Code != 200 || w.Header().Get("Content-Type") != "application/pdf" || w.Header().Get("Content-Length") != "20" ||
		w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" ||
		!strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment") || w.Body.String() != "%PDF-1.4 placeholder" || svc.got.id != "r1" {
		t.Errorf("GET content = %d %v %q", w.Code, w.Header(), w.Body)
	}
}

// Every refusal is a fixed code; the field keys come with a 422 on fields; a failure is a fixed 500.
func TestHTTP_MapsTheErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		err    error
		body   string
		status int
	}{
		{composition.ErrNotFound, `{"error":"not_found"}`, 404},
		{composition.ErrForbidden, `{"error":"forbidden"}`, 403},
		{composition.ErrConsentRequired, `{"error":"consent_required"}`, 403},
		{composition.ErrTranscriptionConsentRequired, `{"error":"transcription_consent_required"}`, 403},
		{composition.ErrTemplateNotFound, `{"error":"template_not_found"}`, 404},
		{composition.ErrLanguageUnavailable, `{"error":"language_unavailable"}`, 422},
		{composition.ErrVersionMismatch, `{"error":"version_mismatch"}`, 412},
		{composition.ErrNotPreliminary, `{"error":"not_preliminary"}`, 409},
		{composition.ErrDraftingUnavailable, `{"error":"drafting_unavailable"}`, 503},
		{&composition.InvalidTemplateError{Keys: []string{"a"}}, `{"error":"invalid_template","keys":["a"]}`, 422},
		{&composition.InvalidTemplateError{}, `{"error":"invalid_template","keys":[]}`, 422},
		{&composition.FieldsError{Err: composition.ErrInvalidValues, Keys: []string{"b"}}, `{"error":"invalid_values","keys":["b"]}`, 422},
		{&composition.FieldsError{Err: composition.ErrRequiredMissing, Keys: []string{"c"}}, `{"error":"required_missing","keys":["c"]}`, 422},
		{&composition.FieldsError{Err: composition.ErrDiagnosisUnconfirmed, Keys: []string{"d"}}, `{"error":"diagnosis_unconfirmed","keys":["d"]}`, 422},
		{errBackend, `{"error":"internal error"}`, 500},
	} {
		svc := stubbed()
		svc.err = tt.err
		w := serve(t, svc, true, "POST", "/encounters/"+encA+"/drafts", `{"templateId":"`+tplID+`"}`, nil)
		if w.Code != tt.status || strings.TrimSpace(w.Body.String()) != tt.body {
			t.Errorf("%v: %d %s, want %d %s", tt.err, w.Code, w.Body, tt.status, tt.body)
		}
	}
	for _, target := range []string{"/document-templates", "/encounters/" + encA + "/drafts", "/document-references/r1/content"} {
		if w := serve(t, stubbed(), false, "GET", target, "", nil); w.Code != 401 {
			t.Errorf("%s signed out = %d, want 401", target, w.Code)
		}
	}
	svc := stubbed()
	svc.err = errors.New("down")
	if w := serve(t, svc, true, "GET", "/document-references/r1/content", "", nil); w.Code != 500 {
		t.Errorf("content down = %d, want 500", w.Code)
	}
}
