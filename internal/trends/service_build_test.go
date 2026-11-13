package trends_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/internal/trends"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

// fakeSources serves synthetic Conditions and appointments per clinic, pseudonymised under the key as the real
// sources do (any keyed hash will do for distinct counts).
type fakeSources struct {
	recorded map[string][]condition.Recorded // by clinic, Patient holding a plain id in its first byte
	starts   map[string][]appointment.Start
	roles    map[string]map[string]string
	locs     []organization.Location
	asked    []string // the clinics read
	keys     [][]byte
}

func pseudo(key []byte, p [32]byte) [32]byte {
	return sha256.Sum256(append(slices.Clone(key), p[:]...))
}

func (f *fakeSources) Recorded(_ context.Context, org string, from, to time.Time, key []byte) ([]condition.Recorded, error) {
	f.asked, f.keys = append(f.asked, org), append(f.keys, slices.Clone(key))
	var out []condition.Recorded
	for _, r := range f.recorded[org] {
		if !r.RecordedAt.Before(from) && r.RecordedAt.Before(to) {
			r.Patient = pseudo(key, r.Patient)
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeSources) Fulfilled(_ context.Context, org string, from, to time.Time, key []byte) ([]appointment.Start, error) {
	f.asked = append(f.asked, org)
	var out []appointment.Start
	for _, s := range f.starts[org] {
		if s.Status == appointment.StatusFulfilled && !s.Start.Before(from) && s.Start.Before(to) {
			s.Patient = pseudo(key, s.Patient)
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeSources) Specialties(_ context.Context, org string) (map[string]string, error) {
	return f.roles[org], nil
}

func (f *fakeSources) Locations(context.Context) ([]organization.Location, error) { return f.locs, nil }

// fakeCatalog groups J codes as respiratory-infections (J00 as common-cold) and I10 as high-blood-pressure.
type fakeCatalog struct{}

func (fakeCatalog) Groups(context.Context) (terminology.Groups, error) {
	return terminology.NewGroups([]terminology.DiagnosisGroup{
		{Code: "respiratory-infections"}, {Code: "common-cold", Parent: "respiratory-infections"}, {Code: "high-blood-pressure"}, {Code: "other"},
	}, []terminology.GroupRange{
		{Group: "respiratory-infections", From: "J00", To: "J22"}, {Group: "common-cold", From: "J00", To: "J00"},
		{Group: "high-blood-pressure", From: "I10", To: "I15"},
	})
}

func (fakeCatalog) AllPlaces(context.Context) ([]terminology.Place, error) {
	return []terminology.Place{
		{ID: "BA-BIH", Kind: "entity"}, {ID: "BA-09", Kind: "region", Parent: "BA-BIH"}, {ID: "BA-01", Kind: "region", Parent: "BA-BIH"},
		{ID: "BA-SRP", Kind: "region"}, {ID: "Q1", Kind: "city", Parent: "BA-09"}, {ID: "Q2", Kind: "city", Parent: "BA-09"},
		{ID: "Q3", Kind: "city", Parent: "BA-01"}, {ID: "Q4", Kind: "city", Parent: "BA-SRP"},
	}, nil
}

// memRepo keeps the periods, cells and flags in memory; written only inside a transaction.
type memRepo struct {
	periods map[string]trends.Period
	cells   map[string][]trends.Row
	flags   map[string][]trends.Flag
	err     error
}

func newMemRepo() *memRepo {
	return &memRepo{periods: map[string]trends.Period{}, cells: map[string][]trends.Row{}, flags: map[string][]trends.Flag{}}
}

func (m *memRepo) Built(_ context.Context, p string) (bool, error) {
	_, ok := m.periods[p]
	return ok, m.err
}

func (m *memRepo) SavePeriod(ctx context.Context, p trends.Period) error {
	if ctx.Value(txKey{}) == nil {
		return errors.New("outside a transaction")
	}
	if _, ok := m.periods[p.Period]; ok {
		return trends.ErrFrozen
	}
	m.periods[p.Period] = p
	return nil
}

func (m *memRepo) SaveCells(_ context.Context, p string, rows []trends.Row) error {
	m.cells[p] = append(m.cells[p], rows...)
	return nil
}

func (m *memRepo) SaveFlags(_ context.Context, p string, flags []trends.Flag) error {
	m.flags[p] = append(m.flags[p], flags...)
	return nil
}

func (m *memRepo) Periods(_ context.Context, kind string) ([]trends.Period, error) {
	var out []trends.Period
	for _, p := range m.periods {
		if p.Kind == kind {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b trends.Period) int { return compare(a.Period, b.Period) })
	return out, m.err
}

func compare(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func (m *memRepo) Cells(_ context.Context, table, from, to string) ([]trends.Row, error) {
	var out []trends.Row
	for p, rows := range m.cells {
		if p < from || p > to {
			continue
		}
		for _, r := range rows {
			if r.Table == table {
				out = append(out, r)
			}
		}
	}
	return out, m.err
}

func (m *memRepo) Flags(_ context.Context, weeks []string) ([]trends.Flag, error) {
	var out []trends.Flag
	for _, w := range weeks {
		out = append(out, m.flags[w]...)
	}
	return out, m.err
}

func (m *memRepo) DeleteCellsBefore(context.Context, time.Time) (int64, error) { return 0, nil }
func (m *memRepo) DeleteFlagsBefore(context.Context, time.Time) (int64, error) { return 0, nil }

type txKey struct{}

type fakeTx struct{}

func (fakeTx) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(context.WithValue(ctx, txKey{}, true))
}

type fakeAudit struct {
	err    error
	events []auditevent.AuditEvent
}

func (a *fakeAudit) Record(_ context.Context, e auditevent.AuditEvent) error {
	if a.err != nil {
		return a.err
	}
	a.events = append(a.events, e)
	return nil
}

var since = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// clinicAt is a contributing clinic in a city of region since the start of 2026.
func clinicAt(id, region, city string) organization.Location {
	return organization.Location{ID: id, TimeZone: "Europe/Sarajevo", Region: region, City: city, Contributes: true, LocationSince: since, ContributesSince: since}
}

func patientID(n int) [32]byte {
	var p [32]byte
	copy(p[:], fmt.Sprintf("patient-%d", n))
	return p
}

// diagnosesOf are n patients of a clinic (numbered from first) with code, recorded on day of October 2026.
func diagnosesOf(first, n int, code, role string, at time.Time) []condition.Recorded {
	out := make([]condition.Recorded, n)
	for i := range out {
		out[i] = condition.Recorded{RecordedAt: at, Patient: patientID(first + i), PractitionerRoleID: role, Code: code}
	}
	return out
}

func newBuild(src *fakeSources, repo *memRepo, audit *fakeAudit, now time.Time) *trends.Builder {
	return trends.NewBuilder(trends.BuildPorts{
		Diagnoses: src, Appointments: src, Specialties: src, Locations: src, Catalog: fakeCatalog{}, Repo: repo, Tx: fakeTx{},
		Audit: audit, Clock: clock.Fixed{T: now}, Agent: "trends_build", K: 10,
	})
}

var october = time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)

// fourClinics are four clinics in Sarajevo's cantons with 12 patients each of J06.9 in October.
func fourClinics() *fakeSources {
	src := &fakeSources{recorded: map[string][]condition.Recorded{}, roles: map[string]map[string]string{}}
	for i, city := range []string{"Q1", "Q1", "Q2", "Q2"} {
		id := fmt.Sprintf("clinic-%d", i)
		src.locs = append(src.locs, clinicAt(id, "BA-09", city))
		src.recorded[id] = diagnosesOf(i*100, 12, "J06.9", "role-"+id, october)
		src.roles[id] = map[string]string{"role-" + id: "419772000"}
	}
	return src
}

func TestBuild_RefusesAnOpenPeriodAndAFrozenOne(t *testing.T) {
	src, repo := fourClinics(), newMemRepo()
	for period, now := range map[string]time.Time{
		"2026-10": time.Date(2026, 11, 7, 23, 0, 0, 0, time.UTC), "2026-Q4": time.Date(2027, 1, 7, 0, 0, 0, 0, time.UTC),
		"2026-W44": time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC), "2026-13": october, "2026-W54": october, "x": october,
	} {
		if _, err := newBuild(src, repo, &fakeAudit{}, now).Build(t.Context(), period); !errors.Is(err, trends.ErrInvalidPeriod) {
			t.Errorf("Build(%s) at %v = %v, want ErrInvalidPeriod", period, now, err)
		}
	}
	if len(src.asked) != 0 {
		t.Errorf("sources read %v, want none", src.asked)
	}
	due := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)
	if _, err := newBuild(src, repo, &fakeAudit{}, due).Build(t.Context(), "2026-10"); err != nil {
		t.Fatalf("Build on the 8th = %v", err)
	}
	before := slices.Clone(repo.cells["2026-10"])
	src.recorded["clinic-0"] = append(src.recorded["clinic-0"], diagnosesOf(900, 40, "I10", "role-clinic-0", october)...)
	if _, err := newBuild(src, repo, &fakeAudit{}, due.AddDate(0, 1, 0)).Build(t.Context(), "2026-10"); !errors.Is(err, trends.ErrFrozen) {
		t.Errorf("a second build = %v, want ErrFrozen", err)
	}
	if !slices.Equal(repo.cells["2026-10"], before) {
		t.Errorf("a later build changed the frozen month")
	}
}

// A month build counts the clinics that kept their location and their contribution for the whole month, with one
// new key per build, writes G×D, G×S and S×D, and audits its read by count.
func TestBuild_AMonthOfTheClinicsThatCount(t *testing.T) {
	src, repo, audit := fourClinics(), newMemRepo(), &fakeAudit{}
	moved := clinicAt("moved", "BA-01", "Q3")
	moved.LocationSince = time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	out := clinicAt("out", "BA-09", "Q1")
	out.Contributes = false
	optedBack := clinicAt("back", "BA-09", "Q1")
	optedBack.ContributesSince = time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	src.locs = append(src.locs, moved, out, optedBack, organization.Location{ID: "nowhere", TimeZone: "UTC", Contributes: true, ContributesSince: since})
	s, err := newBuild(src, repo, audit, time.Date(2026, 11, 8, 4, 0, 0, 0, time.UTC)).Build(t.Context(), "2026-10")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"clinic-0", "clinic-1", "clinic-2", "clinic-3"}; !slices.Equal(src.asked, want) {
		t.Errorf("read %v, want only the clinics that count all month", src.asked)
	}
	for _, k := range src.keys {
		if len(k) != 32 || !slices.Equal(k, src.keys[0]) {
			t.Errorf("keys %x, want one 32-byte key for the build", src.keys)
		}
	}
	tables := map[string]bool{}
	var total trends.Row
	for _, r := range repo.cells["2026-10"] {
		tables[r.Table] = true
		if r.Table == trends.TableGD && r.Entity == "" && r.Group == "" && r.Specialty == "" {
			total = r
		}
	}
	if !tables[trends.TableGD] || !tables[trends.TableGS] || !tables[trends.TableSD] || total.Suppressed || total.Count != 50 {
		t.Errorf("tables %v, total %+v; want the three tables and 48 patients published as 50", tables, total)
	}
	if p := repo.periods["2026-10"]; p.Status != trends.StatusPublished || p.Kind != trends.KindMonth || p.MinCount != 10 {
		t.Errorf("period %+v, want a published month", p)
	}
	if s.Cells == 0 || s.Refused {
		t.Errorf("summary %+v", s)
	}
	if len(audit.events) != 1 || audit.events[0].EntityName != "condition" || audit.events[0].EntityCount != 48 || audit.events[0].AgentName != "trends_build" {
		t.Errorf("audit %+v, want one read of 48 Conditions by the build", audit.events)
	}
	first := src.keys[0]
	src.asked, src.keys = nil, nil
	if _, err := newBuild(src, newMemRepo(), &fakeAudit{}, time.Date(2026, 11, 8, 4, 0, 0, 0, time.UTC)).Build(t.Context(), "2026-10"); err != nil {
		t.Fatal(err)
	}
	if slices.Equal(src.keys[0], first) {
		t.Error("two builds used the same key, want a new one per build")
	}
}

// The same input gives the same published month, whatever key the build draws.
func TestBuild_IsDeterministic(t *testing.T) {
	at := time.Date(2026, 11, 8, 4, 0, 0, 0, time.UTC)
	a, b := newMemRepo(), newMemRepo()
	if _, err := newBuild(fourClinics(), a, &fakeAudit{}, at).Build(t.Context(), "2026-10"); err != nil {
		t.Fatal(err)
	}
	if _, err := newBuild(fourClinics(), b, &fakeAudit{}, at).Build(t.Context(), "2026-10"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(a.cells["2026-10"], b.cells["2026-10"]) {
		t.Error("two builds of the same month differ")
	}
}

// A failing audit stops the build before anything is written.
func TestBuild_WritesNothingWithoutItsAudit(t *testing.T) {
	repo := newMemRepo()
	_, err := newBuild(fourClinics(), repo, &fakeAudit{err: errors.New("down")}, time.Date(2026, 11, 8, 4, 0, 0, 0, time.UTC)).Build(t.Context(), "2026-10")
	if err == nil || len(repo.periods) != 0 {
		t.Errorf("Build = %v with %d periods, want the failure and none", err, len(repo.periods))
	}
}

// The quarter's heatmap counts the fulfilled appointments' patients by the clinic's local weekday and hour, and
// reads no other appointment (the DPIA: only fulfilled ones).
func TestBuild_AQuarterHeatmap(t *testing.T) {
	src := fourClinics()
	src.starts = map[string][]appointment.Start{}
	monday9 := time.Date(2026, 8, 3, 7, 0, 0, 0, time.UTC) // 09:00 in Sarajevo (CEST)
	for i := range 4 {
		id := fmt.Sprintf("clinic-%d", i)
		for p := range 12 {
			src.starts[id] = append(src.starts[id],
				appointment.Start{Start: monday9, Patient: patientID(i*100 + p), Status: appointment.StatusFulfilled},
				appointment.Start{Start: monday9, Patient: patientID(5000 + i*100 + p), Status: appointment.StatusCancelled})
		}
	}
	repo, audit := newMemRepo(), &fakeAudit{}
	if _, err := newBuild(src, repo, audit, time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)).Build(t.Context(), "2026-Q3"); err != nil {
		t.Fatal(err)
	}
	if len(audit.events) != 1 || audit.events[0].EntityCount != 48 {
		t.Errorf("audit %+v, want one read of the 48 fulfilled appointments only", audit.events)
	}
	found := false
	for _, r := range repo.cells["2026-Q3"] {
		if r.Table == trends.TableHeatmap && r.Weekday == "1" && r.Hour == "9" && r.Entity == "" && !r.Suppressed && r.Count == 50 {
			found = true
		}
		if r.City != "" || r.Group != "" {
			t.Errorf("heatmap row %+v has a city or a diagnosis", r)
		}
	}
	if !found {
		t.Errorf("no published Monday 9:00 cell of 48 fulfilled patients in %+v", repo.cells["2026-Q3"])
	}
}

// The weekly rises: a step in the region's respiratory infections is flagged; a clinic that joined the region within
// the ten weeks is left out, so its many patients raise nothing.
func TestBuild_AWeekOfRises(t *testing.T) {
	w44 := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC) // Monday of 2026-W44
	at := time.Date(2026, 11, 3, 1, 0, 0, 0, time.UTC)   // its Tuesday
	build := func(step bool, joiner bool) []trends.Flag {
		src := &fakeSources{recorded: map[string][]condition.Recorded{}}
		for i := range 4 {
			id := fmt.Sprintf("clinic-%d", i)
			src.locs = append(src.locs, clinicAt(id, "BA-09", "Q1"))
			for w := range 10 {
				n := 3
				if step && w == 9 {
					n = 9
				}
				day := w44.AddDate(0, 0, 7*(w-9)+2).Add(9 * time.Hour)
				src.recorded[id] = append(src.recorded[id], diagnosesOf(i*10000+w*100, n, "J06.9", "r", day)...)
			}
		}
		if joiner {
			j := clinicAt("joiner", "BA-09", "Q2")
			j.LocationSince = w44.AddDate(0, 0, -14)
			src.locs = append(src.locs, j)
			src.recorded["joiner"] = diagnosesOf(90000, 80, "J06.9", "r", w44.AddDate(0, 0, 2))
		}
		repo := newMemRepo()
		if _, err := newBuild(src, repo, &fakeAudit{}, at).Build(t.Context(), "2026-W44"); err != nil {
			t.Fatal(err)
		}
		return repo.flags["2026-W44"]
	}
	if got := build(true, false); len(got) != 1 || got[0] != (trends.Flag{Week: "2026-W44", Region: "BA-09", Group: "respiratory-infections"}) {
		t.Errorf("a step: %+v, want one flag", got)
	}
	if got := build(false, true); len(got) != 0 {
		t.Errorf("a joining clinic: %+v, want none", got)
	}
}
