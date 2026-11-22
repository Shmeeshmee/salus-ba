package practitionerrole_test

import (
	"errors"
	"testing"
)

// The seats of a clinic's membership: its active roles only, never another clinic's, unaudited.
func TestActiveCount_CountsTheClinicsActiveRoles(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	repo.roles[2].Active = false // clinic A's second role
	repo.leak = true             // a database without the policy
	svc := newService(repo, audit)
	if n, err := svc.ActiveCount(t.Context(), clinicA); err != nil || n != 1 {
		t.Errorf("ActiveCount(A) = %d, %v, want 1", n, err)
	}
	if n, err := svc.ActiveCount(t.Context(), clinicB); err != nil || n != 1 {
		t.Errorf("ActiveCount(B) = %d, %v, want 1", n, err)
	}
	if len(audit.events) != 0 {
		t.Errorf("%d audit events, want none", len(audit.events))
	}
	repo.err = errors.New("down")
	if _, err := svc.ActiveCount(t.Context(), clinicA); err == nil {
		t.Error("ActiveCount() with the repository down: error = nil")
	}
}
