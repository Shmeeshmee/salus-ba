package organization_test

import (
	"net/http"
	"testing"

	"github.com/Shmeeshmee/salus/internal/organization"
)

const defaultLanguageURL = "https://salus.invalid/fhir/StructureDefinition/organization-default-language"

// The clinic settings screen reads and sets the clinic's default language as the organization-default-language
// extension (docs/i18n/PROGRESS.md, 0.1 "Clinic settings are FHIR").
func TestHandlerRead_SendsTheDefaultLanguageExtension(t *testing.T) {
	o := clinic(3)
	o.DefaultLanguage = "sr-Cyrl"
	s := newOrgServer(&stubClinics{org: o})

	rec := s.do(t, http.MethodGet, clinicPath, "", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	assertGolden(t, "organization_default_language.json", rec.Body.Bytes())
}

func TestHandlerUpdate_SetsTheDefaultLanguageFromItsExtension(t *testing.T) {
	updated := clinic(4)
	updated.DefaultLanguage = "hr"
	for name, tc := range map[string]struct {
		extensions string
		want       string // "" for no change asked
	}{
		"language":          {`[{"url":"` + defaultLanguageURL + `","valueCode":"hr"}]`, "hr"},
		"with the others":   {`[{"url":"` + listedURL + `","valueBoolean":true},{"url":"` + timeZoneURL + `","valueCode":"Europe/Sarajevo"},{"url":"` + defaultLanguageURL + `","valueCode":"hr"}]`, "hr"},
		"left out":          {`[{"url":"` + listedURL + `","valueBoolean":true}]`, ""},
		"not a staff tag":   {`[{"url":"` + defaultLanguageURL + `","valueCode":"de"}]`, "de"}, // the service refuses it
		"another extension": {`[{"url":"https://elsewhere.test/x","valueCode":"y"}]`, ""},
	} {
		s := newOrgServer(&stubClinics{org: updated})
		body := `{"resourceType":"Organization","id":"` + clinicA + `","name":"Synthetic clinic","extension":` + tc.extensions + `}`

		rec := s.do(t, http.MethodPut, clinicPath, `W/"3"`, body)

		got := s.svc.changes.DefaultLanguage
		if rec.Code != http.StatusOK || (got == nil) != (tc.want == "") || (got != nil && *got != tc.want) {
			t.Fatalf("%s: status = %d %s with default language %v, want 200 and %q", name, rec.Code, rec.Body, got, tc.want)
		}
		if name == "language" {
			assertGolden(t, "organization_default_language_updated.json", rec.Body.Bytes())
		}
	}
}

func TestHandlerUpdate_RejectsADefaultLanguageWithoutItsCodeBeforeCallingTheService(t *testing.T) {
	for _, extensions := range []string{
		`[{"url":"` + defaultLanguageURL + `"}]`,
		`[{"url":"` + defaultLanguageURL + `","valueCode":""}]`,
		`[{"url":"` + defaultLanguageURL + `","valueBoolean":true}]`,
		`[{"url":"` + defaultLanguageURL + `","valueCode":"bs"},{"url":"` + defaultLanguageURL + `","valueCode":"hr"}]`,
	} {
		s := newOrgServer(&stubClinics{})
		body := `{"resourceType":"Organization","id":"` + clinicA + `","name":"Synthetic clinic","extension":` + extensions + `}`

		rec := s.do(t, http.MethodPut, clinicPath, `W/"3"`, body)

		if rec.Code != http.StatusBadRequest || s.svc.calls != 0 {
			t.Errorf("extension %s: status = %d after %d service calls, want 400 and none", extensions, rec.Code, s.svc.calls)
		}
		assertGolden(t, "invalid_resource.json", rec.Body.Bytes())
	}
}

func TestHandlerUpdate_MapsAnInvalidDefaultLanguage(t *testing.T) {
	s := newOrgServer(&stubClinics{err: organization.ErrInvalidDefaultLanguage})

	rec := s.do(t, http.MethodPut, clinicPath, `W/"3"`, renameBody(clinicA))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	assertGolden(t, "invalid_default_language.json", rec.Body.Bytes())
}
