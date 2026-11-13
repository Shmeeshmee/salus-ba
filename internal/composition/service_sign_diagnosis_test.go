package composition_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/pkg/fhir"
)

// withDiagnosis is values() with the coded field filled, confirmed or not.
func withDiagnosis(code string, confirmed bool) []composition.Value {
	v := values()
	v[0] = composition.Value{Key: "diagnosis", Value: code, Confirmed: confirmed}
	return v
}

// A confirmed code becomes a confirmed Condition of the encounter's patient and clinic, recorded at the signing
// by the signing role, from the document's field, in the signing transaction, and its creation is audited with
// the signature (docs/plans/analytics.md §3).
func TestSign_RecordsAConfirmedConditionPerCodedField(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.drafted(t)

	got, err := f.svc.Sign(t.Context(), pract, d.ID, withDiagnosis("j06.9", true), d.VersionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.conditions.recorded) != 1 {
		t.Fatalf("recorded %+v, want one Condition", f.conditions.recorded)
	}
	c := f.conditions.recorded[0]
	want := condition.Condition{
		ID: c.ID, RecordedAt: now, OrganizationID: clinicA, PatientID: patientA, EncounterID: encA, DocumentID: d.ID,
		PractitionerRoleID: roleA1, FieldKey: "diagnosis", Code: "J06.9", VerificationStatus: condition.VerificationConfirmed,
	}
	if !fhir.IsUUID(c.ID) || c != want {
		t.Errorf("recorded %+v, want %+v", c, want)
	}
	if codes := f.codes(); !slices.Contains(codes, "create 0 Condition/"+c.ID) || len(codes) != 4 {
		t.Errorf("audit = %q, want the Condition's creation beside the signature's three", codes)
	}
	if got.Status != composition.StatusFinal {
		t.Errorf("status %s", got.Status)
	}
	if want := []string{clinicA + " " + now.UTC().Format(time.RFC3339)}; !slices.Equal(f.jobs.refreshes, want) {
		t.Errorf("refreshes %v, want the clinic's statistics of the signing's day %v", f.jobs.refreshes, want)
	}
}

// A filled coded field the practitioner did not confirm refuses the signature, audited, naming the field: nothing
// is signed or recorded. An empty coded field needs no confirmation and records nothing.
func TestSign_RefusesAnUnconfirmedDiagnosis(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.drafted(t)

	_, err := f.svc.Sign(t.Context(), pract, d.ID, withDiagnosis("J06.9", false), d.VersionID)
	var fe *composition.FieldsError
	if !errors.Is(err, composition.ErrDiagnosisUnconfirmed) || !errors.As(err, &fe) || !slices.Equal(fe.Keys, []string{"diagnosis"}) {
		t.Fatalf("Sign() = %v, want ErrDiagnosisUnconfirmed for diagnosis", err)
	}
	if len(f.store.objects) != 0 || len(f.conditions.recorded) != 0 || f.repo.drafts[d.ID].Status != composition.StatusPreliminary ||
		len(f.audit.events) != 1 || f.audit.events[0].Outcome == 0 {
		t.Errorf("%d objects, %d Conditions, status %s, audit %q", len(f.store.objects), len(f.conditions.recorded), f.repo.drafts[d.ID].Status, f.codes())
	}

	if _, err := f.svc.Sign(t.Context(), pract, d.ID, values(), d.VersionID); err != nil || len(f.conditions.recorded) != 0 {
		t.Errorf("an empty diagnosis: Sign() = %v with %d Conditions, want a signature and none", err, len(f.conditions.recorded))
	}
	if len(f.jobs.refreshes) != 0 {
		t.Errorf("refreshes %v without a diagnosis, want none", f.jobs.refreshes)
	}
}

// When the Conditions cannot be recorded the signature is rolled back and its PDF removed.
func TestSign_AFailingConditionSignsNothing(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	d := f.drafted(t)
	f.conditions.err = errors.New("down")

	if _, err := f.svc.Sign(t.Context(), pract, d.ID, withDiagnosis("J06.9", true), d.VersionID); err == nil {
		t.Fatal("Sign() succeeded")
	}
	if len(f.store.objects) != 0 || f.repo.drafts[d.ID].Status != composition.StatusPreliminary || len(f.jobs.signed)+len(f.jobs.refreshes) != 0 {
		t.Errorf("%d objects, status %s, %d jobs, refreshes %v", len(f.store.objects), f.repo.drafts[d.ID].Status, len(f.jobs.signed), f.jobs.refreshes)
	}
}
