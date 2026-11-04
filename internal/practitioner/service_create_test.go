package practitioner_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitioner"
)

const createdID = "a3000000-0000-4000-8000-00000000c0de"

func (r *fakeRepository) Create(ctx context.Context, org, given, family string) (practitioner.Practitioner, error) {
	r.seen(ctx, "Create")
	if r.err != nil {
		return practitioner.Practitioner{}, r.err
	}
	if org != r.clinic {
		return practitioner.Practitioner{}, errors.New("row-level security: another clinic")
	}
	p := practitioner.Practitioner{ID: createdID, OrganizationID: org, GivenName: given, FamilyName: family, VersionID: 1, Active: true, LastUpdated: now}
	r.byID[p.ID] = p
	return p, nil
}

func createEvent(c practitioner.Caller, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	e := readEvent(c, entity, outcome)
	e.Subtype.Code, e.Action = auditevent.CodeCreate, auditevent.ActionCreate
	return e
}

func TestCreate_AddsAPractitionerToTheCallersClinicAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}

	got, err := newService(repo, audit).Create(t.Context(), staffA, "Cy", "Example")

	want := practitioner.Practitioner{ID: createdID, OrganizationID: clinicA, GivenName: "Cy", FamilyName: "Example", VersionID: 1, Active: true, LastUpdated: now}
	if err != nil || got != want {
		t.Fatalf("Create() = %+v, %v; want %+v", got, err, want)
	}
	if !slices.Equal(repo.calls, []string{"SetOrganization", "Create"}) || repo.clinic != clinicA {
		t.Errorf("calls %v with clinic %q, want the insert inside a transaction scoped to clinic A", repo.calls, repo.clinic)
	}
	if e := createEvent(staffA, "Practitioner/"+createdID, auditevent.OutcomeSuccess); len(audit.events) != 1 || audit.events[0] != e {
		t.Errorf("audit events %+v, want %+v", audit.events, e)
	}
}

func TestCreate_RefusesACallerWithoutAClinic(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}

	_, err := newService(repo, audit).Create(t.Context(), patient, "Cy", "Example")

	if !errors.Is(err, practitioner.ErrForbidden) || len(repo.calls) != 0 {
		t.Errorf("Create() by a patient = %v after calls %v, want ErrForbidden and no repository call", err, repo.calls)
	}
	if e := createEvent(patient, "", auditevent.OutcomeMinorFailure); len(audit.events) != 1 || audit.events[0] != e {
		t.Errorf("audit events %+v, want %+v", audit.events, e)
	}
}

func TestCreate_FailsOnARepositoryOrAuditError(t *testing.T) {
	repo := newFakeRepository()
	repo.err = errors.New("connection reset")
	if _, err := newService(repo, &fakeAuditor{}).Create(t.Context(), staffA, "Cy", "Example"); err == nil {
		t.Error("Create() with a failing repository: error = nil")
	}
	for _, c := range []practitioner.Caller{staffA, patient} {
		audit := &fakeAuditor{err: errors.New("audit down")}
		if _, err := newService(newFakeRepository(), audit).Create(t.Context(), c, "Cy", "Example"); err == nil || errors.Is(err, practitioner.ErrForbidden) {
			t.Errorf("Create() as %+v with a failing audit = %v, want its error", c, err)
		}
	}
}
