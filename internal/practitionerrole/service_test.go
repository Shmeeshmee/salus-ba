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

const (
	clinicA = "a0000000-0000-4000-8000-00000000000a"
	clinicB = "b0000000-0000-4000-8000-00000000000b"
	roleA1  = "a4000000-0000-4000-8000-0000000000a1"
	roleA2  = "a4000000-0000-4000-8000-0000000000a2"
	roleB   = "b4000000-0000-4000-8000-0000000000b1"
)

var now = time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

type txKey struct{}

type fakeTransactor struct{}

func (fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

// fakeRepository holds roles in order and, unless leak is set, shows only those of the clinic set on the
// transaction, as row-level security does.
type fakeRepository struct {
	err    error
	roles  []practitionerrole.PractitionerRole
	clinic string
	calls  []string // "outside:" prefixed when made outside a transaction
	leak   bool     // ignore the clinic set: a database without the policy
}

func role(id, org, practitioner string) practitionerrole.PractitionerRole {
	return practitionerrole.PractitionerRole{ID: id, OrganizationID: org, PractitionerID: practitioner, VersionID: 1, Active: true, LastUpdated: now}
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{roles: []practitionerrole.PractitionerRole{
		role(roleA1, clinicA, "a3000000-0000-4000-8000-0000000000a1"),
		role(roleB, clinicB, "b3000000-0000-4000-8000-0000000000b1"),
		role(roleA2, clinicA, "a3000000-0000-4000-8000-0000000000a2"),
	}}
}

func (r *fakeRepository) seen(ctx context.Context, call string) {
	if ctx.Value(txKey{}) == nil {
		call = "outside:" + call
	}
	r.calls = append(r.calls, call)
}

func (r *fakeRepository) visible(p practitionerrole.PractitionerRole) bool {
	return r.leak || p.OrganizationID == r.clinic
}

func (r *fakeRepository) SetOrganization(ctx context.Context, id string) error {
	r.seen(ctx, "SetOrganization")
	r.clinic = id
	return nil
}

func (r *fakeRepository) ByID(ctx context.Context, id string) (practitionerrole.PractitionerRole, error) {
	r.seen(ctx, "ByID")
	if r.err != nil {
		return practitionerrole.PractitionerRole{}, r.err
	}
	for _, p := range r.roles {
		if p.ID == id && r.visible(p) {
			return p, nil
		}
	}
	return practitionerrole.PractitionerRole{}, practitionerrole.ErrNotFound
}

func (r *fakeRepository) ByOrganization(ctx context.Context, org string) ([]practitionerrole.PractitionerRole, error) {
	r.seen(ctx, "ByOrganization")
	if r.err != nil {
		return nil, r.err
	}
	var out []practitionerrole.PractitionerRole
	for _, p := range r.roles {
		if (r.leak || p.OrganizationID == org) && r.visible(p) {
			out = append(out, p)
		}
	}
	return out, nil
}

type fakeAuditor struct {
	err    error
	events []auditevent.AuditEvent
}

func (a *fakeAuditor) Record(_ context.Context, e auditevent.AuditEvent) error {
	if a.err != nil {
		return a.err
	}
	a.events = append(a.events, e)
	return nil
}

func newService(repo *fakeRepository, audit *fakeAuditor) *practitionerrole.Service {
	return practitionerrole.NewService(repo, audit, fakeTransactor{})
}

var (
	staffA  = practitionerrole.Caller{AccountID: "a1000000-0000-4000-8000-0000000000a1", OrganizationID: clinicA}
	patient = practitionerrole.Caller{AccountID: "c1000000-0000-4000-8000-0000000000c1"}
)

func event(c practitionerrole.Caller, interaction string, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := auditevent.ActionRead
	if interaction == auditevent.CodeSearchType {
		action = auditevent.ActionExecute // FHIR AuditEvent.action: a query is an execute
	}
	e := auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: interaction},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         action,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
	if entity != "" {
		e.EntityReference = "PractitionerRole/" + entity
	}
	return e
}

func TestRead_GivesStaffARoleOfTheirClinicAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).Read(t.Context(), staffA, roleA1)
	if err != nil || got != repo.roles[0] {
		t.Fatalf("Read() = %+v, %v; want role A1", got, err)
	}
	if !slices.Equal(repo.calls, []string{"SetOrganization", "ByID"}) || repo.clinic != clinicA {
		t.Errorf("calls %v with clinic %q, want the read inside a transaction scoped to clinic A", repo.calls, repo.clinic)
	}
	if want := event(staffA, auditevent.CodeRead, roleA1, auditevent.OutcomeSuccess); len(audit.events) != 1 || audit.events[0] != want {
		t.Errorf("audit events %+v, want %+v", audit.events, want)
	}
}

func TestRead_HidesAndAuditsARoleOutsideTheCallersClinic(t *testing.T) {
	tests := []struct {
		name   string
		caller practitionerrole.Caller
		id     string
		calls  int
		leak   bool
	}{
		{name: "a patient", caller: patient, id: roleA1},
		{name: "another clinic's role", caller: staffA, id: roleB, calls: 2},
		{name: "another clinic's role, the database not scoping", caller: staffA, id: roleB, leak: true, calls: 2},
	}
	for _, tt := range tests {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		repo.leak = tt.leak
		_, err := newService(repo, audit).Read(t.Context(), tt.caller, tt.id)
		if !errors.Is(err, practitionerrole.ErrNotFound) || len(repo.calls) != tt.calls {
			t.Errorf("%s: Read() = %v after calls %v, want ErrNotFound after %d calls", tt.name, err, repo.calls, tt.calls)
		}
		if want := event(tt.caller, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure); len(audit.events) != 1 || audit.events[0] != want {
			t.Errorf("%s: audit events %+v, want %+v", tt.name, audit.events, want)
		}
	}
}

func TestSearch_ListsTheClinicsRolesAndAuditsEachOne(t *testing.T) {
	for _, leak := range []bool{false, true} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		repo.leak = leak
		got, err := newService(repo, audit).Search(t.Context(), staffA, clinicA)
		if err != nil || len(got) != 2 || got[0].ID != roleA1 || got[1].ID != roleA2 {
			t.Fatalf("leak %v: Search() = %+v, %v; want roles A1 and A2 in order", leak, got, err)
		}
		if !slices.Equal(repo.calls, []string{"SetOrganization", "ByOrganization"}) || repo.clinic != clinicA {
			t.Errorf("calls %v with clinic %q, want the search inside a transaction scoped to clinic A", repo.calls, repo.clinic)
		}
		want := []auditevent.AuditEvent{
			event(staffA, auditevent.CodeSearchType, roleA1, auditevent.OutcomeSuccess),
			event(staffA, auditevent.CodeSearchType, roleA2, auditevent.OutcomeSuccess),
		}
		if !slices.Equal(audit.events, want) {
			t.Errorf("leak %v: audit events %+v, want one per role returned %+v", leak, audit.events, want)
		}
	}
}

func TestSearch_AuditsASearchThatFindsNothing(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	repo.roles = nil
	got, err := newService(repo, audit).Search(t.Context(), staffA, clinicA)
	if err != nil || len(got) != 0 {
		t.Fatalf("Search() = %+v, %v; want no roles", got, err)
	}
	if want := event(staffA, auditevent.CodeSearchType, "", auditevent.OutcomeSuccess); len(audit.events) != 1 || audit.events[0] != want {
		t.Errorf("audit events %+v, want %+v", audit.events, want)
	}
}

func TestSearch_RefusesAndAuditsAnyoneOutsideTheClinic(t *testing.T) {
	for _, c := range []practitionerrole.Caller{patient, {AccountID: staffA.AccountID, OrganizationID: clinicB}} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		_, err := newService(repo, audit).Search(t.Context(), c, clinicA)
		if !errors.Is(err, practitionerrole.ErrNotFound) || len(repo.calls) != 0 {
			t.Errorf("Search() as %+v = %v after calls %v, want ErrNotFound without touching the repository", c, err, repo.calls)
		}
		if want := event(c, auditevent.CodeSearchType, "", auditevent.OutcomeMinorFailure); len(audit.events) != 1 || audit.events[0] != want {
			t.Errorf("audit events %+v, want %+v", audit.events, want)
		}
	}
}

func TestReadAndSearch_FailOnARepositoryOrAuditError(t *testing.T) {
	repo := newFakeRepository()
	repo.err = errors.New("connection reset")
	svc := newService(repo, &fakeAuditor{})
	if _, err := svc.Read(t.Context(), staffA, roleA1); err == nil || errors.Is(err, practitionerrole.ErrNotFound) {
		t.Errorf("Read() with a failing repository = %v, want its error", err)
	}
	if _, err := svc.Search(t.Context(), staffA, clinicA); err == nil {
		t.Errorf("Search() with a failing repository = nil, want its error")
	}
	for _, c := range []practitionerrole.Caller{staffA, patient} {
		svc := newService(newFakeRepository(), &fakeAuditor{err: errors.New("audit down")})
		if _, err := svc.Read(t.Context(), c, roleA1); err == nil || errors.Is(err, practitionerrole.ErrNotFound) {
			t.Errorf("Read() as %+v with a failing audit = %v, want its error", c, err)
		}
		if _, err := svc.Search(t.Context(), c, clinicA); err == nil || errors.Is(err, practitionerrole.ErrNotFound) {
			t.Errorf("Search() as %+v with a failing audit = %v, want its error", c, err)
		}
	}
}
