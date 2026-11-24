package patient_test

import (
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/patient"
)

// Until package D4 of docs/clinic-cms/WAVE5.md builds them, restricting and lifting a restriction refuse and touch
// nothing.
func TestRestriction_RefusesUntilD4(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	s := newService(repo, audit)
	admin := patient.Caller{AccountID: accountA, OrganizationID: clinicA, Admin: true}
	if err := s.Restrict(t.Context(), admin, patientA, ""); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Restrict: %v, want errors.ErrUnsupported", err)
	}
	if err := s.LiftRestriction(t.Context(), admin, patientA); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("LiftRestriction: %v, want errors.ErrUnsupported", err)
	}
	if len(repo.calls) != 0 || len(audit.events) != 0 {
		t.Errorf("calls %v, audit %v, want none", repo.calls, audit.events)
	}
}
