package binary_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/binary"
)

// SAL-39: Clean gives, of the files its holder lists in the patient's timeline, those that are the patient's own
// and clean, in their scope, unaudited (history.Timeline audits the page); a quarantined, rejected or another
// patient's file is left out.
func TestClean_GivesThePatientsOwnCleanFiles(t *testing.T) {
	f := newFixture(t)
	ids := []string{
		"b1000000-0000-4000-8000-0000000000b1", "b2000000-0000-4000-8000-0000000000b2",
		"b3000000-0000-4000-8000-0000000000b3", "b4000000-0000-4000-8000-0000000000b4",
	}
	for i, st := range []binary.Status{binary.StatusClean, binary.StatusQuarantined, binary.StatusRejected, binary.StatusClean} {
		patient := patientA
		if i == 3 {
			patient = patientB
		}
		f.repo.files[ids[i]] = binary.Binary{ID: ids[i], OrganizationID: clinicA, PatientID: patient, ResponseID: responseA, Status: st}
	}
	got, err := f.svc.Clean(t.Context(), me, append(ids, "b5000000-0000-4000-8000-0000000000b5"))
	if err != nil || !slices.Equal(got, ids[:1]) {
		t.Fatalf("Clean() = %v, %v; want the first file only", got, err)
	}
	if f.repo.patient != patientA {
		t.Errorf("read in patient %q, want patient A's scope", f.repo.patient)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %+v, want nothing: history.Timeline audits the page it returns", f.audit.events)
	}
}

// No id asks nothing; staff get ErrForbidden; nothing is audited; the backend's failure is returned.
func TestClean_NoneStaffAndFailure(t *testing.T) {
	f := newFixture(t)
	if got, err := f.svc.Clean(t.Context(), me, nil); err != nil || len(got) != 0 || len(f.audit.events) != 0 {
		t.Errorf("Clean(none) = %v, %v, audited %d; want nothing", got, err, len(f.audit.events))
	}
	if _, err := f.svc.Clean(t.Context(), pract, []string{"b1000000-0000-4000-8000-0000000000b1"}); !errors.Is(err, binary.ErrForbidden) {
		t.Errorf("Clean() as staff = %v, want ErrForbidden", err)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %+v, want nothing", f.audit.events)
	}
	f.repo.err = errBackend
	if _, err := f.svc.Clean(t.Context(), me, []string{"b1000000-0000-4000-8000-0000000000b1"}); !errors.Is(err, errBackend) {
		t.Errorf("Clean() = %v, want the backend's error", err)
	}
	if got, err := f.svc.Clean(t.Context(), binary.Caller{AccountID: accountA}, []string{"x"}); err != nil || len(got) != 0 {
		t.Errorf("Clean() without a profile = %v, %v; want none", got, err)
	}
}
