package trends

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/Shmeeshmee/salus/internal/terminology"
)

// Query is a breakdown of one month table (§5, §8): a table, a range of months, some of its dimensions fixed, and
// the one dimension (or month) it breaks down by. No other combination exists.
type Query struct {
	Fixed    map[string]string // dimension → value
	Table    string
	From, To string // months, 2026-01
	By       string // a dimension of the table, or "month"
}

// The dimensions of the month tables and their hierarchies (level 1 is the coarsest).
const (
	DimNameEntity    = "entity"
	DimNameRegion    = "region"
	DimNameCity      = "city"
	DimNameGroup     = "group"
	DimNameSubgroup  = "subgroup"
	DimNameCode      = "code"
	DimNameSpecialty = "specialty"
	ByMonth          = "month"
)

// hierarchy is a dimension's hierarchy: geography, diagnosis or specialty; level is its depth in it.
var hierarchy = map[string]struct {
	name  string
	level int
}{
	DimNameEntity: {"geo", 1}, DimNameRegion: {"geo", 2}, DimNameCity: {"geo", 3},
	DimNameGroup: {"diag", 1}, DimNameSubgroup: {"diag", 2}, DimNameCode: {"diag", 3},
	DimNameSpecialty: {"spec", 1},
}

// tableHierarchies are the hierarchies of each month table.
var tableHierarchies = map[string][]string{
	TableGD: {"geo", "diag"}, TableGS: {"geo", "spec"}, TableSD: {"spec", "diag"},
}

// maxMonths bounds a query's range.
const maxMonths = 60

// Cells returns the breakdown q asks for: every key of its dimension (A8), each with the sum of its published month
// cells over the range, null when none is published, and Partial when a count leaves out a null month (A9). A
// query outside the tables' combinations gives ErrInvalidQuery.
func (s *Service) Cells(ctx context.Context, c Caller, q Query) (Breakdown, error) {
	months, ok := q.check()
	if !ok {
		return Breakdown{}, ErrInvalidQuery
	}
	var out Breakdown
	err := s.read(ctx, c, "trends_cell", func(ctx context.Context) error {
		built, err := s.builtMonths(ctx, months)
		if err != nil {
			return err
		}
		rows, err := s.p.Repo.Cells(ctx, q.Table, q.From, q.To)
		if err != nil {
			return err
		}
		keys, err := s.keysOf(ctx, q, rows)
		if err != nil {
			return err
		}
		out = q.breakdown(rows, built, keys)
		return nil
	})
	return out, err
}

// check validates q and returns its months, from first to last; false for a query outside the combinations.
func (q Query) check() ([]string, bool) {
	hs, ok := tableHierarchies[q.Table]
	if !ok {
		return nil, false
	}
	from, err1 := parsePeriod(q.From)
	to, err2 := parsePeriod(q.To)
	if err1 != nil || err2 != nil || from.kind != KindMonth || to.kind != KindMonth || q.From > q.To {
		return nil, false
	}
	var months []string
	for m := from; m.name <= q.To; m = nextMonth(m) {
		if months = append(months, m.name); len(months) > maxMonths {
			return nil, false
		}
	}
	in := func(dim string) bool { h, ok := hierarchy[dim]; return ok && slices.Contains(hs, h.name) }
	for dim, v := range q.Fixed {
		if !in(dim) || v == "" || dim == q.By {
			return nil, false
		}
	}
	if q.By == ByMonth {
		return months, true
	}
	if !in(q.By) {
		return nil, false
	}
	by := hierarchy[q.By]
	for dim := range q.Fixed { // a fixed dimension of the same hierarchy lies above the breakdown
		if h := hierarchy[dim]; h.name == by.name && h.level >= by.level {
			return nil, false
		}
	}
	return months, true
}

func nextMonth(m span) span {
	t := m.start(time.UTC).AddDate(0, 1, 0)
	n, _ := parsePeriod(t.Format("2006-01"))
	return n
}

// builtMonths are the months of the range that were built (published or refused), by name.
func (s *Service) builtMonths(ctx context.Context, months []string) (map[string]bool, error) {
	ps, err := s.p.Repo.Periods(ctx, KindMonth)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, p := range ps {
		if slices.Contains(months, p.Period) {
			out[p.Period] = true
		}
	}
	return out, nil
}

// column is a row's value of a dimension.
func column(r Row, dim string) string {
	switch dim {
	case DimNameEntity:
		return r.Entity
	case DimNameRegion:
		return r.Region
	case DimNameCity:
		return r.City
	case DimNameGroup:
		return r.Group
	case DimNameSubgroup:
		return r.Subgroup
	case DimNameCode:
		return r.Code
	}
	return r.Specialty
}

// matches reports whether the row is a cell of q's breakdown: its fixed dimensions equal, its breakdown dimension
// set, and every level below the deepest one q names of a hierarchy at "all".
func (q Query) matches(r Row) bool {
	deepest := map[string]int{}
	for dim := range hierarchy {
		if _, fixed := q.Fixed[dim]; fixed || dim == q.By {
			h := hierarchy[dim]
			deepest[h.name] = max(deepest[h.name], h.level)
		}
	}
	for dim, h := range hierarchy {
		v, fixed := q.Fixed[dim]
		switch {
		case fixed && column(r, dim) != v:
			return false
		case dim == q.By && column(r, dim) == "":
			return false
		case !fixed && dim != q.By && h.level > deepest[h.name] && column(r, dim) != "":
			return false
		}
	}
	return true
}

// breakdown sums each key's published cells over the built months; a key is a code only in the months it is a key.
func (q Query) breakdown(rows []Row, built map[string]bool, keys []string) Breakdown {
	type sum struct {
		total, nulls int
		any          bool
	}
	codeKeys := codeKeysByMonth(rows)
	sums := map[string]*sum{}
	for _, k := range keys {
		sums[k] = &sum{}
	}
	published := map[[2]string]int{} // key, month → count
	for _, r := range rows {
		if r.Suppressed || !built[r.Period] || !q.matches(r) || !codeAllowed(q, r, codeKeys) {
			continue
		}
		key := r.Period
		if q.By != ByMonth {
			key = column(r, q.By)
		}
		published[[2]string{key, r.Period}] = r.Count
	}
	for _, k := range keys {
		for m := range built {
			if q.By == ByMonth && m != k {
				continue
			}
			if v, ok := published[[2]string{k, m}]; ok {
				sums[k].total += v
				sums[k].any = true
			} else {
				sums[k].nulls++
			}
		}
	}
	out := Breakdown{Rows: make([]Count, 0, len(keys))}
	for _, k := range keys {
		c := Count{Key: k}
		if s := sums[k]; s.any {
			c.Count = &s.total
			out.Partial = out.Partial || s.nulls > 0
		}
		out.Rows = append(out.Rows, c)
	}
	return out
}

// codeKeysByMonth are, per month, the codes published at geography = all and specialty = all (A8).
func codeKeysByMonth(rows []Row) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, r := range rows {
		if r.Code != "" && !r.Suppressed && r.Entity == "" && r.Region == "" && r.City == "" && r.Specialty == "" {
			if out[r.Period] == nil {
				out[r.Period] = map[string]bool{}
			}
			out[r.Period][r.Code] = true
		}
	}
	return out
}

// codeAllowed reports whether a row's code, if it has one, is a key of its month.
func codeAllowed(q Query, r Row, keys map[string]map[string]bool) bool {
	return r.Code == "" || keys[r.Period][r.Code]
}

// keysOf are the keys of q's breakdown: every place, group, subgroup or specialty of its level under the fixed
// ones; for codes, those that are a key in some month of the range; the months for a breakdown by month.
func (s *Service) keysOf(ctx context.Context, q Query, rows []Row) ([]string, error) {
	switch q.By {
	case ByMonth:
		months, _ := q.check()
		return months, nil
	case DimNameSpecialty:
		return slices.Clone(s.p.Specialties), nil
	case DimNameEntity, DimNameRegion, DimNameCity:
		places, err := s.p.Catalog.AllPlaces(ctx)
		if err != nil {
			return nil, err
		}
		return placeKeys(places, q), nil
	case DimNameCode:
		return codeKeys(rows, q), nil
	}
	groups, err := s.p.Catalog.Groups(ctx)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, g := range groups.List() {
		isSub := g.Parent != ""
		if (q.By == DimNameSubgroup) == isSub && (!isSub || q.Fixed[DimNameGroup] == "" || q.Fixed[DimNameGroup] == g.Parent) {
			out = append(out, g.Code)
		}
	}
	return out, nil
}

// placeKeys are the places of q's breakdown level under its fixed geography. An entity is the Federation, or a
// region that is its own entity.
func placeKeys(places []terminology.Place, q Query) []string {
	parent := map[string]string{}
	for _, p := range places {
		parent[p.ID] = p.Parent
	}
	entityOf := func(region string) string {
		if parent[region] == "" {
			return region
		}
		return parent[region]
	}
	var out []string
	for _, p := range places {
		var entity, region string
		switch {
		case q.By == DimNameEntity && (p.Kind == "entity" || (p.Kind == "region" && p.Parent == "")):
			entity = p.ID
		case q.By == DimNameRegion && p.Kind == "region":
			entity, region = entityOf(p.ID), p.ID
		case q.By == DimNameCity && p.Kind == "city":
			entity, region = entityOf(p.Parent), p.Parent
		default:
			continue
		}
		if f := q.Fixed[DimNameEntity]; f != "" && f != entity {
			continue
		}
		if f := q.Fixed[DimNameRegion]; f != "" && f != region {
			continue
		}
		out = append(out, p.ID)
	}
	slices.Sort(out)
	return out
}

// codeKeys are the codes that are a key in some month of the range, under the fixed group or subgroup.
func codeKeys(rows []Row, q Query) []string {
	seen := map[string]bool{}
	for _, codes := range codeKeysByMonth(rows) {
		for code := range codes {
			seen[code] = true
		}
	}
	var out []string
	for _, r := range rows {
		if !seen[r.Code] || r.Entity != "" || r.Region != "" || r.City != "" || r.Specialty != "" {
			continue
		}
		if g := q.Fixed[DimNameGroup]; g != "" && r.Group != g {
			continue
		}
		if sg := q.Fixed[DimNameSubgroup]; sg != "" && r.Subgroup != sg {
			continue
		}
		if !slices.Contains(out, r.Code) {
			out = append(out, r.Code)
		}
	}
	slices.SortFunc(out, strings.Compare)
	return out
}
