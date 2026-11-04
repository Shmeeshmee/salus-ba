package practitionerrole_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
)

const (
	createdRole       = "a4000000-0000-4000-8000-00000000c0de"
	newPractitionerA  = "a3000000-0000-4000-8000-00000000c0de"
	createInteraction = auditevent.CodeCreate
)

func (r *fakeRepository) Create(ctx context.Context, org, practitionerID string) (practitionerrole.PractitionerRole, error) {
	r.seen(ctx, "Create")
	if r.err != nil {
		return practitionerrole.PractitionerRole{}, r.err
	}
	if org != r.clinic {
		return practitionerrole.PractitionerRole{}, errors.New("row-level security: another clinic")
	}
	p := role(createdRole, org, practitionerID)
	r.roles = append(r.roles, p)
	return p, nil
}

func createEvent(c practitionerrole.Caller, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := event(c, createInteraction, entity, outcome)
	e.Action = auditevent.ActionCreate
	return e
}

func TestCreate_GivesAPractitionerTheirRoleInTheCallersClinicAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}

	got, err := newService(repo, audit).Create(t.Context(), staffA, newPractitionerA)

	if want := role(createdRole, clinicA, newPractitionerA); err != nil || got != want {
		t.Fatalf("Create() = %+v, %v; want %+v", got, err, want)
	}
	if !slices.Equal(repo.calls, []string{"SetOrganization", "Create"}) || repo.clinic != clinicA {
		t.Errorf("calls %v with clinic %q, want the insert inside a transaction scoped to clinic A", repo.calls, repo.clinic)
	}
	if e := createEvent(staffA, createdRole, auditevent.OutcomeSuccess); len(audit.events) != 1 || audit.events[0] != e {
		t.Errorf("audit events %+v, want %+v", audit.events, e)
	}
}

func TestCreate_RefusesACallerWithoutAClinic(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}

	_, err := newService(repo, audit).Create(t.Context(), patient, newPractitionerA)

	if !errors.Is(err, practitionerrole.ErrForbidden) || len(repo.calls) != 0 {
		t.Errorf("Create() by a patient = %v after calls %v, want ErrForbidden and no repository call", err, repo.calls)
	}
	if e := createEvent(patient, "", auditevent.OutcomeMinorFailure); len(audit.events) != 1 || audit.events[0] != e {
		t.Errorf("audit events %+v, want %+v", audit.events, e)
	}
}

func TestCreate_FailsOnARepositoryOrAuditError(t *testing.T) {
	repo := newFakeRepository()
	repo.err = errors.New("connection reset")
	if _, err := newService(repo, &fakeAuditor{}).Create(t.Context(), staffA, newPractitionerA); err == nil {
		t.Error("Create() with a failing repository: error = nil")
	}
	for _, c := range []practitionerrole.Caller{staffA, patient} {
		audit := &fakeAuditor{err: errors.New("audit down")}
		if _, err := newService(newFakeRepository(), audit).Create(t.Context(), c, newPractitionerA); err == nil ||
			errors.Is(err, practitionerrole.ErrForbidden) {
			t.Errorf("Create() as %+v with a failing audit = %v, want its error", c, err)
		}
	}
}
