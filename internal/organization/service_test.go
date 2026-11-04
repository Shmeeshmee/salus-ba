package organization_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const (
	clinicA = "a0000000-0000-4000-8000-00000000000a"
	clinicB = "b0000000-0000-4000-8000-00000000000b"
	adminA  = "a1000000-0000-4000-8000-0000000000a1"
)

var now = time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

type txKey struct{}

type fakeTransactor struct{}

func (fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

// fakeRepository holds clinics by id and shows only the clinic set on the transaction, as row-level
// security does.
type fakeRepository struct {
	err       error // returned by ByID when set
	renameErr error // returned by Rename when set
	orgs      map[string]organization.Organization
	clinic    string   // set by SetOrganization
	calls     []string // the calls, "outside:" prefixed when made outside a transaction
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{orgs: map[string]organization.Organization{
		clinicA: {ID: clinicA, Name: "Synthetic clinic A", VersionID: 2, Active: true, LastUpdated: now.Add(-time.Hour)},
		clinicB: {ID: clinicB, Name: "Synthetic clinic B", VersionID: 1, Active: true, LastUpdated: now.Add(-time.Hour)},
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

func (r *fakeRepository) ByID(ctx context.Context, id string) (organization.Organization, error) {
	r.seen(ctx, "ByID")
	if r.err != nil {
		return organization.Organization{}, r.err
	}
	o, ok := r.orgs[id]
	if !ok || id != r.clinic {
		return organization.Organization{}, organization.ErrNotFound
	}
	return o, nil
}

func (r *fakeRepository) Rename(ctx context.Context, id, name string, version int32, at time.Time) (organization.Organization, error) {
	r.seen(ctx, "Rename")
	if r.renameErr != nil {
		return organization.Organization{}, r.renameErr
	}
	o := r.orgs[id]
	if id != r.clinic || o.VersionID != version {
		return organization.Organization{}, organization.ErrConflict
	}
	o.Name, o.VersionID, o.LastUpdated = name, version+1, at
	r.orgs[id] = o
	return o, nil
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

func newService(repo *fakeRepository, audit *fakeAuditor) *organization.Service {
	return organization.NewService(repo, audit, fakeTransactor{}, clock.Fixed{T: now})
}

var (
	admin        = organization.Caller{AccountID: adminA, OrganizationID: clinicA, Admin: true}
	practitioner = organization.Caller{AccountID: "a2000000-0000-4000-8000-0000000000a2", OrganizationID: clinicA}
	otherAdmin   = organization.Caller{AccountID: "b1000000-0000-4000-8000-0000000000b1", OrganizationID: clinicB, Admin: true}
	patient      = organization.Caller{AccountID: "c1000000-0000-4000-8000-0000000000c1"}
)

func TestRead_GivesStaffTheirOwnClinicThroughItsScope(t *testing.T) {
	for _, c := range []organization.Caller{admin, practitioner} {
		repo := newFakeRepository()
		got, err := newService(repo, &fakeAuditor{}).Read(t.Context(), c, clinicA)
		if err != nil || got != repo.orgs[clinicA] {
			t.Errorf("Read() as %+v = %+v, %v; want clinic A", c, got, err)
		}
		if want := []string{"SetOrganization", "ByID"}; !slices.Equal(repo.calls, want) || repo.clinic != clinicA {
			t.Errorf("calls %v with clinic %q set, want %v inside the transaction with clinic A", repo.calls, repo.clinic, want)
		}
	}
}

func TestRead_HidesAClinicFromAnyoneOutsideIt(t *testing.T) {
	for _, c := range []organization.Caller{otherAdmin, patient} {
		repo := newFakeRepository()
		_, err := newService(repo, &fakeAuditor{}).Read(t.Context(), c, clinicA)
		if !errors.Is(err, organization.ErrNotFound) || len(repo.calls) != 0 {
			t.Errorf("Read() as %+v: %v after calls %v; want ErrNotFound without touching the repository", c, err, repo.calls)
		}
	}
}

func TestRead_FailsOnARepositoryError(t *testing.T) {
	repo := newFakeRepository()
	repo.err = errors.New("connection reset")
	_, err := newService(repo, &fakeAuditor{}).Read(t.Context(), admin, clinicA)
	if err == nil || errors.Is(err, organization.ErrNotFound) {
		t.Errorf("Read() = %v, want the repository error", err)
	}
}

func TestUpdate_RenamesTheClinicAndAuditsIt(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	got, err := newService(repo, audit).Update(t.Context(), admin, clinicA, "  Synthetic clinic renamed ", 2)
	want := organization.Organization{ID: clinicA, Name: "Synthetic clinic renamed", VersionID: 3, Active: true, LastUpdated: now}
	if err != nil || got != want {
		t.Fatalf("Update() = %+v, %v;\nwant %+v", got, err, want)
	}
	if !slices.Equal(repo.calls, []string{"SetOrganization", "ByID", "Rename"}) {
		t.Errorf("calls %v, want the clinic set, then the read and the rename in one transaction", repo.calls)
	}
	wantEvent := updateEvent(admin, auditevent.OutcomeSuccess)
	wantEvent.EntityReference = "Organization/" + clinicA
	if len(audit.events) != 1 || audit.events[0] != wantEvent {
		t.Errorf("audit events %+v, want %+v", audit.events, wantEvent)
	}
}

func TestUpdate_RejectsABlankOrOverlongName(t *testing.T) {
	long := make([]rune, 201)
	for i := range long {
		long[i] = 'é'
	}
	for _, name := range []string{"", "   ", string(long)} {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		_, err := newService(repo, audit).Update(t.Context(), admin, clinicA, name, 2)
		if !errors.Is(err, organization.ErrInvalidName) || len(repo.calls) != 0 || len(audit.events) != 0 {
			t.Errorf("Update(%d characters) = %v, calls %v, events %d; want ErrInvalidName before any access",
				len([]rune(name)), err, repo.calls, len(audit.events))
		}
	}
}

func TestUpdate_RefusesAndAuditsAnyoneButTheClinicsAdmin(t *testing.T) {
	tests := []struct {
		want   error
		caller organization.Caller
	}{
		{caller: practitioner, want: organization.ErrForbidden},
		{caller: otherAdmin, want: organization.ErrNotFound},
		{caller: patient, want: organization.ErrNotFound},
	}
	for _, tt := range tests {
		repo, audit := newFakeRepository(), &fakeAuditor{}
		_, err := newService(repo, audit).Update(t.Context(), tt.caller, clinicA, "Renamed", 2)
		if !errors.Is(err, tt.want) || len(repo.calls) != 0 {
			t.Errorf("Update() as %+v = %v after calls %v, want %v without touching the repository", tt.caller, err, repo.calls, tt.want)
		}
		// Nothing was accessed, so the event names no entity.
		if want := updateEvent(tt.caller, auditevent.OutcomeMinorFailure); len(audit.events) != 1 || audit.events[0] != want {
			t.Errorf("audit events %+v, want %+v", audit.events, want)
		}
	}
}

func TestUpdate_GivesAConflictForAStaleVersion(t *testing.T) {
	for name, repo := range map[string]*fakeRepository{
		"stale when read": newFakeRepository(),
		"changed between read and update": func() *fakeRepository {
			r := newFakeRepository()
			r.renameErr = organization.ErrConflict
			return r
		}(),
	} {
		audit := &fakeAuditor{}
		version := int32(1)
		if repo.renameErr != nil {
			version = 2
		}
		_, err := newService(repo, audit).Update(t.Context(), admin, clinicA, "Renamed", version)
		if !errors.Is(err, organization.ErrConflict) || repo.orgs[clinicA].Name != "Synthetic clinic A" {
			t.Errorf("%s: Update() = %v, name %q; want ErrConflict and no change", name, err, repo.orgs[clinicA].Name)
		}
		want := updateEvent(admin, auditevent.OutcomeMinorFailure)
		want.EntityReference = "Organization/" + clinicA
		if len(audit.events) != 1 || audit.events[0] != want {
			t.Errorf("%s: audit events %+v, want %+v", name, audit.events, want)
		}
	}
}

func TestUpdate_AuditsAClinicTheDatabaseDoesNotShow(t *testing.T) {
	repo, audit := newFakeRepository(), &fakeAuditor{}
	delete(repo.orgs, clinicA)
	_, err := newService(repo, audit).Update(t.Context(), admin, clinicA, "Renamed", 2)
	if !errors.Is(err, organization.ErrNotFound) {
		t.Errorf("Update() = %v, want ErrNotFound", err)
	}
	if want := updateEvent(admin, auditevent.OutcomeMinorFailure); len(audit.events) != 1 || audit.events[0] != want {
		t.Errorf("audit events %+v, want %+v", audit.events, want)
	}
}

func TestUpdate_FailsOnARepositoryOrAuditError(t *testing.T) {
	repo := newFakeRepository()
	repo.err = errors.New("connection reset")
	if _, err := newService(repo, &fakeAuditor{}).Update(t.Context(), admin, clinicA, "Renamed", 2); err == nil ||
		errors.Is(err, organization.ErrNotFound) {
		t.Errorf("Update() with a failing read = %v, want the repository error", err)
	}
	repo = newFakeRepository()
	repo.renameErr = errors.New("connection reset")
	if _, err := newService(repo, &fakeAuditor{}).Update(t.Context(), admin, clinicA, "Renamed", 2); err == nil ||
		errors.Is(err, organization.ErrConflict) {
		t.Errorf("Update() with a failing rename = %v, want the repository error", err)
	}
	for _, c := range []organization.Caller{admin, practitioner} {
		audit := &fakeAuditor{err: errors.New("audit down")}
		if _, err := newService(newFakeRepository(), audit).Update(t.Context(), c, clinicA, "Renamed", 2); err == nil {
			t.Errorf("Update() as %+v with a failing audit = nil, want an error", c)
		}
	}
}

func updateEvent(c organization.Caller, outcome auditevent.Outcome) auditevent.AuditEvent {
	return auditevent.AuditEvent{
		Type:           auditevent.Coding{System: auditevent.SystemAuditEventType, Code: auditevent.CodeRest},
		Subtype:        auditevent.Coding{System: auditevent.SystemRestfulInteraction, Code: auditevent.CodeUpdate},
		Purpose:        auditevent.Coding{System: auditevent.SystemActReason, Code: auditevent.CodeHOPERAT},
		Action:         auditevent.ActionUpdate,
		Outcome:        outcome,
		AgentAccountID: c.AccountID,
		OrganizationID: c.OrganizationID,
	}
}
