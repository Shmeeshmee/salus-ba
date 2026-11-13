package allergyintolerance

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shmeeshmee/salus/pkg/fhir"
)

const provenanceID = "9e000000-0000-4000-8000-0000000000b1"

// stubProvenances answers the Provenance read with what it is given.
type stubProvenances struct {
	stubReader
	err   error
	p     Provenance
	found bool
}

func (s *stubProvenances) Provenance(_ context.Context, c Caller, id string) (Provenance, bool, error) {
	s.caller, s.id = c, id
	return s.p, s.found, s.err
}

// provenanceGet serves GET /fhir/R4/Provenance/{id} by the handler's source and reports whether it answered.
func provenanceGet(t *testing.T, svc *stubProvenances, signedIn bool) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	caller := func(context.Context) (Caller, bool, error) {
		return Caller{AccountID: "x", PatientID: patientA}, signedIn, nil
	}
	h := NewFHIRHandler(svc, caller, "https://api.example.test/fhir/R4", slog.New(slog.DiscardHandler))
	var answered bool
	mux := http.NewServeMux()
	mux.HandleFunc("GET /fhir/R4/Provenance/{id}", func(w http.ResponseWriter, r *http.Request) { answered = h.Provenance(w, r) })
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/fhir/R4/Provenance/"+provenanceID, nil))
	return rec, answered
}

// A clinic's confirmation as an R4 Provenance: the version it produced, the verifier on behalf of the clinic, the
// version it revised; written once.
func TestFHIRHandler_Provenance(t *testing.T) {
	svc := &stubProvenances{found: true, p: Provenance{Recorded: recorded, ID: provenanceID, EntryID: allergyID, RoleID: roleA,
		OrganizationID: clinicA, Activity: fhir.ActivityUpdate, AgentType: fhir.AgentVerifier, Version: 2}}
	rec, answered := provenanceGet(t, svc, true)
	if !answered || rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"1"` || rec.Header().Get("Content-Type") != "application/fhir+json" ||
		svc.id != provenanceID || svc.caller.PatientID != patientA {
		t.Fatalf("read: answered %v, %d %v, asked %q", answered, rec.Code, rec.Header(), svc.id)
	}
	assertBody(t, "provenance.json", rec.Body.Bytes())
	// The patient's addition of a clinic's entry to their profile: the patient the agent, on no one's behalf. (A
	// removal's Provenance is read by no one: nobody sees an entered-in-error entry.)
	svc.p.Activity, svc.p.AgentType, svc.p.RoleID, svc.p.OrganizationID, svc.p.PatientID = fhir.ActivityUpdate, fhir.AgentAuthor, "", "", patientA
	rec, _ = provenanceGet(t, svc, true)
	assertBody(t, "provenance_patient.json", rec.Body.Bytes())
	// Recording an entry confirms it with the recorder's role: one agent its author and its verifier, no revision.
	svc.p = Provenance{Recorded: svc.p.Recorded, ID: provenanceID, EntryID: svc.p.EntryID, RoleID: roleA, OrganizationID: clinicA,
		Activity: fhir.ActivityCreate, AgentType: fhir.AgentAuthor, Version: 1}
	rec, _ = provenanceGet(t, svc, true)
	assertBody(t, "provenance_recorded.json", rec.Body.Bytes())
}

// An id that is no allergy's Provenance is left to the route's next source, nothing written; a refusal is a 404, an
// internal error a 500, nobody signed in a 401.
func TestFHIRHandler_ProvenanceAnswersOnlyItsOwn(t *testing.T) {
	if rec, answered := provenanceGet(t, &stubProvenances{}, true); answered || rec.Body.Len() != 0 || len(rec.Header()) != 0 {
		t.Errorf("not its own: answered %v, wrote %d %q", answered, rec.Code, rec.Body)
	}
	for name, tt := range map[string]struct {
		svc      *stubProvenances
		signedIn bool
		want     int
	}{
		"refused":        {&stubProvenances{found: true, err: ErrNotFound}, true, http.StatusNotFound},
		"internal error": {&stubProvenances{found: true, err: errors.New("x")}, true, http.StatusInternalServerError},
		"nobody":         {&stubProvenances{}, false, http.StatusUnauthorized},
	} {
		if rec, answered := provenanceGet(t, tt.svc, tt.signedIn); !answered || rec.Code != tt.want {
			t.Errorf("%s: answered %v, %d; want %d", name, answered, rec.Code, tt.want)
		}
	}
}
