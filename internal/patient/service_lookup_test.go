package patient_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/patient"
)

// Lookup runs in the caller's transaction and scope (an appointment's, which checked access and audits the
// read itself): it sets no scope and writes no event.
func TestLookup_ReadsTheProfileInTheCallersScopeWithoutAnAudit(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	repo.clinic = clinicA // set by the caller's transaction
	ctx := context.WithValue(t.Context(), txKey{}, true)
	got, err := newService(repo, audit).Lookup(ctx, patientA)
	if err != nil || got.ID != patientA || got.GivenName == "" {
		t.Fatalf("Lookup() = %+v, %v; want patient A", got, err)
	}
	if want := []string{"ByID"}; !slices.Equal(repo.calls, want) || len(audit.events) != 0 {
		t.Errorf("calls %v, events %v; want only %v in the caller's transaction and no event", repo.calls, audit.events, want)
	}

	repo.clinic = ""
	if _, err := newService(repo, audit).Lookup(ctx, patientA); !errors.Is(err, patient.ErrNotFound) {
		t.Errorf("Lookup() outside any scope = %v, want ErrNotFound", err)
	}
	repo.err = errors.New("synthetic failure")
	if _, err := newService(repo, audit).Lookup(ctx, patientA); err == nil {
		t.Error("Lookup() on a failing repository: error = nil")
	}
}
