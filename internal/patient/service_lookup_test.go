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

// LookupAll is Lookup for several patients in one read: the profiles the scope shows, by id, none for the others.
func TestLookupAll_ReadsTheProfilesInOneCallWithoutAnAudit(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	repo.clinic = clinicA
	ctx := context.WithValue(t.Context(), txKey{}, true)
	got, err := newService(repo, audit).LookupAll(ctx, []string{patientA, "d9000000-0000-4000-8000-0000000000d9"})
	if err != nil || len(got) != 1 || got[patientA].GivenName == "" {
		t.Fatalf("LookupAll() = %+v, %v; want patient A only", got, err)
	}
	if want := []string{"ByIDs"}; !slices.Equal(repo.calls, want) || len(audit.events) != 0 {
		t.Errorf("calls %v, events %v; want only %v and no event", repo.calls, audit.events, want)
	}
	repo.err = errors.New("synthetic failure")
	if _, err := newService(repo, audit).LookupAll(ctx, []string{patientA}); err == nil {
		t.Error("LookupAll() on a failing repository: error = nil")
	}
}

// RestrictedAll is Restricted for several patients in one read.
func TestRestrictedAll_TellsEachPatientsRecordInOneCall(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	repo.restricted = map[string]string{patientA + "/" + clinicA: "staff"}
	ctx := context.WithValue(t.Context(), txKey{}, true)
	got, err := newService(repo, audit).RestrictedAll(ctx, []string{patientA, patientB}, clinicA)
	if err != nil || !got[patientA] || got[patientB] {
		t.Fatalf("RestrictedAll() = %v, %v; want A only", got, err)
	}
	if want := []string{"RestrictedAll"}; !slices.Equal(repo.calls, want) {
		t.Errorf("calls %v, want %v", repo.calls, want)
	}
}
