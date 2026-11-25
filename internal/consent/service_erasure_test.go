package consent_test

import (
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/consent"
)

// The erasure job reads at which clinics the patient has a record: every clinic of a consent, active or not, in the
// patient's scope, audited as the job's (docs/clinic-cms/WAVE5.md S6).
func TestClinicsOf_AreTheClinicsOfEveryConsentAuditedAsTheErasureJob(t *testing.T) {
	f := newFixture()
	got, err := f.svc.ClinicsOf(t.Context(), patientA)
	if err != nil {
		t.Fatalf("ClinicsOf() error = %v", err)
	}
	if want := []string{clinicA, clinicB}; !slices.Equal(got, want) {
		t.Errorf("ClinicsOf() = %v, want %v", got, want)
	}
	if f.repo.patient != patientA {
		t.Errorf("scope %q, want the patient's", f.repo.patient)
	}
	if len(f.audit.events) != 1 || f.audit.events[0].AgentName != consent.ErasureAgent || f.audit.events[0].AgentAccountID != "" ||
		f.audit.events[0].EntityName != "consent" || f.audit.events[0].EntityCount != 2 {
		t.Errorf("audit events %+v, want one search by the erasure job over 2 consents", f.audit.events)
	}
}
