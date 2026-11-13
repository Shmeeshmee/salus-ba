package trends_test

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/trends"
)

// visit is one synthetic patient seen by one clinic in a city with a diagnosis node path.
type visit struct {
	city    string // "c:…"
	diag    []string
	clinic  int32
	patient int32
}

// month is a synthetic month: its geography (city → region → entity) and the diagnosis parents.
type month struct {
	geo  map[string][]string
	diag map[string][]string
}

func testMonth() month {
	return month{
		geo: map[string][]string{
			"c:C1": {"r:R1"}, "c:C2": {"r:R1"}, "c:C3": {"r:R2"}, "r:R1": {"e:E"}, "r:R2": {"e:E"}, "e:E": {""},
		},
		diag: map[string][]string{
			"g:G1": {""}, "g:G2": {""}, "g:G3": {""}, "s:S1": {"g:G1"}, "c:A01": {"g:G1", "s:S1"}, "c:A02": {"g:G1", "s:S1"},
			"c:B01": {"g:G2"},
		},
	}
}

// path is node and its ancestors up to "all", the first parent each time.
func path(parents map[string][]string, node string) []string {
	out := []string{node}
	for node != "" {
		ps := parents[node]
		if len(ps) == 0 {
			break
		}
		node = ps[0]
		out = append(out, node)
	}
	return out
}

// input is the G×D cells of the visits (specialty all), every combination of their geography and diagnosis paths.
func (m month) input(visits []visit, k int) trends.Input {
	cells := map[trends.Key]trends.Contribution{}
	for _, v := range visits {
		diag := v.diag
		if len(diag) == 0 {
			diag = []string{""}
		}
		diagNodes := map[string]bool{}
		for _, d := range diag {
			for _, n := range path(m.diag, d) {
				diagNodes[n] = true
			}
		}
		if parents := m.diag[diag[0]]; len(parents) > 1 { // a code: its subgroup too
			for _, n := range parents {
				diagNodes[n] = true
			}
		}
		for _, g := range path(m.geo, v.city) {
			for d := range diagNodes {
				k := trends.Key{g, "", d}
				if cells[k] == nil {
					cells[k] = trends.Contribution{}
				}
				ps := cells[k][v.clinic]
				if i, found := slices.BinarySearch(ps, v.patient); !found {
					cells[k][v.clinic] = slices.Insert(ps, i, v.patient)
				}
			}
		}
	}
	return trends.Input{Cells: cells, Parents: [3]map[string][]string{m.geo, {}, m.diag}, K: k}
}

// spread is n patients of clinic, numbered from first, in city with diagnosis node d.
func spread(clinic, first int32, n int, city, d string) []visit {
	out := make([]visit, n)
	for i := range out {
		out[i] = visit{clinic: clinic, patient: first + int32(i), city: city, diag: []string{d}}
	}
	return out
}

func single(c trends.Contribution) trends.Input {
	return trends.Input{Cells: map[trends.Key]trends.Contribution{{}: c}, K: 10}
}

func patients(first int32, n int) []int32 {
	out := make([]int32, n)
	for i := range out {
		out[i] = first + int32(i)
	}
	return out
}

// A4: at least k patients from at least 3 clinics, and at least max(k, 0.1·x1) left once the two largest clinics'
// own are taken out. A10: published counts are rounded to the nearest 5.
func TestSuppress_A4PrimaryAndA10Rounding(t *testing.T) {
	tests := []struct {
		in   trends.Contribution
		name string
		want int // published, or -1 for suppressed
	}{
		{trends.Contribution{1: patients(0, 3), 2: patients(3, 3), 3: patients(6, 3)}, "fewer than k", -1},
		{trends.Contribution{1: patients(0, 30), 2: patients(30, 30)}, "two clinics", -1},
		{trends.Contribution{1: patients(0, 12), 2: patients(12, 10), 3: patients(22, 8)}, "two largest leave fewer than k", -1},
		{trends.Contribution{1: patients(0, 200), 2: patients(200, 5), 3: patients(205, 15)}, "the largest dominates", -1},
		{trends.Contribution{1: patients(0, 10), 2: patients(5, 10), 3: patients(10, 10)}, "a shared patient is one", -1},
		{trends.Contribution{1: patients(0, 10), 2: patients(10, 10), 3: patients(20, 10)}, "balanced, 30 rounds to 30", 30},
		{trends.Contribution{1: patients(0, 10), 2: patients(10, 10), 3: patients(20, 12)}, "32 rounds to 30", 30},
		{trends.Contribution{1: patients(0, 11), 2: patients(11, 11), 3: patients(22, 11)}, "33 rounds to 35", 35},
		{trends.Contribution{1: patients(0, 12), 2: patients(12, 12), 3: patients(24, 14)}, "38 rounds to 40", 40},
	}
	for _, tt := range tests {
		r := trends.Suppress(single(tt.in))
		got, published := r.Published[trends.Key{}]
		if (tt.want < 0) == published || (published && got != tt.want) {
			t.Errorf("%s: published %v as %d, want %d", tt.name, published, got, tt.want)
		}
	}
}

// A6: a cover relation never has exactly one suppressed cell: a small city's region shows the city as the region
// minus the other city, so the other city is suppressed too (the attack "a region minus a city leaving one or two
// clinics").
func TestSuppress_A6ARegionMinusACity(t *testing.T) {
	m := testMonth()
	var visits []visit
	for c := range int32(4) {
		visits = append(visits, spread(c, c*100, 12, "c:C1", "g:G2")...)
	}
	visits = append(visits, spread(5, 500, 6, "c:C2", "g:G2")...) // one clinic alone
	visits = append(visits, spread(6, 600, 6, "c:C2", "g:G2")...) // and a second
	r := trends.Suppress(m.input(visits, 10))
	for _, city := range []string{"c:C1", "c:C2"} {
		if !r.Suppressed[trends.Key{city, "", ""}] {
			t.Errorf("%s published, want both cities of the region suppressed", city)
		}
	}
	checkRelations(t, m, visits, r)
}

// The attack "a city's specialty from the city total and its other specialties": a G×S city with one small
// specialty. And "two large clinics recovering a small third": the third's patients are hidden. Both checked by
// the relations rule and the exact interval of every suppressed cell.
func TestSuppress_NamedAttacks(t *testing.T) {
	m := testMonth()
	cases := map[string][]visit{
		"a city's small group beside large ones": append(append(append(
			spread(1, 0, 15, "c:C1", "g:G2"), spread(2, 100, 15, "c:C1", "g:G2")...),
			spread(3, 200, 15, "c:C1", "g:G2")...), spread(3, 300, 3, "c:C1", "g:G3")...),
		"two large clinics and a small third": append(append(
			spread(1, 0, 30, "c:C1", "g:G2"), spread(2, 100, 30, "c:C1", "g:G2")...), spread(3, 200, 4, "c:C1", "g:G2")...),
		"two subgroups holding the same patients": func() []visit {
			var vs []visit
			for c := range int32(4) {
				for i := range int32(6) {
					p := c*100 + i
					vs = append(vs, visit{clinic: c, patient: p, city: "c:C1", diag: []string{"c:A01"}},
						visit{clinic: c, patient: p, city: "c:C1", diag: []string{"c:A02"}})
				}
			}
			return vs
		}(),
	}
	for name, visits := range cases {
		t.Run(name, func(t *testing.T) {
			r := trends.Suppress(m.input(visits, 10))
			checkRelations(t, m, visits, r)
			checkExact(t, m.input(visits, 10), r)
		})
	}
}

// A8: a code is a key only when its cell at geography = all is published; a code that is not is suppressed in
// every cell. A7: a month that still fails without its code level is refused.
func TestSuppress_A8CodeKeysAndTheCodeLevel(t *testing.T) {
	m := testMonth()
	var visits []visit
	for c := range int32(4) {
		visits = append(visits, spread(c, c*100, 12, "c:C1", "c:B01")...)
	}
	visits = append(visits, spread(9, 900, 2, "c:C3", "c:A01")...)
	in := m.input(visits, 10)
	in.KeyCell = func(k trends.Key) (trends.Key, bool) {
		if strings.HasPrefix(k[2], "c:") {
			return trends.Key{"", "", k[2]}, true
		}
		return trends.Key{}, false
	}
	in.Droppable = func(k trends.Key) bool { return strings.HasPrefix(k[2], "c:") }
	r := trends.Suppress(in)
	for k := range in.Cells {
		if k[2] == "c:A01" && !r.Suppressed[k] {
			t.Errorf("%v published, want every cell of a code that is no key suppressed", k)
		}
	}
	if r.Refused {
		t.Errorf("refused, want published")
	}
}

// The same input gives the same suppression (map order and all).
func TestSuppress_IsDeterministic(t *testing.T) {
	m := testMonth()
	visits := randomVisits(rand.New(rand.NewPCG(7, 7)))
	first := trends.Suppress(m.input(visits, 10))
	for range 5 {
		again := trends.Suppress(m.input(visits, 10))
		if !mapsEqual(first.Published, again.Published) || !mapsEqual(first.Suppressed, again.Suppressed) {
			t.Fatalf("two runs differ")
		}
	}
}

// The exact integer check (§5 "Tests"): over many small synthetic months, every month the audit passes has every
// suppressed cell's integer interval at least k wide, enumerated with the two largest clinics' own patients known
// and the published values read as intervals.
func TestSuppress_TheAuditNeverPassesAMonthTheExactCheckFails(t *testing.T) {
	m := testMonth()
	checked, hidden := 0, 0
	for seed := range uint64(60) {
		visits := randomVisits(rand.New(rand.NewPCG(seed, 99)))
		in := m.input(visits, 10)
		r := trends.Suppress(in)
		if r.Refused || len(r.Published) == 0 {
			continue
		}
		checked++
		hidden += len(r.Suppressed)
		checkRelations(t, m, visits, r)
		checkExact(t, in, r)
	}
	if checked < 20 || hidden < 100 {
		t.Errorf("only %d months published with %d suppressed cells, want enough to check", checked, hidden)
	}
}

// randomVisits is a synthetic month of 3 to 5 clinics. A patient may see a second diagnosis, or another clinic (so the
// known clinics share patients, A7's overlap), and one month in four has a clinic with more than 10·k patients of one
// diagnosis (A4's 10% rule).
func randomVisits(rng *rand.Rand) []visit {
	cities := []string{"c:C1", "c:C2", "c:C3"}
	diags := []string{"g:G2", "g:G3", "c:A01", "c:A02", "c:B01"}
	var vs []visit
	clinics := 3 + rng.IntN(3)
	cityOf := make([]string, clinics)
	for c := range cityOf {
		cityOf[c] = cities[rng.IntN(len(cities))]
	}
	for c := range int32(clinics) {
		city := cityOf[c]
		for p := range int32(5 + rng.IntN(14)) {
			d := diags[rng.IntN(len(diags))]
			vs = append(vs, visit{clinic: c, patient: c*1000 + p, city: city, diag: []string{d}})
			if rng.IntN(4) == 0 {
				vs = append(vs, visit{clinic: c, patient: c*1000 + p, city: city, diag: []string{diags[rng.IntN(len(diags))]}})
			}
			if rng.IntN(4) == 0 { // the same patient at another clinic, with the same diagnosis
				other := (c + 1 + rng.Int32N(int32(clinics)-1)) % int32(clinics)
				vs = append(vs, visit{clinic: other, patient: c*1000 + p, city: cityOf[other], diag: []string{d}})
			}
		}
	}
	if rng.IntN(4) == 0 { // a large clinic: more than 10·k patients of one diagnosis
		c := rng.Int32N(int32(clinics))
		vs = append(vs, spread(c, 50000, 101+rng.IntN(20), cityOf[c], diags[rng.IntN(len(diags))])...)
	}
	return vs
}

func mapsEqual[V comparable](a, b map[trends.Key]V) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// checkRelations asserts A4 on every published cell and A6 on every cover relation of the month: no relation has
// exactly one suppressed cell.
func checkRelations(t *testing.T, m month, visits []visit, r trends.Result) {
	t.Helper()
	in := m.input(visits, 10)
	for _, rel := range relationsOf(in) {
		if !rel.cover {
			continue
		}
		n := 0
		for _, k := range append([]trends.Key{rel.parent}, rel.children...) {
			if r.Suppressed[k] {
				n++
			}
		}
		if n == 1 {
			t.Errorf("relation %v of %v has one suppressed cell", rel.parent, rel.children)
		}
	}
	for k := range r.Published {
		single := trends.Suppress(trends.Input{Cells: map[trends.Key]trends.Contribution{{}: in.Cells[k]}, K: 10})
		if len(single.Published) == 0 {
			t.Errorf("%v published, but fails A4", k)
		}
	}
}

// BenchmarkSuppress_ASyntheticMonth times a month's suppression on synthetic data: 40 clinics in 3 regions and 12
// cities, 4,000 patients, 27 groups with 3 to 6 codes each, 12 specialties (docs/plans/analytics.PROGRESS.md).
func BenchmarkSuppress_ASyntheticMonth(b *testing.B) {
	rng := rand.New(rand.NewPCG(1, 2))
	in := trends.Input{Cells: map[trends.Key]trends.Contribution{}, Parents: [3]map[string][]string{{}, {}, {}}, K: 10}
	in.KeyCell = func(k trends.Key) (trends.Key, bool) {
		if strings.HasPrefix(k[2], "c:") {
			return trends.Key{"", "", k[2]}, true
		}
		return trends.Key{}, false
	}
	in.Droppable = func(k trends.Key) bool { return strings.HasPrefix(k[2], "c:") }
	add := func(k trends.Key, clinic, patient int32) {
		if in.Cells[k] == nil {
			in.Cells[k] = trends.Contribution{}
		}
		ps := in.Cells[k][clinic]
		if i, found := slices.BinarySearch(ps, patient); !found {
			in.Cells[k][clinic] = slices.Insert(ps, i, patient)
		}
	}
	for c := range int32(40) {
		region, city := "r:R"+string(rune('0'+c%3)), "c:C"+string(rune('a'+c%12))
		in.Parents[0][city], in.Parents[0][region], in.Parents[0]["e:E"] = []string{region}, []string{"e:E"}, []string{""}
		spec := "S" + string(rune('a'+c%12))
		in.Parents[1][spec] = []string{""}
		for p := range int32(100) {
			g := rng.IntN(27)
			group, code := "g:G"+string(rune('a'+g)), "c:"+string(rune('A'+g))+string(rune('0'+rng.IntN(3+g%4)))
			in.Parents[2][group], in.Parents[2][code] = []string{""}, []string{group}
			patient := c*1000 + p
			for _, geo := range []string{"", "e:E", region, city} {
				for _, d := range []string{"", group, code} {
					add(trends.Key{geo, "", d}, c, patient)
				}
				add(trends.Key{geo, spec, ""}, c, patient)
			}
			for _, d := range []string{"", group, code} {
				add(trends.Key{"", spec, d}, c, patient)
			}
		}
	}
	for b.Loop() {
		r := trends.Suppress(in)
		b.ReportMetric(float64(len(in.Cells)), "cells")
		b.ReportMetric(float64(len(r.Suppressed)), "suppressed")
	}
}

// A7: a published cell passed A4, a public rule, so an attacker knows that its patients outside the two known clinics
// are at least k, not only [v-2, v+2]. Here (k = 12) the subgroup S1 is suppressed by A6 beside its small sibling S2;
// its code A01 (22 patients, read as 20) and its group G1 (30) are published, and two clinics hold 5 of A01 each.
// Read only as [18, 22], A01 leaves S1 within [8, 22] outside the two clinics, 14 wide; with A01's residual known to
// be at least 12, S1 is within [12, 22], 10 wide, under k: S1 must not be left as the only thing that hides it.
func TestSuppress_A7KnowsThatAPublishedCellPassedA4(t *testing.T) {
	m := month{
		geo:  map[string][]string{},
		diag: map[string][]string{"g:G1": {""}, "s:S1": {"g:G1"}, "s:S2": {"g:G1"}, "c:A01": {"g:G1", "s:S1"}},
	}
	visits := append(append(append(append(append(
		spread(0, 0, 5, "", "c:A01"), spread(1, 100, 5, "", "c:A01")...),
		spread(2, 200, 4, "", "c:A01")...), spread(3, 300, 4, "", "c:A01")...),
		spread(4, 400, 4, "", "c:A01")...), spread(5, 500, 4, "", "s:S1")...)
	visits = append(visits, spread(6, 600, 4, "", "s:S2")...)
	in := m.input(visits, 12)
	r := trends.Suppress(in)
	if r.Refused {
		t.Fatal("the month is refused")
	}
	if !r.Suppressed[trends.Key{"", "", "s:S1"}] || !r.Suppressed[trends.Key{"", "", "s:S2"}] {
		t.Errorf("S1 and S2 not suppressed: %v", r.Suppressed)
	}
	if !r.Suppressed[trends.Key{"", "", "c:A01"}] {
		t.Errorf("A01 published, want it suppressed by the audit's repair: %v", r.Published)
	}
	checkExact(t, in, r)
}

// A7: A4 reveals more than r >= k. With the two known clinics holding y1 and y2 of a published cell's patients, o of
// them shared, its patients outside them are at least o + max(k, ceil(ymax/10)). Here (k = 12) two clinics hold 10
// patients each of A01, 5 of them the same: A01 (32, read as 30) has at least 5 + 12 = 17 patients outside them, so
// S1, hidden by A6 beside S2 under G1 (40), is within [32, 42], 10 wide, under k; read with r >= 12 only, it seemed
// [28, 42].
func TestSuppress_A7KnowsTheOverlapOfTheKnownClinics(t *testing.T) {
	m := month{
		geo:  map[string][]string{},
		diag: map[string][]string{"g:G1": {""}, "s:S1": {"g:G1"}, "s:S2": {"g:G1"}, "c:A01": {"g:G1", "s:S1"}},
	}
	visits := slices.Concat(spread(0, 0, 10, "", "c:A01"), spread(1, 5, 10, "", "c:A01"), // patients 5 to 9 at both
		spread(2, 200, 6, "", "c:A01"), spread(3, 300, 6, "", "c:A01"), spread(4, 400, 5, "", "c:A01"),
		spread(5, 500, 4, "", "s:S1"), spread(6, 600, 4, "", "s:S2"))
	in := m.input(visits, 12)
	r := trends.Suppress(in)
	if r.Refused {
		t.Fatal("the month is refused")
	}
	if !r.Suppressed[trends.Key{"", "", "s:S1"}] || !r.Suppressed[trends.Key{"", "", "c:A01"}] {
		t.Errorf("S1 and A01 not both suppressed: published %v", r.Published)
	}
	checkExact(t, in, r)
}
