package encounter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Shmeeshmee/salus/internal/access"
	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

// Lookup, for a system path that has only the encounter's id (the consultation's stream consumer), scopes the
// caller's transaction to the encounter's clinic and gives the encounter as that scope shows it, unaudited: the
// caller audits its own write.
func TestLookup_ScopesTheCallersTransactionToTheEncountersClinic(t *testing.T) {
	f := started()
	err := fakeTransactor{f: f}.WithTx(t.Context(), func(ctx context.Context) error {
		got, err := f.svc.Lookup(ctx, encounterOf(apptA))
		if err != nil || got != f.repo.encounters[0] {
			t.Errorf("Lookup() = %+v, %v; want encounter A", got, err)
		}
		if f.repo.clinic != clinicA || f.repo.patient != "" {
			t.Errorf("scoped to clinic %q, patient %q; want clinic A", f.repo.clinic, f.repo.patient)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %d events, want none", len(f.audit.events))
	}
}

func TestLookup_IsNotFoundForAnUnknownEncounter(t *testing.T) {
	f := started()
	if _, err := f.svc.Lookup(t.Context(), unknownID); !errors.Is(err, encounter.ErrNotFound) {
		t.Errorf("Lookup(unknown) error = %v, want ErrNotFound", err)
	}
}

func TestLookup_FailsWhenTheDatabaseFails(t *testing.T) {
	f := started()
	f.repo.err = errBackend
	if _, err := f.svc.Lookup(t.Context(), encounterOf(apptA)); !errors.Is(err, errBackend) || errors.Is(err, encounter.ErrNotFound) {
		t.Errorf("Lookup() error = %v, want the backend's", err)
	}
}

// Access gives the encounter and the caller's attendance of its appointment in the caller's own scope,
// unaudited, at any level the appointment shows them (the caller's rule decides what that allows).
func TestAccess_GivesTheEncounterWithTheCallersAttendance(t *testing.T) {
	f := started()
	att := f.appts.attendance[practAcc+apptA]
	att.Level, att.ConsentActive = access.Name, false
	f.appts.attendance[practAcc+apptA] = att

	e, got, err := f.svc.Access(t.Context(), pract, encounterOf(apptA))
	if err != nil || e != f.repo.encounters[0] || got != att {
		t.Errorf("Access() = %+v, %+v, %v; want encounter A and the practitioner's attendance", e, got, err)
	}
	if f.repo.clinic != clinicA {
		t.Errorf("read in clinic %q, want clinic A", f.repo.clinic)
	}
	if len(f.audit.events) != 0 {
		t.Errorf("audited %d events, want none", len(f.audit.events))
	}
}

func TestAccess_IsNotFoundForAnyoneTheAppointmentDoesNotShow(t *testing.T) {
	for _, tt := range []struct {
		name   string
		id     string
		caller appointment.Caller
	}{
		{name: "another patient", caller: other, id: encounterOf(apptA)},
		{name: "no profile", caller: noProfile, id: encounterOf(apptA)},
		{name: "another clinic's admin", caller: adminB, id: encounterOf(apptA)},
		{name: "an unknown id", caller: pract, id: unknownID},
	} {
		f := started()
		if _, _, err := f.svc.Access(t.Context(), tt.caller, tt.id); !errors.Is(err, encounter.ErrNotFound) {
			t.Errorf("%s: Access() error = %v, want ErrNotFound", tt.name, err)
		}
	}
}

func TestAccess_FailsWhenTheAppointmentsFail(t *testing.T) {
	f := started()
	f.appts.err = errBackend
	if _, _, err := f.svc.Access(t.Context(), pract, encounterOf(apptA)); !errors.Is(err, errBackend) {
		t.Errorf("Access() error = %v, want the backend's", err)
	}
}
