package encounter_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/encounter"
)

func checkEvents(t *testing.T, name string, a *fakeAuditor, want ...auditevent.AuditEvent) {
	t.Helper()
	if !slices.Equal(a.events, want) {
		t.Errorf("%s: audited %+v,\nwant %+v", name, a.events, want)
	}
}

func (f *fixture) start(t *testing.T, appt string) (encounter.Encounter, error) {
	t.Helper()
	var got encounter.Encounter
	err := f.inClinic(t.Context(), func(ctx context.Context) error {
		var err error
		got, err = f.svc.Start(ctx, agent, appt, joined)
		return err
	})
	return got, err
}

func (f *fixture) finish(t *testing.T, appt string, patientJoined bool) error {
	t.Helper()
	return f.inClinic(t.Context(), func(ctx context.Context) error {
		return f.svc.Finish(ctx, agent, appt, now, patientJoined)
	})
}

// The practitioner's first join starts the encounter in progress at the join time, with the appointment's
// patient, role and clinic, audited as a create by the agent.
func TestStart_CreatesTheEncounterInProgressAtTheJoinTime(t *testing.T) {
	f := newFixture()
	got, err := f.start(t, apptA)
	want := encounter.Encounter{
		ID: got.ID, AppointmentID: apptA, OrganizationID: clinicA, PatientID: patientA, PractitionerRoleID: roleA1,
		Status: encounter.StatusInProgress, PeriodStart: joined, LastUpdated: joined, VersionID: 1,
	}
	if err != nil || got != want || got.ID == "" {
		t.Fatalf("Start() = %+v, %v;\nwant %+v", got, err, want)
	}
	checkEvents(t, "Start", f.audit, systemEvent(auditevent.CodeCreate, got.ID))
}

// Start is idempotent: a second start (a repeated webhook, a rejoin) keeps the one encounter with the first
// time and audits nothing more.
func TestStart_TwiceKeepsOneEncounterWithTheFirstTime(t *testing.T) {
	f := newFixture()
	first, err := f.start(t, apptA)
	if err != nil {
		t.Fatal(err)
	}
	var second encounter.Encounter
	err = f.inClinic(t.Context(), func(ctx context.Context) error { // later
		second, err = f.svc.Start(ctx, agent, apptA, now)
		return err
	})
	if err != nil || second != first || len(f.repo.encounters) != 1 {
		t.Errorf("Start() again = %+v, %v with %d encounters; want the first %+v", second, err, len(f.repo.encounters), first)
	}
	checkEvents(t, "Start twice", f.audit, systemEvent(auditevent.CodeCreate, first.ID))
}

// An appointment the scope does not show starts nothing.
func TestStart_IsNotFoundForAnAppointmentOutsideTheScope(t *testing.T) {
	f := newFixture()
	if _, err := f.start(t, unknownID); !errors.Is(err, encounter.ErrNotFound) {
		t.Errorf("Start() = %v, want ErrNotFound", err)
	}
	if len(f.repo.encounters) != 0 || len(f.audit.events) != 0 {
		t.Errorf("stored %v and audited %v, want nothing", f.repo.encounters, f.audit.events)
	}
}

func TestStart_FailsWithTheBackend(t *testing.T) {
	for name, fail := range map[string]func(f *fixture){
		"appointments": func(f *fixture) { f.appts.err = errBackend },
		"repository":   func(f *fixture) { f.repo.err = errBackend },
		"audit":        func(f *fixture) { f.audit.err = errBackend },
	} {
		f := newFixture()
		fail(f)
		if _, err := f.start(t, apptA); !errors.Is(err, errBackend) {
			t.Errorf("%s: Start() = %v, want its error", name, err)
		}
		if len(f.repo.encounters) != 0 {
			t.Errorf("%s: stored %v, want nothing", name, f.repo.encounters)
		}
	}
}

// Finish ends the encounter at the time given, audited as an update by the agent, and records the
// appointment's outcome by whether the patient joined: fulfilled or noshow.
func TestFinish_FinishesTheEncounterAndRecordsTheOutcome(t *testing.T) {
	for _, tt := range []struct {
		outcome string
		joined  bool
	}{{"fulfilled", true}, {"noshow", false}} {
		f := started()
		if err := f.finish(t, apptA, tt.joined); err != nil {
			t.Fatalf("Finish() error = %v", err)
		}
		got := f.repo.encounters[0]
		if got.Status != encounter.StatusFinished || !got.PeriodEnd.Equal(now) || !got.PeriodStart.Equal(start) || got.VersionID != 2 {
			t.Errorf("patient joined %t: encounter %+v, want finished now at version 2", tt.joined, got)
		}
		if want := []string{agent + " " + apptA + " " + tt.outcome}; !slices.Equal(f.appts.outcomes, want) {
			t.Errorf("patient joined %t: outcomes %v, want %v", tt.joined, f.appts.outcomes, want)
		}
		checkEvents(t, "Finish", f.audit, systemEvent(auditevent.CodeUpdate, encounterOf(apptA)))
	}
}

// A finished encounter never changes: a second Finish changes, records and audits nothing, and a later Start
// gives it back as it is.
func TestFinish_TwiceChangesNothingTheSecondTime(t *testing.T) {
	f := started()
	if err := f.finish(t, apptA, true); err != nil {
		t.Fatal(err)
	}
	finished, events := f.repo.encounters[0], len(f.audit.events)
	if err := f.finish(t, apptA, false); err != nil {
		t.Fatal(err)
	}
	again, err := f.start(t, apptA)
	if err != nil || again != finished || f.repo.encounters[0] != finished {
		t.Errorf("after a second Finish and a Start: %+v, %v; want %+v", again, err, finished)
	}
	if len(f.appts.outcomes) != 1 || len(f.audit.events) != events {
		t.Errorf("outcomes %v, %d events; want the first finish's only", f.appts.outcomes, len(f.audit.events))
	}
}

// When the practitioner never joined there is no encounter: Finish creates none and records no outcome.
func TestFinish_NeverStartedCreatesNothing(t *testing.T) {
	f := newFixture()
	if err := f.finish(t, apptA, true); err != nil {
		t.Fatalf("Finish() error = %v", err)
	}
	if len(f.repo.encounters) != 0 || len(f.appts.outcomes) != 0 || len(f.audit.events) != 0 {
		t.Errorf("encounters %v, outcomes %v, events %v; want none", f.repo.encounters, f.appts.outcomes, f.audit.events)
	}
}

func TestFinish_FailsWithTheBackendAndRollsBack(t *testing.T) {
	for name, fail := range map[string]func(f *fixture){
		"repository":   func(f *fixture) { f.repo.err = errBackend },
		"audit":        func(f *fixture) { f.audit.err = errBackend },
		"appointments": func(f *fixture) { f.appts.err = errBackend },
	} {
		f := started()
		fail(f)
		if err := f.finish(t, apptA, true); !errors.Is(err, errBackend) {
			t.Errorf("%s: Finish() = %v, want its error", name, err)
		}
		if f.repo.encounters[0].Status != encounter.StatusInProgress {
			t.Errorf("%s: encounter %+v, want it still in progress", name, f.repo.encounters[0])
		}
	}
}
