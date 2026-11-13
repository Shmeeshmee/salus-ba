package clinicstats_test

import (
	"cmp"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const gp = "394814009" // SNOMED CT general practice, roleA1's specialty

// buildFixture is a Builder on fakes: clinics A and B in Sarajevo, roleA1 a general practitioner, roleA2 without
// a specialty.
type buildFixture struct {
	src  *sources
	repo *fakeBuildRepo
	tx   *fakeTx
	b    *clinicstats.Builder
}

func newBuildFixture(t *testing.T, now time.Time) *buildFixture {
	t.Helper()
	f := &buildFixture{
		src: &sources{
			starts: map[string][]appointment.Start{}, recorded: map[string][]condition.Recorded{},
			specialties: map[string]map[string]string{clinicA: {roleA1: gp, roleA2: ""}},
			locations:   []organization.Location{{ID: clinicA, TimeZone: "Europe/Sarajevo"}, {ID: clinicB, TimeZone: "Europe/Sarajevo"}},
			groups:      sampleGroups(t),
		},
		repo: newFakeBuildRepo(),
		tx:   &fakeTx{},
	}
	f.b = clinicstats.NewBuilder(clinicstats.BuildPorts{
		Repo: f.repo, Tx: f.tx, Clock: clock.Fixed{T: now}, Clinics: f.src, Appointments: f.src, Conditions: f.src,
		Specialties: f.src, Groups: f.src,
	})
	return f
}

func local(y int, m time.Month, d, h, minute int) time.Time {
	return time.Date(y, m, d, h, minute, 0, 0, sarajevo)
}

func sortedAppts(c []clinicstats.AppointmentCount) []clinicstats.AppointmentCount {
	c = slices.Clone(c)
	slices.SortFunc(c, func(a, b clinicstats.AppointmentCount) int {
		return cmp.Or(a.Day.Compare(b.Day), cmp.Compare(a.Hour, b.Hour), cmp.Compare(a.PractitionerRoleID, b.PractitionerRoleID), cmp.Compare(a.Status, b.Status))
	})
	return c
}

func sortedDiags(c []clinicstats.DiagnosisCount) []clinicstats.DiagnosisCount {
	c = slices.Clone(c)
	slices.SortFunc(c, func(a, b clinicstats.DiagnosisCount) int {
		return cmp.Or(a.Day.Compare(b.Day), cmp.Compare(a.Hour, b.Hour), cmp.Compare(a.PractitionerRoleID, b.PractitionerRoleID), cmp.Compare(a.Code, b.Code))
	})
	return c
}

// A refresh counts the clinic's local day of the instant it is given, in one transaction under the clinic's lock:
// appointments by the local hour of their start, role, the role's specialty now and status; confirmed Conditions by
// hour, role, specialty, group, subgroup and code. Nothing of the days around it.
func TestRefresh_CountsTheClinicsLocalDay(t *testing.T) {
	t.Parallel()
	f := newBuildFixture(t, local(2026, 11, 12, 9, 0))
	f.src.starts[clinicA] = []appointment.Start{
		{Start: local(2026, 11, 2, 23, 59), PractitionerRoleID: roleA1, Status: appointment.StatusFulfilled}, // the day before
		{Start: local(2026, 11, 3, 0, 0), PractitionerRoleID: roleA1, Status: appointment.StatusFulfilled},
		{Start: local(2026, 11, 3, 9, 0), PractitionerRoleID: roleA1, Status: appointment.StatusFulfilled},
		{Start: local(2026, 11, 3, 9, 30), PractitionerRoleID: roleA1, Status: appointment.StatusFulfilled},
		{Start: local(2026, 11, 3, 9, 45), PractitionerRoleID: roleA1, Status: appointment.StatusCancelled},
		{Start: local(2026, 11, 3, 14, 0), PractitionerRoleID: roleA2, Status: appointment.StatusNoShow},
		{Start: local(2026, 11, 3, 23, 30), PractitionerRoleID: roleA2, Status: appointment.StatusBooked},
		{Start: local(2026, 11, 4, 0, 0), PractitionerRoleID: roleA2, Status: appointment.StatusBooked}, // the day after
	}
	f.src.recorded[clinicA] = []condition.Recorded{
		{RecordedAt: local(2026, 11, 3, 9, 20), PractitionerRoleID: roleA1, Code: "J00"},
		{RecordedAt: local(2026, 11, 3, 9, 50), PractitionerRoleID: roleA1, Code: "J00"},
		{RecordedAt: local(2026, 11, 3, 9, 55), PractitionerRoleID: roleA1, Code: "J06.9"},
		{RecordedAt: local(2026, 11, 3, 14, 20), PractitionerRoleID: roleA2, Code: "I10"},
		{RecordedAt: local(2026, 11, 4, 0, 1), PractitionerRoleID: roleA2, Code: "I10"}, // the day after
	}

	// 23:30 UTC on 2 November is 00:30 on 3 November in Sarajevo.
	if err := f.b.Refresh(t.Context(), clinicA, time.Date(2026, 11, 2, 23, 30, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	d := day(2026, 11, 3)
	wantAppts := []clinicstats.AppointmentCount{
		{Day: d, Hour: 0, PractitionerRoleID: roleA1, SpecialtyCode: gp, Status: "fulfilled", Count: 1},
		{Day: d, Hour: 9, PractitionerRoleID: roleA1, SpecialtyCode: gp, Status: "cancelled", Count: 1},
		{Day: d, Hour: 9, PractitionerRoleID: roleA1, SpecialtyCode: gp, Status: "fulfilled", Count: 2},
		{Day: d, Hour: 14, PractitionerRoleID: roleA2, Status: "noshow", Count: 1},
		{Day: d, Hour: 23, PractitionerRoleID: roleA2, Status: "booked", Count: 1},
	}
	if got := sortedAppts(f.repo.appts[clinicA]); !slices.Equal(got, wantAppts) {
		t.Errorf("appointments\n got %+v\nwant %+v", got, wantAppts)
	}
	wantDiags := []clinicstats.DiagnosisCount{
		{Day: d, Hour: 9, PractitionerRoleID: roleA1, SpecialtyCode: gp, Group: "respiratory-infections", Subgroup: "common-cold", Code: "J00", Count: 2},
		{Day: d, Hour: 9, PractitionerRoleID: roleA1, SpecialtyCode: gp, Group: "respiratory-infections", Subgroup: "upper-airway-infections", Code: "J06.9", Count: 1},
		{Day: d, Hour: 14, PractitionerRoleID: roleA2, Group: "other", Code: "I10", Count: 1},
	}
	if got := sortedDiags(f.repo.diags[clinicA]); !slices.Equal(got, wantDiags) {
		t.Errorf("diagnoses\n got %+v\nwant %+v", got, wantDiags)
	}
	if f.tx.n != 1 || !slices.Equal(f.repo.locked, []string{clinicA}) {
		t.Errorf("%d transactions, locked %v; want one under clinic A's lock", f.tx.n, f.repo.locked)
	}
}

// On the day clocks go forward the local day has 23 hours: the refresh asks for exactly those, and the hour after
// 01:59 is 03.
func TestRefresh_ADaylightSavingDayHas23Hours(t *testing.T) {
	t.Parallel()
	f := newBuildFixture(t, local(2026, 4, 1, 9, 0))
	f.src.starts[clinicA] = []appointment.Start{
		{Start: time.Date(2026, 3, 29, 0, 30, 0, 0, time.UTC), PractitionerRoleID: roleA1, Status: appointment.StatusFulfilled}, // 01:30 CET
		{Start: time.Date(2026, 3, 29, 1, 30, 0, 0, time.UTC), PractitionerRoleID: roleA1, Status: appointment.StatusFulfilled}, // 03:30 CEST
		{Start: time.Date(2026, 3, 29, 21, 59, 0, 0, time.UTC), PractitionerRoleID: roleA1, Status: appointment.StatusFulfilled},
	}
	if err := f.b.Refresh(t.Context(), clinicA, local(2026, 3, 29, 12, 0)); err != nil {
		t.Fatal(err)
	}
	r := f.src.ranges[0]
	if !r.from.Equal(time.Date(2026, 3, 28, 23, 0, 0, 0, time.UTC)) || !r.to.Equal(time.Date(2026, 3, 29, 22, 0, 0, 0, time.UTC)) {
		t.Errorf("asked [%v, %v), want the 23 hours of 29 March in Sarajevo", r.from.UTC(), r.to.UTC())
	}
	var hours []int32
	for _, c := range sortedAppts(f.repo.appts[clinicA]) {
		hours = append(hours, c.Hour)
	}
	if !slices.Equal(hours, []int32{1, 3, 23}) {
		t.Errorf("hours %v, want 1, 3 and 23", hours)
	}
}

// A refresh replaces its day: run again it gives the same rows, and after a change at the source (a cancellation)
// it gives the new counts and nothing of the old.
func TestRefresh_IsIdempotent(t *testing.T) {
	t.Parallel()
	f := newBuildFixture(t, local(2026, 11, 12, 9, 0))
	f.src.starts[clinicA] = []appointment.Start{{Start: local(2026, 11, 3, 9, 0), PractitionerRoleID: roleA1, Status: appointment.StatusBooked}}
	at := local(2026, 11, 3, 9, 0)
	for range 2 {
		if err := f.b.Refresh(t.Context(), clinicA, at); err != nil {
			t.Fatal(err)
		}
	}
	if got := f.repo.appts[clinicA]; len(got) != 1 || got[0].Count != 1 || got[0].Status != "booked" {
		t.Errorf("after two refreshes %+v, want the one booked appointment", got)
	}
	f.src.starts[clinicA][0].Status = appointment.StatusCancelled
	if err := f.b.Refresh(t.Context(), clinicA, at); err != nil {
		t.Fatal(err)
	}
	if got := f.repo.appts[clinicA]; len(got) != 1 || got[0].Status != "cancelled" {
		t.Errorf("after the cancellation %+v, want it cancelled only", got)
	}
}

// A clinic the builds do not know, and a source that fails, refresh nothing.
func TestRefresh_FailsForAnUnknownClinicOrASourceError(t *testing.T) {
	t.Parallel()
	f := newBuildFixture(t, local(2026, 11, 12, 9, 0))
	if err := f.b.Refresh(t.Context(), "00000000-0000-4000-8000-0000000000ff", local(2026, 11, 3, 9, 0)); !errors.Is(err, clinicstats.ErrUnknownClinic) {
		t.Errorf("an unknown clinic: %v, want ErrUnknownClinic", err)
	}
	boom := errors.New("boom")
	f.src.err = boom
	if err := f.b.Refresh(t.Context(), clinicA, local(2026, 11, 3, 9, 0)); !errors.Is(err, boom) {
		t.Errorf("a failing source: %v, want it", err)
	}
	if len(f.repo.appts[clinicA]) != 0 {
		t.Errorf("stored %+v", f.repo.appts[clinicA])
	}
}

// The nightly pass recomputes the last 35 local days of every clinic, today included, one transaction each.
func TestRefreshRecent_RecomputesTheLast35DaysOfEveryClinic(t *testing.T) {
	t.Parallel()
	f := newBuildFixture(t, time.Date(2026, 11, 12, 1, 30, 0, 0, time.UTC))
	f.src.starts[clinicB] = []appointment.Start{
		{Start: local(2026, 10, 8, 23, 0), PractitionerRoleID: roleA1, Status: appointment.StatusFulfilled}, // day 36
		{Start: local(2026, 10, 9, 0, 0), PractitionerRoleID: roleA1, Status: appointment.StatusFulfilled},
		{Start: local(2026, 11, 12, 23, 59), PractitionerRoleID: roleA1, Status: appointment.StatusBooked},
	}
	if err := f.b.RefreshRecent(t.Context()); err != nil {
		t.Fatal(err)
	}
	want := []timeRange{
		{org: clinicA, from: local(2026, 10, 9, 0, 0), to: local(2026, 11, 13, 0, 0)},
		{org: clinicB, from: local(2026, 10, 9, 0, 0), to: local(2026, 11, 13, 0, 0)},
	}
	if len(f.src.ranges) != 2 || f.tx.n != 2 {
		t.Fatalf("asked %v in %d transactions, want %v in two", f.src.ranges, f.tx.n, want)
	}
	for i, r := range f.src.ranges {
		if r.org != want[i].org || !r.from.Equal(want[i].from) || !r.to.Equal(want[i].to) {
			t.Errorf("asked %v, want %v", r, want[i])
		}
	}
	if got := sortedAppts(f.repo.appts[clinicB]); len(got) != 2 || !got[0].Day.Equal(day(2026, 10, 9)) || !got[1].Day.Equal(day(2026, 11, 12)) {
		t.Errorf("clinic B: %+v, want 9 October and 12 November", got)
	}
}
