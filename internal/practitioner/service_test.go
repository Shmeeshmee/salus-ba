package practitioner_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitioner"
)

const (
	clinicA        = "a0000000-0000-4000-8000-00000000000a"
	clinicB        = "b0000000-0000-4000-8000-00000000000b"
	practitionerA  = "a3000000-0000-4000-8000-0000000000a3"
	practitionerB  = "b3000000-0000-4000-8000-0000000000b3"
	callerAccountA = "a1000000-0000-4000-8000-0000000000a1"
)

var now = time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

type txKey struct{}

type fakeTransactor struct{}

func (fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

// fakeRepository holds practitioners by id and, unless leak is set, shows only those of the clinic set on
// the transaction, as row-level security does.
type fakeRepository struct {
	err    error
	byID   map[string]practitioner.Practitioner
	clinic string
	calls  []string // "outside:" prefixed when made outside a transaction
	leak   bool     // ignore the clinic set: a database without the policy
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byID: map[string]practitioner.Practitioner{
		practitionerA: {ID: practitionerA, OrganizationID: clinicA, GivenName: "Ada", FamilyName: "Example", VersionID: 1, Active: true, LastUpdated: now},
		practitionerB: {ID: practitionerB, OrganizationID: clinicB, GivenName: "Bo", FamilyName: "Example", VersionID: 1, Active: true, LastUpdated: now},
	}}
}

func (r *fakeRepository) seen(ctx context.Context, call string) {
	if ctx.Value(txKey{}) == nil {
		call = "outside:" + call
	}
	r.calls = append(r.calls, call)
}

func (r *fakeRepository) SetOrganization(ctx context.Context, id string) error {
	r.seen(ctx, "SetOrganization")
	r.clinic = id
	return nil
}

func (r *fakeRepository) ByID(ctx context.Context, id string) (practitioner.Practitioner, error) {
	r.seen(ctx, "ByID")
	if r.err != nil {
		return practitioner.Practitioner{}, r.err
	}
	p, ok := r.byID[id]
	if !ok || (!r.leak && p.OrganizationID != r.clinic) {
		return practitioner.Practitioner{}, practitioner.ErrNotFound
	}
	return p, nil
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

func newService(repo *fakeRepository, audit *fakeAuditor) *practitioner.Service {
	return practitioner.NewService(repo, audit, fakeTransactor{})
}

var (
	staffA  = practitioner.Caller{AccountID: callerAccountA, OrganizationID: clinicA}
	patient = practitioner.Caller{AccountID: "c1000000-0000-4000-8000-0000000000c1"}
)

func readEvent(c practitioner.Caller, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:            auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:         auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeRead},
		Purpose:         auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:          auditevent.ActionRead,
		Outcome:         outcome,
		AgentAccountID:  c.AccountID,
		OrganizationID:  c.OrganizationID,
		EntityReference: entity,
	}
}

func TestRead_GivesStaffAPractitionerOfTheirClinicAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).Read(t.Context(), staffA, practitionerA)
	if err != nil || got != repo.byID[practitionerA] {
		t.Fatalf("Read() = %+v, %v; want practitioner A", got, err)
	}
	if !slices.Equal(repo.calls, []string{"SetOrganization", "ByID"}) || repo.clinic != clinicA {
		t.Errorf("calls %v with clinic %q, want the read inside a transaction scoped to clinic A", repo.calls, repo.clinic)
	}
	want := readEvent(staffA, "Practitioner/"+practitionerA, auditevent.OutcomeSuccess)
	if len(audit.events) != 1 || audit.events[0] != want {
		t.Errorf("audit events %+v, want %+v", audit.events, want)
	}
}

func TestRead_HidesAndAuditsAPractitionerOutsideTheCallersClinic(t *testing.T) {
	tests := []struct {
		name   string
		caller practitioner.Caller
		id     string
		leak   bool
		calls  int
	}{
		{name: "a patient", caller: patient, id: practitionerA},
		{name: "another clinic's practitioner", caller: staffA, id: practitionerB, calls: 2},
		{name: "another clinic's practitioner, the database not scoping", caller: staffA, id: practitionerB, leak: true, calls: 2},
		{name: "an unknown id", caller: staffA, id: "not-a-uuid", calls: 2},
	}
	for _, tt := range tests {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		repo.leak = tt.leak
		_, err := newService(repo, audit).Read(t.Context(), tt.caller, tt.id)
		if !errors.Is(err, practitioner.ErrNotFound) || len(repo.calls) != tt.calls {
			t.Errorf("%s: Read() = %v after calls %v, want ErrNotFound after %d calls", tt.name, err, repo.calls, tt.calls)
		}
		// Nothing was disclosed: the event names no entity, so no id from the request is stored.
		if want := readEvent(tt.caller, "", auditevent.OutcomeMinorFailure); len(audit.events) != 1 || audit.events[0] != want {
			t.Errorf("%s: audit events %+v, want %+v", tt.name, audit.events, want)
		}
	}
}

func TestRead_FailsOnARepositoryOrAuditError(t *testing.T) {
	repo := newFakeRepository()
	repo.err = errors.New("connection reset")
	if _, err := newService(repo, &fakeAuditor{}).Read(t.Context(), staffA, practitionerA); err == nil ||
		errors.Is(err, practitioner.ErrNotFound) {
		t.Errorf("Read() with a failing repository = %v, want its error", err)
	}
	for _, c := range []practitioner.Caller{staffA, patient} {
		audit := &fakeAuditor{err: errors.New("audit down")}
		if _, err := newService(newFakeRepository(), audit).Read(t.Context(), c, practitionerA); err == nil ||
			errors.Is(err, practitioner.ErrNotFound) {
			t.Errorf("Read() as %+v with a failing audit = %v, want its error", c, err)
		}
	}
}
