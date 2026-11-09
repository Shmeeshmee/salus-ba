package questionnaire_test

import (
	"bytes"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

// docs/i18n/SPEC.md §3 "Questionnaires": a draft's language and translations travel in both directions.

const translatedBody = `{"title":"` + placeholderTitle + `","item":[],"language":"en","translations":[` +
	`{"language":"de","title":"Placeholder title in German","description":"Placeholder description in German",` +
	`"items":{"q1":{"text":"Placeholder question in German","help":"Placeholder help in German"},"q2":{"options":["Placeholder first",""]}}}]}`

var translatedIn = []questionnaire.Translation{{
	Language: "de", Title: "Placeholder title in German", Description: "Placeholder description in German",
	Items: map[string]questionnaire.ItemTranslation{
		"q1": {Text: "Placeholder question in German", Help: "Placeholder help in German"},
		"q2": {Options: []string{"Placeholder first", ""}},
	},
}}

func TestHTTPHandler_DecodesTheLanguageAndTheTranslations(t *testing.T) {
	for _, r := range []request{
		{http.MethodPost, "/questionnaires", "", translatedBody},
		{http.MethodPut, "/questionnaires/" + draftA, `W/"3"`, translatedBody},
	} {
		svc := &stubLibrary{out: savedDraft}
		if rec := serveHTTP(t, svc, true, &bytes.Buffer{}, r); rec.Code/100 != 2 {
			t.Fatalf("%s = %d %s, want it read", r.method, rec.Code, rec.Body)
		}
		if svc.in.Language != "en" || !reflect.DeepEqual(svc.in.Translations, translatedIn) {
			t.Errorf("%s passed %q %+v, want en and %+v", r.method, svc.in.Language, svc.in.Translations, translatedIn)
		}
	}

	svc := &stubLibrary{out: savedDraft}
	serveHTTP(t, svc, true, &bytes.Buffer{}, request{http.MethodPut, "/questionnaires/" + draftA, `W/"3"`, draftBody})
	if svc.in.Language != "" || svc.in.Translations != nil {
		t.Errorf("PUT without them passed %q %+v, want none (the draft's kept)", svc.in.Language, svc.in.Translations)
	}
	svc = &stubLibrary{out: savedDraft}
	serveHTTP(t, svc, true, &bytes.Buffer{}, request{http.MethodPut, "/questionnaires/" + draftA, `W/"3"`, `{"title":"x","item":[],"translations":[]}`})
	if svc.in.Translations == nil || len(svc.in.Translations) != 0 {
		t.Errorf("PUT with an empty list passed %+v, want an empty list (all removed)", svc.in.Translations)
	}
	for _, body := range []string{
		`{"title":"x","item":[],"translations":[{"language":"de","title":"","items":{},"status":"x"}]}`,
		`{"title":"x","item":[],"translations":[{"language":"de","title":"","items":{"q1":{"text":"x","code":"x"}}}]}`,
		`{"title":"x","item":[],"language":7}`,
	} {
		svc := &stubLibrary{}
		if rec := serveHTTP(t, svc, true, &bytes.Buffer{}, request{http.MethodPost, "/questionnaires", "", body}); rec.Code != http.StatusBadRequest || svc.called != "" {
			t.Errorf("POST %s = %d %s, want 400 without calling the service", body, rec.Code, rec.Body)
		}
	}
}

func TestHTTPHandler_SendsTheLanguageAndTheTranslations(t *testing.T) {
	out := savedDraft
	out.Language = "bs"
	out.Translations = []questionnaire.Translation{{
		Language: "en", Title: "Placeholder title in English",
		Items: map[string]questionnaire.ItemTranslation{"q1": {Text: "Placeholder question in English"}},
	}}
	rec := serveHTTP(t, &stubLibrary{out: out}, true, &bytes.Buffer{}, request{http.MethodPost, "/questionnaires", "", draftBody})

	want := `"translations":[{"items":{"q1":{"text":"Placeholder question in English"}},"language":"en","title":"Placeholder title in English"}]`
	if rec.Code != http.StatusCreated || !strings.Contains(rec.Body.String(), want) || !strings.Contains(rec.Body.String(), `"language":"bs"`) {
		t.Errorf("POST = %d %s, want 201 with bs and %s", rec.Code, rec.Body, want)
	}
}

func TestHTTPHandler_MapsAnIncompleteTranslation(t *testing.T) {
	for _, tt := range []struct {
		err  error
		body string
	}{
		{&questionnaire.TranslationIncompleteError{Language: "en", LinkIDs: []string{"q1", "q2"}}, `{"error":"translation_incomplete","language":"en","linkIds":["q1","q2"]}`},
		{&questionnaire.TranslationIncompleteError{Language: "sr-Cyrl"}, `{"error":"translation_incomplete","language":"sr-Cyrl","linkIds":[]}`},
	} {
		rec := serveHTTP(t, &stubLibrary{err: tt.err}, true, &bytes.Buffer{}, request{http.MethodPost, "/questionnaires/" + draftA + "/publish", `W/"3"`, ""})
		if rec.Code != http.StatusUnprocessableEntity || rec.Body.String() != tt.body+"\n" {
			t.Errorf("publish with %v = %d %s, want 422 %s", tt.err, rec.Code, rec.Body, tt.body)
		}
	}
}
