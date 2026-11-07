package questionnaire_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/Shmeeshmee/salus/internal/questionnaire"
)

// docs/i18n/SPEC.md §3 "FHIR": Questionnaire.language is the original language; each translated text carries
// the R4 core translation extension (lang, content) on its primitive element.

// translated is everyType as a draft written in Bosnian, completely translated into English and partly into
// German (placeholder wording).
var translated = func() questionnaire.Questionnaire {
	q := everyType
	q.ID, q.Version, q.Status, q.VersionID = draftA, 2, questionnaire.StatusDraft, 3
	q.Translations = []questionnaire.Translation{
		{
			Language: "en", Title: "Placeholder title in English", Description: "Placeholder description in English",
			Items: map[string]questionnaire.ItemTranslation{
				"intro":   {Text: "Placeholder instructions in English"},
				"smoker":  {Text: "Placeholder yes or no question in English", Help: "Placeholder help text in English"},
				"freq":    {Text: "Placeholder single choice in English", Options: []string{"Placeholder option A in English", "Placeholder option B in English"}},
				"details": {Text: "Placeholder section in English", Help: "Placeholder section help in English"},
				"name":    {Text: "Placeholder short answer in English"},
				"notes":   {Text: "Placeholder long answer in English"},
				"many": {Text: "Placeholder multiple choice in English", Options: []string{
					"Placeholder option C in English", "Placeholder option D in English", "Placeholder option E in English",
				}},
				"count":  {Text: "Placeholder whole number in English"},
				"weight": {Text: "Placeholder decimal number in English", Help: "Placeholder unit hint in English"},
				"since":  {Text: "Placeholder date in English"},
			},
		},
		{
			Language: "de", Title: "Placeholder title in German",
			Items: map[string]questionnaire.ItemTranslation{
				"smoker": {Text: "Placeholder yes or no question in German"},
				"freq":   {Options: []string{"", "Placeholder option B in German"}},
			},
		},
	}
	return q
}()

func TestHandlerRead_SendsTheLanguageAndTheTranslations(t *testing.T) {
	svc := &stubReader{out: []questionnaire.Questionnaire{translated}}
	rec := getFHIR(t, svc, as(practitionerA1), &bytes.Buffer{}, "/fhir/R4/Questionnaire/"+draftA)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	assertGolden(t, "questionnaire_translated.json", rec.Body.Bytes())
}
