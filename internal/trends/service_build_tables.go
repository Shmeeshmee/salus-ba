package trends

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"
)

// month builds a calendar month's G×D, G×S and S×D tables (§5) over the confirmed Conditions of the clinics that
// count, each in its own zone's month.
func (b *Builder) month(ctx context.Context, p span, src sources, key []byte) (Summary, error) {
	acc, parents, people, read := cells{}, newParents(), interner{}, 0
	for i, loc := range src.locations {
		z := zone(loc)
		if !counts(loc, p.start(z), src.entity) {
			continue
		}
		recs, err := b.p.Diagnoses.Recorded(ctx, loc.ID, p.start(z), p.end(z), key)
		if err != nil {
			return Summary{}, err
		}
		roles, err := b.p.Specialties.Specialties(ctx, loc.ID)
		if err != nil {
			return Summary{}, err
		}
		read += len(recs)
		geo := parents.geo(loc.City, loc.Region, src.entity[loc.Region])
		for _, r := range recs {
			group, sub := src.groups.Of(r.Code)
			diag := parents.diag(group, sub, strings.ToUpper(r.Code))
			spec := []string{""}
			if s := roles[r.PractitionerRoleID]; s != "" {
				spec = append(spec, s)
				parents[DimSpecialty][s] = []string{""}
			}
			addMonth(acc, geo, spec, diag, int32(i), people.of(r.Patient))
		}
	}
	if err := b.audit(ctx, "condition", read); err != nil {
		return Summary{}, err
	}
	r := Suppress(Input{Cells: acc.contributions(), Parents: parents, K: b.p.K, KeyCell: codeKey, Droppable: isCode})
	rows := monthRows(p.name, r, parents)
	return b.publish(ctx, p, r, rows)
}

// addMonth adds a patient's diagnosis to the month's cells: geography × diagnosis, geography × specialty and
// specialty × diagnosis, each with their "all" levels.
func addMonth(acc cells, geo, spec, diag []string, clinic, patient int32) {
	for _, g := range geo {
		for _, d := range diag {
			acc.add(Key{g, "", d}, clinic, patient)
		}
		for _, s := range spec {
			acc.add(Key{g, s, ""}, clinic, patient)
		}
	}
	for _, s := range spec {
		for _, d := range diag {
			acc.add(Key{"", s, d}, clinic, patient)
		}
	}
}

// codeKey is the cell that makes a code a key of the month (A8): the code at geography = all, specialty = all.
func codeKey(k Key) (Key, bool) {
	if isCode(k) {
		return Key{"", "", k[DimDiagnosis]}, true
	}
	return Key{}, false
}

func isCode(k Key) bool { return strings.HasPrefix(k[DimDiagnosis], "c:") }

// quarter builds a calendar quarter's heatmap (§5): geography (to the region) × weekday × hour of the fulfilled
// appointments, in each clinic's zone.
func (b *Builder) quarter(ctx context.Context, p span, src sources, key []byte) (Summary, error) {
	acc, parents, people, read := cells{}, newParents(), interner{}, 0
	for i, loc := range src.locations {
		z := zone(loc)
		if !counts(loc, p.start(z), src.entity) {
			continue
		}
		starts, err := b.p.Appointments.Fulfilled(ctx, loc.ID, p.start(z), p.end(z), key)
		if err != nil {
			return Summary{}, err
		}
		read += len(starts)
		geo := parents.geo("", loc.Region, src.entity[loc.Region])
		for _, s := range starts {
			t := s.Start.In(z)
			day, hour := strconv.Itoa((int(t.Weekday())+6)%7+1), strconv.Itoa(t.Hour())
			parents[DimWeekday][day], parents[DimHour][hour] = []string{""}, []string{""}
			patient := people.of(s.Patient)
			for _, g := range geo {
				for _, d := range []string{"", day} {
					for _, h := range []string{"", hour} {
						acc.add(Key{g, d, h}, int32(i), patient)
					}
				}
			}
		}
	}
	if err := b.audit(ctx, "appointment", read); err != nil {
		return Summary{}, err
	}
	r := Suppress(Input{Cells: acc.contributions(), Parents: parents, K: b.p.K})
	return b.publish(ctx, p, r, heatmapRows(p.name, r, parents))
}

// publish writes the period and its rows, or the period alone as refused (A7), and sums it up in counts (A13).
func (b *Builder) publish(ctx context.Context, p span, r Result, rows []Row) (Summary, error) {
	status := StatusPublished
	if r.Refused {
		status, rows = StatusRefused, nil
	}
	period := Period{Period: p.name, Kind: p.kind, Status: status, MinCount: b.p.K, CodeLevel: !r.Dropped}
	if err := b.save(ctx, period, rows, nil); err != nil {
		return Summary{}, err
	}
	return Summary{Cells: len(r.Published) + len(r.Suppressed), Suppressed: len(r.Suppressed), Refused: r.Refused, CodeDropped: r.Dropped}, nil
}

// parents are the dimensions' trees as a build finds them (Input.Parents).
type parents [3]map[string][]string

func newParents() parents {
	return parents{{}, {}, {}}
}

// geo records the clinic's geography and returns its nodes: all, entity, region and, when given, city.
func (ps parents) geo(city, region, entity string) []string {
	e, r := "e:"+entity, "r:"+region
	ps[DimGeo][e], ps[DimGeo][r] = []string{""}, []string{e}
	nodes := []string{"", e, r}
	if city != "" {
		c := "c:" + city
		ps[DimGeo][c] = []string{r}
		nodes = append(nodes, c)
	}
	return nodes
}

// diag records a code's place in the grouping and returns its nodes: all, group, subgroup (if any) and code. A
// code is within its group and its subgroup; a subgroup within its group.
func (ps parents) diag(group, sub, code string) []string {
	g, c := "g:"+group, "c:"+code
	ps[DimDiagnosis][g] = []string{""}
	nodes := []string{"", g}
	if sub == "" {
		ps[DimDiagnosis][c] = []string{g}
		return append(nodes, c)
	}
	s := "s:" + sub
	ps[DimDiagnosis][s] = []string{g}
	ps[DimDiagnosis][c] = []string{g, s}
	return append(nodes, s, c)
}

// monthRows are the stored rows of a month: each cell once per table it is in, except a dropped code level.
func monthRows(period string, r Result, ps parents) []Row {
	var rows []Row
	for _, k := range sortedKeys(r) {
		if r.Dropped && isCode(k) {
			continue
		}
		base := Row{Period: period, Specialty: k[DimSpecialty]}
		base.Entity, base.Region, base.City = ps.geoColumns(k[DimGeo])
		base.Group, base.Subgroup, base.Code = ps.diagColumns(k[DimDiagnosis])
		base.Count, base.Suppressed = r.Published[k], r.Suppressed[k]
		for _, t := range tablesOf(k) {
			row := base
			row.Table = t
			rows = append(rows, row)
		}
	}
	return rows
}

// tablesOf are the month tables a cell is in: G×D has no specialty, G×S no diagnosis, S×D no geography.
func tablesOf(k Key) []string {
	var out []string
	if k[DimSpecialty] == "" {
		out = append(out, TableGD)
	}
	if k[DimDiagnosis] == "" {
		out = append(out, TableGS)
	}
	if k[DimGeo] == "" {
		out = append(out, TableSD)
	}
	return out
}

// heatmapRows are the stored rows of a quarter.
func heatmapRows(period string, r Result, ps parents) []Row {
	var rows []Row
	for _, k := range sortedKeys(r) {
		row := Row{Period: period, Table: TableHeatmap, Weekday: k[DimWeekday], Hour: k[DimHour]}
		row.Entity, row.Region, _ = ps.geoColumns(k[DimGeo])
		row.Count, row.Suppressed = r.Published[k], r.Suppressed[k]
		rows = append(rows, row)
	}
	return rows
}

// geoColumns are a geography node's entity, region and city columns.
func (ps parents) geoColumns(node string) (entity, region, city string) {
	for n := node; n != ""; {
		lvl, id, _ := strings.Cut(n, ":")
		switch lvl {
		case "c":
			city = id
		case "r":
			region = id
		case "e":
			entity = id
		}
		n = first(ps[DimGeo][n])
	}
	return entity, region, city
}

// diagColumns are a diagnosis node's group, subgroup and code columns.
func (ps parents) diagColumns(node string) (group, sub, code string) {
	lvl, id, _ := strings.Cut(node, ":")
	switch lvl {
	case "c":
		code = id
		for _, p := range ps[DimDiagnosis][node] {
			if g, s := ps.diagColumnsOf(p); g != "" {
				group = g
			} else if s != "" {
				sub = s
			}
		}
	case "s":
		sub = id
		group = strings.TrimPrefix(first(ps[DimDiagnosis][node]), "g:")
	case "g":
		group = id
	}
	return group, sub, code
}

// diagColumnsOf is a group's or a subgroup's id.
func (ps parents) diagColumnsOf(node string) (group, sub string) {
	lvl, id, _ := strings.Cut(node, ":")
	if lvl == "g" {
		return id, ""
	}
	return "", id
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

// sortedKeys are the result's cells in key order.
func sortedKeys(r Result) []Key {
	keys := make([]Key, 0, len(r.Published)+len(r.Suppressed))
	for k := range r.Published {
		keys = append(keys, k)
	}
	for k := range r.Suppressed {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, compareKeys)
	return keys
}

func sortInts(s []int32) []int32 {
	slices.Sort(s)
	return s
}

// weekly is a time's ISO week index from the first of n weeks starting at first (local midnight), or -1.
func weekly(t, first time.Time, n int) int {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	days := int(d.Sub(first).Hours()+12) / 24 // whole days, across a DST change
	if days < 0 || days >= 7*n {
		return -1
	}
	return days / 7
}
