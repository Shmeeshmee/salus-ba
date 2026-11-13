package clinicstats_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/clinicstats"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/terminology"
)

const (
	clinicA = "00000000-0000-4000-8000-00000000000a"
	clinicB = "00000000-0000-4000-8000-00000000000b"
	roleA1  = "00000000-0000-4000-8000-0000000000a1"
	roleA2  = "00000000-0000-4000-8000-0000000000a2"
)

// sarajevo is the clinics' time zone in the fakes.
var sarajevo = mustZone("Europe/Sarajevo")

func mustZone(name string) *time.Location {
	l, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return l
}

// fakeTx runs fn and counts the transactions.
type fakeTx struct{ n int }

func (f *fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.n++
	return fn(ctx)
}

// sources are the fake build sources: every appointment and Condition of each clinic, the roles' specialties and
// the clinics' zones. They answer a range query with the rows in [from, to) and keep the ranges asked.
type sources struct {
	err         error
	starts      map[string][]appointment.Start
	recorded    map[string][]condition.Recorded
	specialties map[string]map[string]string
	ranges      []timeRange
	locations   []organization.Location
	groups      terminology.Groups
}

type timeRange struct {
	from, to time.Time
	org      string
}

func (s *sources) Starts(_ context.Context, org string, from, to time.Time, key []byte) ([]appointment.Start, error) {
	if key != nil {
		panic("clinicstats asked for patient pseudonyms")
	}
	s.ranges = append(s.ranges, timeRange{from: from, to: to, org: org})
	var out []appointment.Start
	for _, a := range s.starts[org] {
		if !a.Start.Before(from) && a.Start.Before(to) {
			out = append(out, a)
		}
	}
	return out, s.err
}

func (s *sources) Recorded(_ context.Context, org string, from, to time.Time, key []byte) ([]condition.Recorded, error) {
	if key != nil {
		panic("clinicstats asked for patient pseudonyms")
	}
	var out []condition.Recorded
	for _, c := range s.recorded[org] {
		if !c.RecordedAt.Before(from) && c.RecordedAt.Before(to) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *sources) Specialties(_ context.Context, org string) (map[string]string, error) {
	return s.specialties[org], nil
}

func (s *sources) Locations(context.Context) ([]organization.Location, error) {
	return s.locations, nil
}

func (s *sources) Groups(context.Context) (terminology.Groups, error) { return s.groups, nil }

// fakeBuildRepo keeps the rows of each clinic-day, replaced as a refresh replaces them, and the clinics locked.
type fakeBuildRepo struct {
	appts  map[string][]clinicstats.AppointmentCount // by clinic
	diags  map[string][]clinicstats.DiagnosisCount
	locked []string
}

func newFakeBuildRepo() *fakeBuildRepo {
	return &fakeBuildRepo{appts: map[string][]clinicstats.AppointmentCount{}, diags: map[string][]clinicstats.DiagnosisCount{}}
}

func (f *fakeBuildRepo) Lock(_ context.Context, org string) error {
	f.locked = append(f.locked, org)
	return nil
}

func (f *fakeBuildRepo) Replace(_ context.Context, org string, first, last time.Time, a []clinicstats.AppointmentCount, d []clinicstats.DiagnosisCount) error {
	in := func(day time.Time) bool { return !day.Before(first) && !day.After(last) }
	f.appts[org] = slices.DeleteFunc(f.appts[org], func(c clinicstats.AppointmentCount) bool { return in(c.Day) })
	f.diags[org] = slices.DeleteFunc(f.diags[org], func(c clinicstats.DiagnosisCount) bool { return in(c.Day) })
	f.appts[org] = append(f.appts[org], a...)
	f.diags[org] = append(f.diags[org], d...)
	return nil
}

// sampleGroups is a slice of appendix A: respiratory infections with two subgroups, and "other".
func sampleGroups(t testing.TB) terminology.Groups {
	t.Helper()
	g, err := terminology.NewGroups(
		[]terminology.DiagnosisGroup{
			{Code: "respiratory-infections"}, {Code: "common-cold", Parent: "respiratory-infections"},
			{Code: "upper-airway-infections", Parent: "respiratory-infections"}, {Code: "other"},
		},
		[]terminology.GroupRange{
			{Group: "respiratory-infections", From: "J00", To: "J22"},
			{Group: "common-cold", From: "J00", To: "J00"},
			{Group: "upper-airway-infections", From: "J01", To: "J06"},
		})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// fakeAuditor keeps the events recorded.
type fakeAuditor struct{ events []auditevent.AuditEvent }

func (f *fakeAuditor) Record(_ context.Context, e auditevent.AuditEvent) error {
	f.events = append(f.events, e)
	return nil
}

// day is the date y-m-d as the aggregates hold it: midnight UTC.
func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
