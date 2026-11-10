package organization_test

import (
	"net/http"
	"testing"
)

const transcriptCopyURL = "https://salus.invalid/fhir/StructureDefinition/organization-patient-transcript-copy"

// SAL-17: the clinic settings screen reads and sets whether the patients get a copy of their transcripts as the
// organization-patient-transcript-copy extension.
func TestHandlerRead_SendsThePatientTranscriptCopyExtension(t *testing.T) {
	o := clinic(3)
	o.PatientTranscriptCopy = true
	rec := newOrgServer(&stubClinics{org: o}).do(t, http.MethodGet, clinicPath, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", rec.Code, rec.Body)
	}
	assertGolden(t, "organization_transcript_copy.json", rec.Body.Bytes())
}

func TestHandlerUpdate_SetsThePatientTranscriptCopyFromItsExtension(t *testing.T) {
	for name, tc := range map[string]struct {
		want       *bool
		extensions string
	}{
		"on":       {new(true), `[{"url":"` + transcriptCopyURL + `","valueBoolean":true}]`},
		"off":      {new(false), `[{"url":"` + transcriptCopyURL + `","valueBoolean":false}]`},
		"left out": {nil, `[{"url":"` + listedURL + `","valueBoolean":true}]`},
	} {
		s := newOrgServer(&stubClinics{org: clinic(4)})
		body := `{"resourceType":"Organization","id":"` + clinicA + `","name":"Synthetic clinic","extension":` + tc.extensions + `}`
		rec := s.do(t, http.MethodPut, clinicPath, `W/"3"`, body)
		got := s.svc.changes.PatientTranscriptCopy
		if rec.Code != http.StatusOK || (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
			t.Errorf("%s: status = %d %s with %v, want 200 and %v", name, rec.Code, rec.Body, got, tc.want)
		}
	}
	for _, extensions := range []string{
		`[{"url":"` + transcriptCopyURL + `"}]`,
		`[{"url":"` + transcriptCopyURL + `","valueCode":"true"}]`,
		`[{"url":"` + transcriptCopyURL + `","valueBoolean":true},{"url":"` + transcriptCopyURL + `","valueBoolean":false}]`,
	} {
		s := newOrgServer(&stubClinics{})
		body := `{"resourceType":"Organization","id":"` + clinicA + `","name":"Synthetic clinic","extension":` + extensions + `}`
		if rec := s.do(t, http.MethodPut, clinicPath, `W/"3"`, body); rec.Code != http.StatusBadRequest || s.svc.calls != 0 {
			t.Errorf("extension %s: status = %d after %d calls, want 400 and none", extensions, rec.Code, s.svc.calls)
		}
	}
}
