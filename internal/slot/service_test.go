package slot_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const (
	clinicA    = "a0000000-0000-4000-8000-00000000000a"
	clinicB    = "b0000000-0000-4000-8000-00000000000b"
	roleA1     = "a4000000-0000-4000-8000-0000000000a1"
	roleB      = "b4000000-0000-4000-8000-0000000000b1"
	scheduleA1 = "a5000000-0000-4000-8000-0000000000a1"
	scheduleB  = "b5000000-0000-4000-8000-0000000000b1"
	slotA1     = "a6000000-0000-4000-8000-0000000000a1"
)

// now is a Wednesday, 10:30 in Sarajevo.
var now = time.Date(2026, 11, 4, 9, 30, 0, 0, time.UTC)

type txKey struct{}

type fakeTransactor struct{}

func (fakeTransactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

type log struct{ calls []string }

func (l *log) seen(ctx context.Context, call string) {
	if ctx.Value(txKey{}) == nil {
		call = "outside:" + call
	}
	l.calls = append(l.calls, call)
}

// fakeRepository records what it is asked and answers with fixed counts.
type fakeRepository struct {
	*log
	err      error
	holdErr  error
	clinic   string
	after    time.Time
	from, to time.Time
	keep     []slot.Period
	inserted []slot.Period
	released bool
}

func (r *fakeRepository) SetOrganization(ctx context.Context, id string) error {
	r.seen(ctx, "SetOrganization:"+id)
	r.clinic = id
	return nil
}

func (r *fakeRepository) SetDirectory(ctx context.Context) error {
	r.seen(ctx, "SetDirectory")
	return nil
}

func (r *fakeRepository) DeleteFreeExcept(ctx context.Context, scheduleID string, after time.Time, keep []slot.Period) (int64, error) {
	r.seen(ctx, "DeleteFreeExcept:"+scheduleID)
	r.after, r.keep = after, keep
	return 2, r.err
}

func (r *fakeRepository) Insert(ctx context.Context, scheduleID, org string, periods []slot.Period) (int64, error) {
	r.seen(ctx, "Insert:"+scheduleID+":"+org)
	r.inserted = periods
	return int64(len(periods)), nil
}

func (r *fakeRepository) Search(ctx context.Context, scheduleID string, from, to time.Time) ([]slot.Slot, error) {
	r.seen(ctx, "Search:"+scheduleID)
	r.from, r.to = from, to
	return []slot.Slot{{ID: slotA1, ScheduleID: scheduleID, Status: slot.StatusFree}}, r.err
}

func (r *fakeRepository) ByID(ctx context.Context, id string) (slot.Slot, error) {
	r.seen(ctx, "ByID:"+id)
	if r.err != nil {
		return slot.Slot{}, r.err
	}
	if id != slotA1 {
		return slot.Slot{}, slot.ErrUnavailable
	}
	return slot.Slot{ID: id, ScheduleID: scheduleA1, OrganizationID: clinicA, Status: slot.StatusFree, Period: slot.Period{Start: now, End: now.Add(time.Hour)}}, nil
}

func (r *fakeRepository) Hold(ctx context.Context, id string, at time.Time) (slot.Slot, error) {
	r.seen(ctx, "Hold:"+id)
	if r.holdErr != nil {
		return slot.Slot{}, r.holdErr
	}
	return slot.Slot{ID: id, OrganizationID: r.clinic, Status: slot.StatusBusy, LastUpdated: at}, nil
}

func (r *fakeRepository) Release(ctx context.Context, id string, at time.Time) (bool, error) {
	r.seen(ctx, "Release:"+id)
	r.after = at
	return r.released, r.err
}

func (r *fakeRepository) DeleteFreeEndedBefore(ctx context.Context, before time.Time) (int64, error) {
	r.seen(ctx, "DeleteFreeEndedBefore")
	r.after = before
	return 5, r.err
}

// fakeSchedules holds schedule A1 (Wednesdays 09:00 to 12:00 by the hour, Mondays 08:00 to 09:00 by the
// half hour) and schedule B, and fails for failing.
type fakeSchedules struct {
	*log
	listErr   error
	failing   string
	schedules []schedule.Schedule
}

func (s *fakeSchedules) ForGeneration(ctx context.Context, org, id string) (schedule.Schedule, error) {
	s.seen(ctx, "ForGeneration:"+id)
	if id == s.failing {
		return schedule.Schedule{}, errors.New("synthetic failure")
	}
	for _, sc := range s.schedules {
		if sc.ID == id && sc.OrganizationID == org {
			return sc, nil
		}
	}
	return schedule.Schedule{}, schedule.ErrNotFound
}

func (s *fakeSchedules) Read(ctx context.Context, id string) (schedule.Schedule, error) {
	s.seen(ctx, "Read:"+id)
	for _, sc := range s.schedules {
		if sc.ID == id && sc.ID != s.failing {
			return sc, nil
		}
	}
	return schedule.Schedule{}, schedule.ErrNotFound
}

func (s *fakeSchedules) Listed(ctx context.Context) ([]schedule.Schedule, error) {
	s.seen(ctx, "Listed")
	return s.schedules, s.listErr
}

type fakeClinics struct {
	*log
	zones map[string]string
}

func (c fakeClinics) Read(ctx context.Context, caller organization.Caller, id string) (organization.Organization, error) {
	c.seen(ctx, "Clinics.Read:"+id)
	if caller.OrganizationID != id {
		return organization.Organization{}, organization.ErrNotFound
	}
	return organization.Organization{ID: id, TimeZone: c.zones[id]}, nil
}

type fixture struct {
	*log
	repo      *fakeRepository
	schedules *fakeSchedules
	clinics   fakeClinics
}

func newFixture() fixture {
	l := &log{}
	return fixture{
		log:  l,
		repo: &fakeRepository{log: l},
		schedules: &fakeSchedules{log: l, schedules: []schedule.Schedule{
			{ID: scheduleA1, OrganizationID: clinicA, PractitionerRoleID: roleA1, SlotMinutes: 60, Rules: []schedule.Rule{
				{Day: 1, Start: "08:00", End: "09:00"}, {Day: 3, Start: "09:00", End: "12:00"},
			}},
			{ID: scheduleB, OrganizationID: clinicB, PractitionerRoleID: roleB, SlotMinutes: 30, Rules: []schedule.Rule{{Day: 2, Start: "10:00", End: "11:00"}}},
		}},
		clinics: fakeClinics{log: l, zones: map[string]string{clinicA: "Europe/Sarajevo", clinicB: "Europe/Lisbon"}},
	}
}

func (f fixture) service(at time.Time) *slot.Service {
	return slot.NewService(f.repo, fakeTransactor{}, clock.Fixed{T: at}, f.schedules, f.clinics)
}

func utc(month time.Month, day, hour, minute int) time.Time {
	return time.Date(2026, month, day, hour, minute, 0, 0, time.UTC)
}

func TestGenerate_GivesTheRulesSlotsAfterNowAndBefore28DaysAheadInTheClinicsZone(t *testing.T) {
	t.Parallel()
	f := newFixture()
	f.schedules.schedules[0].SlotMinutes = 30
	f.schedules.schedules[0].Rules[1].End = "11:00" // Wednesdays 09:00 to 11:00, Mondays 08:00 to 09:00

	got, err := f.service(now).Generate(t.Context(), clinicA, scheduleA1)

	// Sarajevo is UTC+1 in November and December. Today's slots (08:00Z to 09:30Z) are not after now; 28 days
	// ahead is Wednesday 2 December 09:30Z, so that day ends with 09:00Z.
	var want []slot.Period
	add := func(month time.Month, day, hour, minute int) {
		start := utc(month, day, hour, minute)
		want = append(want, slot.Period{Start: start, End: start.Add(30 * time.Minute)})
	}
	for _, monday := range []int{9, 16, 23, 30} {
		add(11, monday, 7, 0)
		add(11, monday, 7, 30)
		if monday < 30 {
			for _, m := range []int{0, 30, 60, 90} {
				add(11, monday+2, 8, m)
			}
		}
	}
	add(12, 2, 8, 0)
	add(12, 2, 8, 30)
	add(12, 2, 9, 0)

	if !slices.Equal(f.repo.inserted, want) {
		t.Errorf("inserted\n%v,\nwant\n%v", f.repo.inserted, want)
	}
	if !slices.Equal(f.repo.keep, want) || !f.repo.after.Equal(now) {
		t.Errorf("deleted the free slots after %v except %d periods, want after now except the same %d", f.repo.after, len(f.repo.keep), len(want))
	}
	if wantGot := (slot.Generated{Schedules: 1, Inserted: int64(len(want)), Deleted: 2}); got != wantGot || err != nil {
		t.Errorf("Generate() = %+v, %v; want %+v", got, err, wantGot)
	}
}

func TestGenerate_SkipsTheMissingHourAndTakesTheFirstOfTheRepeatedOne(t *testing.T) {
	t.Parallel()
	sunday := []schedule.Rule{{Day: 7, Start: "01:00", End: "04:00"}}
	tests := []struct {
		name string
		now  time.Time
		want []time.Time // the first Sunday's starts
	}{
		{
			// 28 March 2027: 02:00 CET becomes 03:00 CEST; 02:00 and 02:30 do not exist.
			name: "spring", now: time.Date(2027, 3, 27, 12, 0, 0, 0, time.UTC),
			want: []time.Time{
				time.Date(2027, 3, 28, 0, 0, 0, 0, time.UTC),  // 01:00 CET
				time.Date(2027, 3, 28, 0, 30, 0, 0, time.UTC), // 01:30 CET
				time.Date(2027, 3, 28, 1, 0, 0, 0, time.UTC),  // 03:00 CEST
				time.Date(2027, 3, 28, 1, 30, 0, 0, time.UTC), // 03:30 CEST
			},
		},
		{
			// 31 October 2027: 03:00 CEST becomes 02:00 CET; 02:00 and 02:30 happen twice, the first is taken.
			name: "autumn", now: time.Date(2027, 10, 30, 12, 0, 0, 0, time.UTC),
			want: []time.Time{
				time.Date(2027, 10, 30, 23, 0, 0, 0, time.UTC),  // 01:00 CEST
				time.Date(2027, 10, 30, 23, 30, 0, 0, time.UTC), // 01:30 CEST
				time.Date(2027, 10, 31, 0, 0, 0, 0, time.UTC),   // 02:00 CEST, the first
				time.Date(2027, 10, 31, 0, 30, 0, 0, time.UTC),  // 02:30 CEST, the first
				time.Date(2027, 10, 31, 2, 0, 0, 0, time.UTC),   // 03:00 CET
				time.Date(2027, 10, 31, 2, 30, 0, 0, time.UTC),  // 03:30 CET
			},
		},
	}
	for _, tt := range tests {
		f := newFixture()
		f.schedules.schedules[0].SlotMinutes, f.schedules.schedules[0].Rules = 30, sunday

		if _, err := f.service(tt.now).Generate(t.Context(), clinicA, scheduleA1); err != nil {
			t.Fatalf("%s: Generate() = %v", tt.name, err)
		}
		var first []time.Time
		for _, p := range f.repo.inserted {
			if p.End.Sub(p.Start) != 30*time.Minute {
				t.Errorf("%s: %v lasts %v, want 30 minutes", tt.name, p.Start, p.End.Sub(p.Start))
			}
			if p.Start.Before(tt.now.Add(36 * time.Hour)) {
				first = append(first, p.Start)
			}
		}
		if !slices.EqualFunc(first, tt.want, time.Time.Equal) {
			t.Errorf("%s: the first Sunday's slots start at %v, want %v", tt.name, first, tt.want)
		}
		if n := len(f.repo.inserted); n != 6*3+len(tt.want) { // three more Sundays of six, as the window ends before the fifth
			t.Errorf("%s: %d slots, want %d", tt.name, n, 6*3+len(tt.want))
		}
	}
}

func TestGenerate_ReadsTheLockedScheduleAndReplacesFreeSlotsInOneTransactionInTheClinic(t *testing.T) {
	t.Parallel()
	f := newFixture()
	if _, err := f.service(now).Generate(t.Context(), clinicA, scheduleA1); err != nil {
		t.Fatal(err)
	}
	want := []string{"SetOrganization:" + clinicA, "ForGeneration:" + scheduleA1, "Clinics.Read:" + clinicA,
		"DeleteFreeExcept:" + scheduleA1, "Insert:" + scheduleA1 + ":" + clinicA}
	if !slices.Equal(f.calls, want) {
		t.Errorf("calls = %v,\nwant %v", f.calls, want)
	}
}

func TestGenerate_FailsWithoutTheScheduleOrAKnownZone(t *testing.T) {
	t.Parallel()
	f := newFixture()
	if _, err := f.service(now).Generate(t.Context(), clinicB, scheduleA1); !errors.Is(err, schedule.ErrNotFound) {
		t.Errorf("Generate(A1 in clinic B) = %v, want schedule.ErrNotFound", err)
	}
	f.clinics.zones[clinicA] = "Mars/Olympus"
	if _, err := f.service(now).Generate(t.Context(), clinicA, scheduleA1); err == nil || slices.ContainsFunc(f.calls, isWrite) {
		t.Errorf("Generate() in an unknown zone = %v after %v, want an error before any write", err, f.calls)
	}
	g := newFixture()
	g.repo.err = errors.New("synthetic failure")
	if _, err := g.service(now).Generate(t.Context(), clinicA, scheduleA1); err == nil || slices.ContainsFunc(g.calls, isInsert) {
		t.Errorf("Generate() when the delete fails = %v after %v, want the error and no insert", err, g.calls)
	}
}

func isWrite(call string) bool {
	return len(call) > 6 && (call[:6] == "Delete" || call[:6] == "Insert")
}

func isInsert(call string) bool { return len(call) > 6 && call[:6] == "Insert" }

func TestGenerateAll_GeneratesEveryListedScheduleInItsOwnClinicAndGoesOnAfterAFailure(t *testing.T) {
	t.Parallel()
	f := newFixture()
	got, err := f.service(now).GenerateAll(t.Context())
	if err != nil || got.Schedules != 2 || got.Deleted != 4 || got.Inserted == 0 {
		t.Errorf("GenerateAll() = %+v, %v; want both schedules", got, err)
	}
	if !slices.Contains(f.calls, "Insert:"+scheduleB+":"+clinicB) || !slices.Contains(f.calls, "SetOrganization:"+clinicB) {
		t.Errorf("calls = %v, want schedule B generated in clinic B", f.calls)
	}

	g := newFixture()
	g.schedules.failing = scheduleA1
	got, err = g.service(now).GenerateAll(t.Context())
	if err == nil || got.Schedules != 1 || !slices.Contains(g.calls, "Insert:"+scheduleB+":"+clinicB) {
		t.Errorf("GenerateAll() with A1 failing = %+v, %v; want B generated and the error", got, err)
	}

	h := newFixture()
	h.schedules.listErr = errors.New("synthetic failure")
	if _, err := h.service(now).GenerateAll(t.Context()); err == nil {
		t.Error("GenerateAll() error = nil when the list fails")
	}
}

func TestSearch_GivesTheDirectorysFreeFutureSlots(t *testing.T) {
	t.Parallel()
	f := newFixture()
	got, err := f.service(now).Search(t.Context(), scheduleA1, now.Add(-time.Hour), now.Add(7*24*time.Hour))
	if err != nil || len(got) != 1 || !f.repo.from.Equal(now) || !f.repo.to.Equal(now.Add(7*24*time.Hour)) {
		t.Errorf("Search() = %v, %v from %v to %v; want the slots from now on", got, err, f.repo.from, f.repo.to)
	}
	if want := []string{"SetDirectory", "Search:" + scheduleA1}; !slices.Equal(f.calls, want) {
		t.Errorf("calls = %v, want %v", f.calls, want)
	}

	g := newFixture()
	if got, err := g.service(now).Search(t.Context(), scheduleA1, now.Add(-2*time.Hour), now.Add(-time.Hour)); err != nil || len(got) != 0 || len(g.calls) != 0 {
		t.Errorf("Search() of the past = %v, %v after %v; want none without a query", got, err, g.calls)
	}
	h := newFixture()
	h.repo.err = errors.New("synthetic failure")
	if _, err := h.service(now).Search(t.Context(), scheduleA1, now, now.Add(time.Hour)); err == nil {
		t.Error("Search() error = nil when the query fails")
	}
}

func TestHoldAndRelease_ChangeTheSlotInTheCallersTransactionInTheClinic(t *testing.T) {
	t.Parallel()
	f := newFixture()
	got, err := f.service(now).Hold(t.Context(), clinicA, slotA1)
	if err != nil || got.ID != slotA1 || got.Status != slot.StatusBusy {
		t.Errorf("Hold() = %+v, %v; want slot A1 busy", got, err)
	}
	f.repo.released = true
	if freed, err := f.service(now).Release(t.Context(), clinicA, slotA1); err != nil || !freed || !f.repo.after.Equal(now) {
		t.Errorf("Release() = %v, %v at %v; want it freed now", freed, err, f.repo.after)
	}
	want := []string{"SetOrganization:" + clinicA, "Hold:" + slotA1, "SetOrganization:" + clinicA, "Release:" + slotA1}
	if !slices.Equal(f.calls, want) {
		t.Errorf("calls = %v, want %v", f.calls, want)
	}

	g := newFixture()
	g.repo.holdErr = slot.ErrUnavailable
	if _, err := g.service(now).Hold(t.Context(), clinicA, slotA1); !errors.Is(err, slot.ErrUnavailable) {
		t.Errorf("Hold() of a taken slot = %v, want ErrUnavailable", err)
	}
	g.repo.err = errors.New("synthetic failure")
	if _, err := g.service(now).Release(t.Context(), clinicA, slotA1); err == nil {
		t.Error("Release() error = nil when the update fails")
	}
}

func TestPurger_DeletesFreeSlotsThatEndedBeforeTheCutOff(t *testing.T) {
	t.Parallel()
	f := newFixture()
	n, err := slot.NewPurger(f.repo).Purge(t.Context(), now)
	if n != 5 || err != nil || !f.repo.after.Equal(now) || !slices.Equal(f.calls, []string{"outside:DeleteFreeEndedBefore"}) {
		t.Errorf("Purge() = %d, %v after %v; want the repository's count for the cut-off", n, err, f.calls)
	}
}
