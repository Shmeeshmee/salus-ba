package schedule_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const (
	clinicA       = "a0000000-0000-4000-8000-00000000000a"
	clinicB       = "b0000000-0000-4000-8000-00000000000b"
	roleA1        = "a4000000-0000-4000-8000-0000000000a1"
	roleA2        = "a4000000-0000-4000-8000-0000000000a2"
	roleB         = "b4000000-0000-4000-8000-0000000000b1"
	practitioner1 = "a3000000-0000-4000-8000-0000000000a1"
	practitioner2 = "a3000000-0000-4000-8000-0000000000a2"
	scheduleA2    = "a5000000-0000-4000-8000-0000000000a2"
	scheduleB     = "b5000000-0000-4000-8000-0000000000b1"
)

var now = time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

var (
	practitionerA1 = schedule.Caller{AccountID: "a1000000-0000-4000-8000-0000000000a1", OrganizationID: clinicA, PractitionerID: practitioner1}
	adminA         = schedule.Caller{AccountID: "a1000000-0000-4000-8000-0000000000ad", OrganizationID: clinicA, Admin: true}
	adminB         = schedule.Caller{AccountID: "b1000000-0000-4000-8000-0000000000bd", OrganizationID: clinicB, Admin: true}
	patient        = schedule.Caller{AccountID: "c1000000-0000-4000-8000-0000000000c1"}
)

var weekdays = schedule.Availability{SlotMinutes: 30, Rules: []schedule.Rule{
	{Day: 3, Start: "13:00", End: "17:00"},
	{Day: 1, Start: "09:00", End: "12:00"},
	{Day: 1, Start: "12:00", End: "13:00"}, // back to back is not an overlap
}}

type txKey struct{}

type fakeTransactor struct{}

func (fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

// log records calls of every fake in order, "outside:" prefixed when made outside a transaction.
type log struct{ calls []string }

func (l *log) seen(ctx context.Context, call string) {
	if ctx.Value(txKey{}) == nil {
		call = "outside:" + call
	}
	l.calls = append(l.calls, call)
}

// fakeRepository holds schedules and, unless leak is set, shows only those of the clinic set, or every
// schedule in the directory scope, as row-level security does.
type fakeRepository struct {
	*log
	err       error
	createErr error
	clinic    string
	schedules []schedule.Schedule
	leak      bool
	directory bool
}

func (r *fakeRepository) visible(s schedule.Schedule) bool {
	return r.leak || r.directory || s.OrganizationID == r.clinic
}

func (r *fakeRepository) SetOrganization(ctx context.Context, id string) error {
	r.seen(ctx, "SetOrganization")
	r.clinic = id
	return nil
}

func (r *fakeRepository) SetDirectory(ctx context.Context) error {
	r.seen(ctx, "SetDirectory")
	r.directory = true
	return nil
}

func (r *fakeRepository) find(match func(schedule.Schedule) bool) (schedule.Schedule, error) {
	if r.err != nil {
		return schedule.Schedule{}, r.err
	}
	for _, s := range r.schedules {
		if match(s) && r.visible(s) {
			return s, nil
		}
	}
	return schedule.Schedule{}, schedule.ErrNotFound
}

func (r *fakeRepository) ByID(ctx context.Context, id string) (schedule.Schedule, error) {
	r.seen(ctx, "ByID")
	return r.find(func(s schedule.Schedule) bool { return s.ID == id })
}

func (r *fakeRepository) ByRole(ctx context.Context, roleID string) (schedule.Schedule, error) {
	r.seen(ctx, "ByRole")
	return r.find(func(s schedule.Schedule) bool { return s.PractitionerRoleID == roleID })
}

func (r *fakeRepository) Lock(ctx context.Context, id string) (schedule.Schedule, error) {
	r.seen(ctx, "Lock")
	return r.find(func(s schedule.Schedule) bool { return s.ID == id })
}

func (r *fakeRepository) Listed(ctx context.Context) ([]schedule.Schedule, error) {
	r.seen(ctx, "Listed")
	var out []schedule.Schedule
	for _, s := range r.schedules {
		if r.visible(s) {
			out = append(out, s)
		}
	}
	return out, r.err
}

func (r *fakeRepository) Create(ctx context.Context, s schedule.Schedule) (schedule.Schedule, error) {
	r.seen(ctx, "Create")
	if r.createErr != nil {
		return schedule.Schedule{}, r.createErr
	}
	s.ID, s.VersionID, s.LastUpdated = "a5000000-0000-4000-8000-0000000000a1", 1, now
	r.schedules = append(r.schedules, s)
	return s, nil
}

func (r *fakeRepository) Update(ctx context.Context, s schedule.Schedule, at time.Time) (schedule.Schedule, error) {
	r.seen(ctx, "Update")
	for i, cur := range r.schedules {
		if cur.ID == s.ID && r.visible(cur) && cur.VersionID == s.VersionID {
			s.VersionID, s.LastUpdated = s.VersionID+1, at
			r.schedules[i] = s
			return s, nil
		}
	}
	return schedule.Schedule{}, schedule.ErrConflict
}

// fakeRoles reads the roles of practitionerrole as its service does: only to staff of the role's clinic.
type fakeRoles struct{ *log }

func (r fakeRoles) Read(ctx context.Context, c practitionerrole.Caller, id string) (practitionerrole.PractitionerRole, error) {
	r.seen(ctx, "Roles.Read")
	for _, p := range []practitionerrole.PractitionerRole{
		{ID: roleA1, OrganizationID: clinicA, PractitionerID: practitioner1, Active: true},
		{ID: roleA2, OrganizationID: clinicA, PractitionerID: practitioner2, Active: true},
		{ID: roleB, OrganizationID: clinicB, PractitionerID: "b3000000-0000-4000-8000-0000000000b1", Active: true},
	} {
		if p.ID == id && c.OrganizationID == p.OrganizationID {
			return p, nil
		}
	}
	return practitionerrole.PractitionerRole{}, practitionerrole.ErrNotFound
}

type job struct{ schedule, org string }

type fakeJobs struct {
	*log
	jobs []job
}

func (j *fakeJobs) EnqueueSlotsGenerate(ctx context.Context, scheduleID, org string) error {
	j.seen(ctx, "EnqueueSlotsGenerate")
	j.jobs = append(j.jobs, job{scheduleID, org})
	return nil
}

type fakeAuditor struct {
	events []auditevent.AuditEvent
}

func (a *fakeAuditor) Record(_ context.Context, e auditevent.AuditEvent) error {
	a.events = append(a.events, e)
	return nil
}

type fixture struct {
	*log
	repo  *fakeRepository
	jobs  *fakeJobs
	audit *fakeAuditor
	svc   *schedule.Service
}

func newFixture() fixture {
	l := &log{}
	f := fixture{
		log: l,
		repo: &fakeRepository{log: l, schedules: []schedule.Schedule{
			{ID: scheduleA2, OrganizationID: clinicA, PractitionerRoleID: roleA2, SlotMinutes: 20, VersionID: 3,
				Rules: []schedule.Rule{{Day: 2, Start: "08:00", End: "10:00"}}},
			{ID: scheduleB, OrganizationID: clinicB, PractitionerRoleID: roleB, SlotMinutes: 15, VersionID: 1},
		}},
		jobs:  &fakeJobs{log: l},
		audit: &fakeAuditor{},
	}
	f.svc = schedule.NewService(schedule.Ports{
		Repo: f.repo, Audit: f.audit, Tx: fakeTransactor{}, Clock: clock.Fixed{T: now}, Roles: fakeRoles{l}, Jobs: f.jobs,
	})
	return f
}

func event(c schedule.Caller, interaction, entity string, outcome auditevent.Outcome) auditevent.AuditEvent {
	action := map[string]auditevent.Action{
		auditevent.CodeRead: auditevent.ActionRead, auditevent.CodeCreate: auditevent.ActionCreate, auditevent.CodeUpdate: auditevent.ActionUpdate,
	}[interaction]
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
		e.EntityReference = "Schedule/" + entity
	}
	return e
}

func TestSaveAvailability_CreatesTheScheduleSortedEnqueuesItsSlotsAndAuditsIt(t *testing.T) {
	t.Parallel()
	f := newFixture()

	got, err := f.svc.SaveAvailability(t.Context(), practitionerA1, roleA1, weekdays, 0)

	want := []schedule.Rule{{Day: 1, Start: "09:00", End: "12:00"}, {Day: 1, Start: "12:00", End: "13:00"}, {Day: 3, Start: "13:00", End: "17:00"}}
	if err != nil || got.ID == "" || got.OrganizationID != clinicA || got.PractitionerRoleID != roleA1 || got.SlotMinutes != 30 ||
		got.VersionID != 1 || !slices.Equal(got.Rules, want) {
		t.Fatalf("SaveAvailability() = %+v, %v; want role A1's new schedule at version 1 with the rules sorted", got, err)
	}
	if want := []string{"SetOrganization", "Roles.Read", "ByRole", "Create", "EnqueueSlotsGenerate"}; !slices.Equal(f.calls, want) {
		t.Errorf("calls = %v, want %v in one transaction", f.calls, want)
	}
	if want := []job{{got.ID, clinicA}}; !slices.Equal(f.jobs.jobs, want) {
		t.Errorf("jobs = %v, want %v", f.jobs.jobs, want)
	}
	if want := []auditevent.AuditEvent{event(practitionerA1, auditevent.CodeCreate, got.ID, auditevent.OutcomeSuccess)}; !slices.Equal(f.audit.events, want) {
		t.Errorf("events = %+v,\nwant %+v", f.audit.events, want)
	}
}

func TestSaveAvailability_UpdatesTheCurrentVersionOfAnyRoleForTheAdmin(t *testing.T) {
	t.Parallel()
	f := newFixture()

	got, err := f.svc.SaveAvailability(t.Context(), adminA, roleA2, weekdays, 3)

	if err != nil || got.ID != scheduleA2 || got.VersionID != 4 || got.SlotMinutes != 30 || len(got.Rules) != 3 || !got.LastUpdated.Equal(now) {
		t.Fatalf("SaveAvailability() = %+v, %v; want schedule A2 at version 4 with the new rules, updated now", got, err)
	}
	if want := []job{{scheduleA2, clinicA}}; !slices.Equal(f.jobs.jobs, want) {
		t.Errorf("jobs = %v, want %v", f.jobs.jobs, want)
	}
	if want := []auditevent.AuditEvent{event(adminA, auditevent.CodeUpdate, scheduleA2, auditevent.OutcomeSuccess)}; !slices.Equal(f.audit.events, want) {
		t.Errorf("events = %+v,\nwant %+v", f.audit.events, want)
	}
}

func TestSaveAvailability_GivesAConflictForAVersionThatIsNotTheCurrentOne(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		createErr error
		role      string
		version   int32
	}{
		"a stale version":                      {role: roleA2, version: 2},
		"a create when the role has one":       {role: roleA2, version: 0},
		"an update when the role has none":     {role: roleA1, version: 1},
		"a schedule created since it was read": {role: roleA1, version: 0, createErr: schedule.ErrConflict},
	} {
		f := newFixture()
		f.repo.createErr = tc.createErr
		_, err := f.svc.SaveAvailability(t.Context(), adminA, tc.role, weekdays, tc.version)
		if !errors.Is(err, schedule.ErrConflict) || len(f.jobs.jobs) != 0 || f.repo.schedules[0].VersionID != 3 {
			t.Errorf("%s: SaveAvailability() = %v with jobs %v; want ErrConflict, no change and no job", name, err, f.jobs.jobs)
		}
		if len(f.audit.events) != 1 || f.audit.events[0].Outcome != auditevent.OutcomeMinorFailure {
			t.Errorf("%s: events = %+v, want one refusal", name, f.audit.events)
		}
	}
}

func TestSaveAvailability_RefusesAndAuditsEveryoneButThePractitionerOfTheRoleAndTheAdmin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		want   error
		role   string
		caller schedule.Caller
	}{
		{caller: practitionerA1, role: roleA2, want: schedule.ErrForbidden},                                           // another practitioner's role
		{caller: schedule.Caller{AccountID: "x", OrganizationID: clinicA}, role: roleA1, want: schedule.ErrForbidden}, // staff who is no practitioner
		{caller: adminB, role: roleA1, want: schedule.ErrNotFound},                                                    // another clinic
		{caller: adminA, role: "a4000000-0000-4000-8000-0000000000ff", want: schedule.ErrNotFound},                    // no such role
		{caller: patient, role: roleA1, want: schedule.ErrNotFound},                                                   // no clinic
	}
	for _, tt := range tests {
		f := newFixture()
		_, err := f.svc.SaveAvailability(t.Context(), tt.caller, tt.role, weekdays, 0)
		if !errors.Is(err, tt.want) || slices.Contains(f.calls, "Create") || len(f.jobs.jobs) != 0 {
			t.Errorf("SaveAvailability(%s) as %+v = %v after %v; want %v and nothing written", tt.role, tt.caller, err, f.calls, tt.want)
		}
		if want := []auditevent.AuditEvent{event(tt.caller, auditevent.CodeUpdate, "", auditevent.OutcomeMinorFailure)}; !slices.Equal(f.audit.events, want) {
			t.Errorf("as %+v: events = %+v, want one refusal without an entity", tt.caller, f.audit.events)
		}
	}
}

func TestSaveAvailability_RejectsAnInvalidLengthOrRuleBeforeAnyAccess(t *testing.T) {
	t.Parallel()
	rules := func(r ...schedule.Rule) schedule.Availability {
		return schedule.Availability{SlotMinutes: 30, Rules: r}
	}
	tests := []struct {
		want error
		name string
		in   schedule.Availability
	}{
		{name: "length 0", in: schedule.Availability{SlotMinutes: 0}, want: schedule.ErrInvalidSlotMinutes},
		{name: "length 4", in: schedule.Availability{SlotMinutes: 4}, want: schedule.ErrInvalidSlotMinutes},
		{name: "length not a multiple of 5", in: schedule.Availability{SlotMinutes: 32}, want: schedule.ErrInvalidSlotMinutes},
		{name: "length 245", in: schedule.Availability{SlotMinutes: 245}, want: schedule.ErrInvalidSlotMinutes},
		{name: "day 0", in: rules(schedule.Rule{Day: 0, Start: "09:00", End: "10:00"}), want: schedule.ErrInvalidAvailability},
		{name: "day 8", in: rules(schedule.Rule{Day: 8, Start: "09:00", End: "10:00"}), want: schedule.ErrInvalidAvailability},
		{name: "end before start", in: rules(schedule.Rule{Day: 1, Start: "10:00", End: "09:00"}), want: schedule.ErrInvalidAvailability},
		{name: "end at start", in: rules(schedule.Rule{Day: 1, Start: "10:00", End: "10:00"}), want: schedule.ErrInvalidAvailability},
		{name: "not HH:MM", in: rules(schedule.Rule{Day: 1, Start: "9:00", End: "10:00"}), want: schedule.ErrInvalidAvailability},
		{name: "hour 24", in: rules(schedule.Rule{Day: 1, Start: "09:00", End: "24:00"}), want: schedule.ErrInvalidAvailability},
		{name: "minute 60", in: rules(schedule.Rule{Day: 1, Start: "09:60", End: "10:00"}), want: schedule.ErrInvalidAvailability},
		{name: "a sign", in: rules(schedule.Rule{Day: 1, Start: "+9:00", End: "10:00"}), want: schedule.ErrInvalidAvailability},
		{name: "overlap", in: rules(schedule.Rule{Day: 2, Start: "09:00", End: "11:00"}, schedule.Rule{Day: 2, Start: "10:30", End: "12:00"}), want: schedule.ErrInvalidAvailability},
		{name: "contained", in: rules(schedule.Rule{Day: 2, Start: "10:00", End: "10:30"}, schedule.Rule{Day: 2, Start: "09:00", End: "12:00"}), want: schedule.ErrInvalidAvailability},
	}
	for _, tt := range tests {
		f := newFixture()
		_, err := f.svc.SaveAvailability(t.Context(), adminA, roleA1, tt.in, 0)
		if !errors.Is(err, tt.want) || len(f.calls) != 0 || len(f.audit.events) != 0 {
			t.Errorf("%s: SaveAvailability() = %v after %v with %d events; want %v and nothing accessed", tt.name, err, f.calls, len(f.audit.events), tt.want)
		}
	}
	// Valid: no rules at all, the bounds of the length, a day to the last minute.
	for _, in := range []schedule.Availability{{SlotMinutes: 5}, {SlotMinutes: 240}, rules(schedule.Rule{Day: 7, Start: "00:00", End: "23:59"})} {
		if _, err := newFixture().svc.SaveAvailability(t.Context(), adminA, roleA1, in, 0); err != nil {
			t.Errorf("SaveAvailability(%+v) = %v, want it saved", in, err)
		}
	}
}

func TestAvailability_GivesTheScheduleOrAnEmptyOneBeforeTheFirstEditAndAuditsTheRead(t *testing.T) {
	t.Parallel()
	f := newFixture()
	got, err := f.svc.Availability(t.Context(), adminA, roleA2)
	if err != nil || got.ID != scheduleA2 || got.VersionID != 3 || len(got.Rules) != 1 {
		t.Errorf("Availability(A2) = %+v, %v; want schedule A2", got, err)
	}
	g := newFixture()
	empty, err := g.svc.Availability(t.Context(), practitionerA1, roleA1)
	if err != nil || empty.ID != "" || empty.PractitionerRoleID != roleA1 || empty.OrganizationID != clinicA || empty.VersionID != 0 || len(empty.Rules) != 0 {
		t.Errorf("Availability(A1) = %+v, %v; want an empty schedule of role A1 at version 0", empty, err)
	}
	if want := []auditevent.AuditEvent{event(adminA, auditevent.CodeRead, scheduleA2, auditevent.OutcomeSuccess)}; !slices.Equal(f.audit.events, want) {
		t.Errorf("events = %+v,\nwant %+v", f.audit.events, want)
	}
	if want := []auditevent.AuditEvent{event(practitionerA1, auditevent.CodeRead, "", auditevent.OutcomeSuccess)}; !slices.Equal(g.audit.events, want) {
		t.Errorf("events = %+v,\nwant %+v", g.audit.events, want)
	}
}

func TestAvailability_RefusesAndAuditsAsTheEditDoes(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		want   error
		role   string
		caller schedule.Caller
	}{
		{caller: practitionerA1, role: roleA2, want: schedule.ErrForbidden},
		{caller: adminB, role: roleA2, want: schedule.ErrNotFound},
		{caller: patient, role: roleA2, want: schedule.ErrNotFound},
	} {
		f := newFixture()
		_, err := f.svc.Availability(t.Context(), tt.caller, tt.role)
		if !errors.Is(err, tt.want) || slices.Contains(f.calls, "ByRole") {
			t.Errorf("Availability(%s) as %+v = %v after %v; want %v without reading the schedule", tt.role, tt.caller, err, f.calls, tt.want)
		}
		if want := []auditevent.AuditEvent{event(tt.caller, auditevent.CodeRead, "", auditevent.OutcomeMinorFailure)}; !slices.Equal(f.audit.events, want) {
			t.Errorf("as %+v: events = %+v, want one refusal", tt.caller, f.audit.events)
		}
	}
}

func TestAvailability_HidesAScheduleOfAnotherClinicEvenWithoutTheDatabasePolicy(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.repo.leak = true
	f.repo.schedules[1].PractitionerRoleID = roleA1 // a row that names a role of clinic A but belongs to B

	got, err := f.svc.Availability(t.Context(), adminA, roleA1)
	if err != nil || got.ID != "" {
		t.Errorf("Availability() = %+v, %v; want clinic B's schedule not shown", got, err)
	}
}

func TestReadAndByActor_ShowTheDirectorysSchedulesWithoutAuditing(t *testing.T) {
	t.Parallel()
	f := newFixture()
	byID, err := f.svc.Read(t.Context(), scheduleB)
	if err != nil || byID.ID != scheduleB {
		t.Errorf("Read() = %+v, %v; want schedule B", byID, err)
	}
	byActor, err := f.svc.ByActor(t.Context(), roleA2)
	if err != nil || byActor.ID != scheduleA2 {
		t.Errorf("ByActor() = %+v, %v; want schedule A2", byActor, err)
	}
	if _, err := f.svc.ByActor(t.Context(), roleA1); !errors.Is(err, schedule.ErrNotFound) {
		t.Errorf("ByActor(a role without one) = %v, want ErrNotFound", err)
	}
	if want := []string{"SetDirectory", "ByID", "SetDirectory", "ByRole", "SetDirectory", "ByRole"}; !slices.Equal(f.calls, want) || len(f.audit.events) != 0 {
		t.Errorf("calls = %v with %d events, want %v and none", f.calls, len(f.audit.events), want)
	}
}

func TestForGeneration_LocksTheScheduleInItsClinic(t *testing.T) {
	t.Parallel()
	f := newFixture()
	got, err := f.svc.ForGeneration(t.Context(), clinicA, scheduleA2)
	if err != nil || got.ID != scheduleA2 || !slices.Equal(f.calls, []string{"SetOrganization", "Lock"}) {
		t.Errorf("ForGeneration() = %+v, %v after %v; want schedule A2 locked in clinic A", got, err, f.calls)
	}
	f.repo.leak = true
	if _, err := f.svc.ForGeneration(t.Context(), clinicA, scheduleB); !errors.Is(err, schedule.ErrNotFound) {
		t.Errorf("ForGeneration(another clinic's) = %v, want ErrNotFound", err)
	}
}

func TestListed_GivesTheDirectorysSchedules(t *testing.T) {
	t.Parallel()
	f := newFixture()
	got, err := f.svc.Listed(t.Context())
	if err != nil || len(got) != 2 || !slices.Equal(f.calls, []string{"SetDirectory", "Listed"}) {
		t.Errorf("Listed() = %d, %v after %v; want both in the directory scope", len(got), err, f.calls)
	}
	f.repo.err = errors.New("synthetic failure")
	if _, err := f.svc.Listed(t.Context()); err == nil {
		t.Error("Listed() error = nil, want the repository's")
	}
}
