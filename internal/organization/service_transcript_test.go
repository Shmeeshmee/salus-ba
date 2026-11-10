package organization_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/organization"
)

// SAL-17: the clinic's admin turns the patients' transcript copy on and off; an update that leaves it out keeps it.
func TestUpdate_SetsThePatientTranscriptCopyAndKeepsItWhenLeftOut(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	on := true
	got, err := newService(repo, audit).Update(t.Context(), admin, clinicA, organization.Changes{Name: "Synthetic clinic A", PatientTranscriptCopy: &on}, 2)
	if err != nil || !got.PatientTranscriptCopy || !repo.orgs[clinicA].PatientTranscriptCopy {
		t.Fatalf("Update(on) = %+v, %v; want the copy on", got, err)
	}
	got, err = newService(repo, audit).Update(t.Context(), admin, clinicA, organization.Changes{Name: "Synthetic clinic A"}, 3)
	if err != nil || !got.PatientTranscriptCopy {
		t.Errorf("Update(left out) = %+v, %v; want the copy kept on", got, err)
	}
}

// TranscriptCopy tells whether the clinic shares its consultations' transcripts with its patients, read in the
// clinic's scope inside the transaction, unaudited (nothing in a clinic is personal data); ErrNotFound for an
// unknown or missing clinic.
func TestTranscriptCopy_ReadsTheClinicsSetting(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	o := repo.orgs[clinicB]
	o.PatientTranscriptCopy = true
	repo.orgs[clinicB] = o
	svc := newService(repo, audit)

	if got, err := svc.TranscriptCopy(t.Context(), clinicB); err != nil || !got {
		t.Errorf("TranscriptCopy(B) = %v, %v; want true", got, err)
	}
	if want := []string{"SetOrganization", "ByID"}; !slices.Equal(repo.calls, want) || repo.clinic != clinicB {
		t.Errorf("calls %v with clinic %q, want %v with clinic B", repo.calls, repo.clinic, want)
	}
	if got, err := svc.TranscriptCopy(t.Context(), clinicA); err != nil || got {
		t.Errorf("TranscriptCopy(A) = %v, %v; want false", got, err)
	}
	if len(audit.events) != 0 {
		t.Errorf("audit events %+v, want none", audit.events)
	}
	for _, id := range []string{"", "d0000000-0000-4000-8000-00000000000d"} {
		if _, err := svc.TranscriptCopy(t.Context(), id); !errors.Is(err, organization.ErrNotFound) {
			t.Errorf("TranscriptCopy(%q) = %v, want ErrNotFound", id, err)
		}
	}
	repo.err = errors.New("connection reset")
	if _, err := svc.TranscriptCopy(t.Context(), clinicA); err == nil || errors.Is(err, organization.ErrNotFound) {
		t.Errorf("TranscriptCopy(down) = %v, want the failure", err)
	}
}
