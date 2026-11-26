package consent_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/consent"
)

// A patient keeps the directory's reads of a clinic they have a care consent with, active or revoked, once its
// membership stops (docs/clinic-cms/PROGRESS.md, "Readiness follow-ups: suspended clinics"): the clinics are read in
// the patient's scope, care consents only, unaudited as State is.
func TestCareClinics_AreTheClinicsOfTheirCareConsentsActiveOrRevoked(t *testing.T) {
	f := newFixture()
	f.repo.consents = append(f.repo.consents,
		consent.Consent{ID: "c9000000-0000-4000-8000-0000000000c9", PatientID: patientA, OrganizationID: "c0000000-0000-4000-8000-00000000000c",
			Kind: consent.KindHistory, Status: consent.StatusActive, DateTime: granted, LastUpdated: granted, VersionID: 1})
	got, err := f.svc.CareClinics(t.Context(), patientA)
	if want := []string{clinicA, clinicB}; err != nil || !slices.Equal(got, want) {
		t.Errorf("CareClinics() = %v, %v; want %v (a history consent is not a care one)", got, err, want)
	}
	if f.repo.patient != patientA || !slices.Contains(f.repo.calls, "SetPatient") || slices.Contains(f.repo.calls, "outside:ByPatient") {
		t.Errorf("scope %q, calls %v; want the patient's, in a transaction", f.repo.patient, f.repo.calls)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audit events %+v, want none: the access check of a public read", f.audit.events)
	}
	if got, err := f.svc.CareClinics(t.Context(), ""); err != nil || len(got) != 0 {
		t.Errorf("CareClinics(no patient) = %v, %v; want none", got, err)
	}
	f = newFixture()
	f.repo.err = errors.New("connection reset")
	if _, err := f.svc.CareClinics(t.Context(), patientA); err == nil {
		t.Error("CareClinics() with a failing read = nil, want the error")
	}
}
