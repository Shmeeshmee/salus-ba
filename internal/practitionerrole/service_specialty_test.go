package practitionerrole_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

const cardiology = "394579002"

var adminA = practitionerrole.Caller{AccountID: "a1000000-0000-4000-8000-0000000000ad", OrganizationID: clinicA, Admin: true}

func (r *fakeRepository) SetSpecialty(ctx context.Context, id, code string, version int32, at time.Time) (practitionerrole.PractitionerRole, error) {
	r.seen(ctx, "SetSpecialty")
	if r.setErr != nil {
		return practitionerrole.PractitionerRole{}, r.setErr
	}
	for i, p := range r.roles {
		if p.ID == id && r.visible(p) && p.VersionID == version {
			p.SpecialtyCode, p.VersionID, p.LastUpdated = code, version+1, at
			r.roles[i] = p
			return p, nil
		}
	}
	return practitionerrole.PractitionerRole{}, practitionerrole.ErrConflict
}

func updateEvent(c practitionerrole.Caller, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := event(c, auditevent.CodeUpdate, entity, outcome)
	e.Action = auditevent.ActionUpdate
	return e
}

func TestSetSpecialty_SetsTheCodeAtTheNextVersionAndAuditsIt(t *testing.T) {
	t.Parallel()
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).SetSpecialty(t.Context(), adminA, roleA1, cardiology, 1)
	if err != nil || got.ID != roleA1 || got.SpecialtyCode != cardiology || got.VersionID != 2 || !got.LastUpdated.Equal(now) {
		t.Fatalf("SetSpecialty() = %+v, %v; want role A1 with cardiology at version 2, updated now", got, err)
	}
	if !slices.Equal(repo.calls, []string{"SetOrganization", "ByID", "SetSpecialty"}) {
		t.Errorf("calls = %v, want the clinic scope, the read and the update in one transaction", repo.calls)
	}
	if want := []auditevent.AuditEvent{updateEvent(adminA, roleA1, auditevent.OutcomeSuccess)}; !slices.Equal(audit.events, want) {
		t.Errorf("events = %+v,\nwant %+v", audit.events, want)
	}
}

func TestSetSpecialty_RejectsACodeNotInTheListBeforeAnyAccess(t *testing.T) {
	t.Parallel()
	for _, code := range []string{"", "394592004", "cardiology"} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		_, err := newService(repo, audit).SetSpecialty(t.Context(), adminA, roleA1, code, 1)
		if !errors.Is(err, practitionerrole.ErrInvalidSpecialty) || len(repo.calls) != 0 || len(audit.events) != 0 {
			t.Errorf("SetSpecialty(%q) = %v, calls %v, events %d; want ErrInvalidSpecialty and nothing accessed",
				code, err, repo.calls, len(audit.events))
		}
	}
}

func TestSetSpecialty_RefusesAndAuditsAnyoneButTheClinicsAdmin(t *testing.T) {
	t.Parallel()
	adminB := practitionerrole.Caller{AccountID: adminA.AccountID, OrganizationID: clinicB, Admin: true}
	tests := []struct {
		want   error
		calls  []string
		caller practitionerrole.Caller
	}{
		{caller: staffA, want: practitionerrole.ErrForbidden},                                            // a practitioner of the clinic
		{caller: patient, want: practitionerrole.ErrNotFound},                                            // no clinic
		{caller: adminB, want: practitionerrole.ErrNotFound, calls: []string{"SetOrganization", "ByID"}}, // another clinic's admin
	}
	for _, tt := range tests {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		_, err := newService(repo, audit).SetSpecialty(t.Context(), tt.caller, roleA1, cardiology, 1)
		if !errors.Is(err, tt.want) || !slices.Equal(repo.calls, tt.calls) {
			t.Errorf("SetSpecialty() as %+v = %v after calls %v; want %v after %v", tt.caller, err, repo.calls, tt.want, tt.calls)
		}
		if want := []auditevent.AuditEvent{updateEvent(tt.caller, "", auditevent.OutcomeMinorFailure)}; !slices.Equal(audit.events, want) {
			t.Errorf("as %+v: events = %+v, want one refusal without an entity", tt.caller, audit.events)
		}
		if repo.roles[0].SpecialtyCode != "" {
			t.Errorf("as %+v: the role changed", tt.caller)
		}
	}
}

func TestSetSpecialty_HidesARoleOfAnotherClinicEvenWithoutTheDatabasePolicy(t *testing.T) {
	t.Parallel()
	repo, audit := newFakeRepository(), &fakeAuditor{}
	repo.leak = true
	_, err := newService(repo, audit).SetSpecialty(t.Context(), adminA, roleB, cardiology, 1)
	if !errors.Is(err, practitionerrole.ErrNotFound) || slices.Contains(repo.calls, "SetSpecialty") {
		t.Errorf("SetSpecialty(role B) as admin A = %v after %v, want ErrNotFound without an update", err, repo.calls)
	}
	if want := []auditevent.AuditEvent{updateEvent(adminA, "", auditevent.OutcomeMinorFailure)}; !slices.Equal(audit.events, want) {
		t.Errorf("events = %+v, want one refusal without an entity", audit.events)
	}
}

func TestSetSpecialty_GivesAConflictForAStaleVersion(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		setErr  error
		version int32
	}{
		"a stale version":         {version: 7},
		"a change since the read": {version: 1, setErr: practitionerrole.ErrConflict},
	} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		repo.setErr = tc.setErr
		_, err := newService(repo, audit).SetSpecialty(t.Context(), adminA, roleA1, cardiology, tc.version)
		if !errors.Is(err, practitionerrole.ErrConflict) || repo.roles[0].SpecialtyCode != "" {
			t.Errorf("%s: SetSpecialty() = %v; want ErrConflict and no change", name, err)
		}
		if want := []auditevent.AuditEvent{updateEvent(adminA, roleA1, auditevent.OutcomeMinorFailure)}; !slices.Equal(audit.events, want) {
			t.Errorf("%s: events = %+v, want one refusal naming the role", name, audit.events)
		}
	}
}

func TestSetSpecialty_FailsOnARepositoryOrAuditError(t *testing.T) {
	t.Parallel()
	broken := errors.New("synthetic failure")
	repo := newFakeRepository()
	repo.err = broken
	if _, err := newService(repo, &fakeAuditor{}).SetSpecialty(t.Context(), adminA, roleA1, cardiology, 1); !errors.Is(err, broken) {
		t.Errorf("SetSpecialty() with a failing read = %v, want the repository error", err)
	}
	repo = newFakeRepository()
	repo.setErr = broken
	if _, err := newService(repo, &fakeAuditor{}).SetSpecialty(t.Context(), adminA, roleA1, cardiology, 1); !errors.Is(err, broken) {
		t.Errorf("SetSpecialty() with a failing update = %v, want the repository error", err)
	}
	for _, c := range []practitionerrole.Caller{adminA, staffA} {
		if _, err := newService(newFakeRepository(), &fakeAuditor{err: broken}).SetSpecialty(t.Context(), c, roleA1, cardiology, 1); !errors.Is(err, broken) {
			t.Errorf("SetSpecialty() as %+v with a failing audit = %v, want the audit error", c, err)
		}
	}
}
