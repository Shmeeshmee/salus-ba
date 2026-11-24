package patient_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/patient"
)

func (s *stubClinic) Correct(_ context.Context, c patient.Caller, in patient.Patient) (patient.Patient, error) {
	s.caller, s.in = c, in
	if len(s.out) == 0 {
		return patient.Patient{}, s.err
	}
	return s.out[0], s.err
}

func (s *stubClinic) WithdrawConsent(_ context.Context, c patient.Caller, id string) (consent.Consent, error) {
	s.caller, s.asked = c, id
	return consent.Consent{}, s.err
}

func (s *stubClinic) ClinicRecord(_ context.Context, c patient.Caller, id string) (patient.ClinicRecord, error) {
	s.caller, s.asked = c, id
	return s.record, s.err
}

const correctBody = `{"givenName":"Cy","familyName":"Example","birthDate":"1980-03-04","phone":"+38761000002"}`

// serveClinicMatch serves the request as the admin, with the If-Match header when it is not empty.
func serveClinicMatch(t *testing.T, svc *stubClinic, method, target, body, ifMatch string) *httptest.ResponseRecorder {
	t.Helper()
	h := patient.NewClinicHandler(svc, func(context.Context) (patient.Caller, bool, error) { return admin, true, nil },
		slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	mux := http.NewServeMux()
	h.Routes(mux)
	req := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestClinicHandler_CorrectsARegisteredProfileAtTheVersionItNames(t *testing.T) {
	t.Parallel()
	svc := &stubClinic{out: []patient.Patient{registeredC}}
	rec := serveClinicMatch(t, svc, http.MethodPut, "/clinic/patients/"+registeredID, correctBody, `W/"1"`)
	if rec.Code != http.StatusOK || rec.Body.String() != registeredJSON+"\n" || rec.Header().Get("ETag") != `W/"1"` || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("PUT = %d %s (ETag %s), want 200 %s", rec.Code, rec.Body, rec.Header().Get("ETag"), registeredJSON)
	}
	want := patient.Patient{ID: registeredID, VersionID: 1, GivenName: "Cy", FamilyName: "Example", BirthDate: date(1980, 3, 4), Phone: "+38761000002"}
	if svc.in != want || svc.caller != admin {
		t.Errorf("corrected %+v as %+v, want %+v", svc.in, svc.caller, want)
	}
}

func TestClinicHandler_RecordsTheWithdrawal(t *testing.T) {
	t.Parallel()
	svc := &stubClinic{}
	rec := serveClinicMatch(t, svc, http.MethodPost, "/clinic/patients/"+registeredID+"/consent-withdrawal", "", "")
	if rec.Code != http.StatusNoContent || svc.asked != registeredID || svc.caller != admin {
		t.Errorf("POST = %d %s for %q, want 204 for the patient", rec.Code, rec.Body, svc.asked)
	}
}

func TestClinicHandler_SendsTheRecordAsAFHIRCollectionToSave(t *testing.T) {
	t.Parallel()
	svc := &stubClinic{record: patient.ClinicRecord{Patient: registeredC, Consents: []consent.Consent{{
		ID: "f4000000-0000-4000-8000-0000000000f4", PatientID: registeredID, OrganizationID: clinicA, Kind: consent.KindCare,
		Status: consent.StatusInactive, DateTime: date(2026, 11, 1), RevokedAt: now, LastUpdated: now, VersionID: 2, GivenInPerson: true,
	}}}}
	rec := serveClinicMatch(t, svc, http.MethodGet, "/clinic/patients/"+registeredID+"/record", "", "")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/fhir+json" || rec.Header().Get("Cache-Control") != "no-store" ||
		rec.Header().Get("Content-Disposition") != `attachment; filename="patient-`+registeredID+`.json"` {
		t.Fatalf("GET = %d %v, want 200 a FHIR attachment named by the id", rec.Code, rec.Header())
	}
	if svc.asked != registeredID {
		t.Errorf("asked for %q", svc.asked)
	}
	assertGolden(t, "clinic_record.json", rec.Body.Bytes())
}

func TestClinicHandler_AnswersEachRightsRefusalWithItsCode(t *testing.T) {
	t.Parallel()
	base := "/clinic/patients/" + registeredID
	for name, tc := range map[string]struct {
		err     error
		method  string
		target  string
		body    string
		ifMatch string
		code    string
		status  int
	}{
		"no If-Match":       {method: http.MethodPut, target: base, body: correctBody, status: 400, code: "if_match_required"},
		"a stale version":   {err: patient.ErrConflict, method: http.MethodPut, target: base, body: correctBody, ifMatch: `W/"1"`, status: 409, code: "version_conflict"},
		"not the clinic's":  {err: patient.ErrNotFound, method: http.MethodPut, target: base, body: correctBody, ifMatch: `W/"1"`, status: 404, code: "not_found"},
		"a bad birth date":  {method: http.MethodPut, target: base, body: strings.Replace(correctBody, "1980-03-04", "x", 1), ifMatch: `W/"1"`, status: 400, code: "invalid_birth_date"},
		"an unknown field":  {method: http.MethodPut, target: base, body: strings.Replace(correctBody, "}", `,"email":"x"}`, 1), ifMatch: `W/"1"`, status: 400, code: "invalid_request"},
		"no active consent": {err: consent.ErrInactive, method: http.MethodPost, target: base + "/consent-withdrawal", status: 409, code: "consent_inactive"},
		"withdraw refused":  {err: patient.ErrForbidden, method: http.MethodPost, target: base + "/consent-withdrawal", status: 403, code: "forbidden"},
		"record refused":    {err: patient.ErrForbidden, method: http.MethodGet, target: base + "/record", status: 403, code: "forbidden"},
		"record not found":  {err: patient.ErrNotFound, method: http.MethodGet, target: base + "/record", status: 404, code: "not_found"},
		"an internal error": {err: errors.New("boom Cy Example"), method: http.MethodGet, target: base + "/record", status: 500},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec := serveClinicMatch(t, &stubClinic{err: tc.err}, tc.method, tc.target, tc.body, tc.ifMatch)
			if rec.Code != tc.status || (tc.code != "" && rec.Body.String() != `{"error":"`+tc.code+`"}`+"\n") {
				t.Errorf("= %d %s, want %d %s", rec.Code, rec.Body, tc.status, tc.code)
			}
			if strings.Contains(rec.Body.String(), "Cy") {
				t.Errorf("the answer carries a name: %s", rec.Body)
			}
		})
	}
}
