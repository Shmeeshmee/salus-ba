package trends

import (
	"cmp"
	"slices"
	"strings"
)

// Input is one period's cells as a build holds them in memory, and how its dimensions nest.
type Input struct {
	// Cells are the period's cells with at least one patient; a cell absent is 0.
	Cells map[Key]Contribution
	// Parents gives, per dimension, the parents of each node: its parent level ("" for "all"), and for an ICD-10
	// code both its group and its subgroup. A node without an entry is the root ("").
	Parents [3]map[string][]string
	// KeyCell, when set, names the cell whose publication makes a node a key of the period (A8: a code is a key
	// only when its cell at geography = all and specialty = all is published); false for a cell it does not apply to.
	KeyCell func(Key) (Key, bool)
	// Droppable, when set, is the level the audit drops for the period when it still fails (A7: the code level).
	Droppable func(Key) bool
	// K is the minimum count (ANALYTICS_MIN_COUNT, at least 10).
	K int
}

// Result is the period as it is published.
type Result struct {
	// Published are the cells published, each rounded to the nearest 5 (A10).
	Published map[Key]int
	// Suppressed are the cells with patients that are not published: suppressed (A4, A6, A7, A8) or dropped.
	Suppressed map[Key]bool
	// Dropped: the Droppable level is left out of the period (A7).
	Dropped bool
	// Refused: the audit still failed without that level; nothing of the period is published (A7).
	Refused bool
}

// maxRepairs bounds the rounds of extra suppression the audit asks for before it drops the level or refuses the
// period (A7): each round suppresses at least one more cell.
const maxRepairs = 40

// Suppress decides the period's publication as a whole (docs/plans/analytics.md §5, A4 to A10): primary
// suppression of every cell (A4), the key rule (A8), secondary suppression over every relation until nothing
// changes (A6), and the audit of every suppressed cell (A7), which fails closed: one more suppression per failing
// cell and again, then without the Droppable level, then refused. Counts are rounded only once decided (A10). Pure:
// the same input gives the same result.
func Suppress(in Input) Result {
	if r, ok := attempt(in); ok {
		return r
	}
	if in.Droppable != nil {
		kept := Input{Parents: in.Parents, KeyCell: in.KeyCell, K: in.K, Cells: map[Key]Contribution{}}
		for k, c := range in.Cells {
			if !in.Droppable(k) {
				kept.Cells[k] = c
			}
		}
		if r, ok := attempt(kept); ok {
			r.Dropped = true
			for k := range in.Cells {
				if in.Droppable(k) {
					r.Suppressed[k] = true
				}
			}
			return r
		}
	}
	all := map[Key]bool{}
	for k := range in.Cells {
		all[k] = true
	}
	return Result{Published: map[Key]int{}, Suppressed: all, Refused: true}
}

// attempt runs A4 to A8 on in and reports whether the audit passed.
func attempt(in Input) (Result, bool) {
	u := newUniverse(in)
	for k, c := range u.cells {
		c.suppressed = !passes(c.share, in.K)
		u.cells[k] = c
	}
	u.secondary()
	audited := map[Key]bool{}
	for range maxRepairs {
		failing := u.audit(audited)
		if len(failing) == 0 {
			return u.result(), true
		}
		for _, k := range failing {
			u.repair(k)
		}
		u.secondary()
	}
	return Result{}, false
}

// cell is a cell of the universe: its contribution, its distinct patients, and whether it is suppressed.
type cell struct {
	share      Contribution
	patients   []int32
	suppressed bool
}

// relation is a parent cell and its children along one dimension, at one level. cover: the children's patients are
// exactly the parent's (max(children) <= parent <= sum(children)); otherwise each child is only within the parent.
type relation struct {
	parent   Key
	children []Key
	cover    bool
}

// universe is a period's cells and relations, in a fixed order (keys sorted), so the same input gives the same
// suppression.
type universe struct {
	cells     map[Key]*cell
	of        map[Key][]int // the relations each cell is in, as parent or child
	keyCell   func(Key) (Key, bool)
	keys      []Key
	relations []relation
	k         int
	largest   int // the most patients of any cell
}

func newUniverse(in Input) *universe {
	u := &universe{cells: make(map[Key]*cell, len(in.Cells)), of: map[Key][]int{}, keyCell: in.KeyCell, k: in.K}
	for k, c := range in.Cells {
		u.cells[k] = &cell{share: c, patients: union(c)}
		u.keys = append(u.keys, k)
		u.largest = max(u.largest, len(u.cells[k].patients))
	}
	slices.SortFunc(u.keys, compareKeys)
	type relKey struct {
		parent Key
		level  string
		dim    int
	}
	index := map[relKey]int{}
	for _, k := range u.keys {
		for d := range 3 {
			for _, p := range in.Parents[d][k[d]] {
				parent := k
				parent[d] = p
				if _, ok := u.cells[parent]; !ok {
					continue // not a cell of the period's tables
				}
				rk := relKey{parent: parent, dim: d, level: level(k[d])}
				i, ok := index[rk]
				if !ok {
					i = len(u.relations)
					index[rk] = i
					u.relations = append(u.relations, relation{parent: parent})
					u.of[parent] = append(u.of[parent], i)
				}
				u.relations[i].children = append(u.relations[i].children, k)
				u.of[k] = append(u.of[k], i)
			}
		}
	}
	for i, r := range u.relations {
		var all []int32
		for _, c := range r.children {
			all = merge(all, u.cells[c].patients)
		}
		u.relations[i].cover = slices.Equal(all, u.cells[r.parent].patients)
	}
	return u
}

// level is a node's level: the prefix before its colon ("" for a node without one).
func level(node string) string {
	before, _, found := strings.Cut(node, ":")
	if !found {
		return ""
	}
	return before
}

// secondary applies the key rule (A8) and secondary suppression (A6) until nothing changes. It ends: suppression
// only grows over a finite set.
func (u *universe) secondary() {
	for changed := true; changed; {
		changed = u.keys8()
		for _, r := range u.relations {
			if r.cover && u.secondaryOne(r) {
				changed = true
			}
		}
	}
}

// keys8 suppresses every cell whose node is not a key of the period (A8), and reports whether it changed any.
func (u *universe) keys8() bool {
	if u.keyCell == nil {
		return false
	}
	changed := false
	for _, k := range u.keys {
		c := u.cells[k]
		if kc, ok := u.keyCell(k); ok && !c.suppressed {
			if key, in := u.cells[kc]; !in || key.suppressed {
				c.suppressed, changed = true, true
			}
		}
	}
	return changed
}

// secondaryOne applies A6 to one cover relation: its suppressed cells are none, or at least two whose distinct union
// passes A4; otherwise its smallest published child is suppressed too, or its total when only that is left. It
// reports whether it suppressed one.
func (u *universe) secondaryOne(r relation) bool {
	members := append([]Key{r.parent}, r.children...)
	var hidden []Key
	for _, k := range members {
		if u.cells[k].suppressed {
			hidden = append(hidden, k)
		}
	}
	if len(hidden) == 0 || (len(hidden) >= 2 && passes(u.unionOf(hidden), u.k)) {
		return false
	}
	return u.suppressSmallest(r)
}

// suppressSmallest suppresses the smallest published child of r (ties by key), or its total when no child is
// published; false when everything of r is suppressed already.
func (u *universe) suppressSmallest(r relation) bool {
	var pick *Key
	for i, k := range r.children {
		if c := u.cells[k]; !c.suppressed && (pick == nil || len(c.patients) < len(u.cells[*pick].patients)) {
			pick = &r.children[i]
		}
	}
	if pick == nil {
		if p := u.cells[r.parent]; !p.suppressed {
			p.suppressed = true
			return true
		}
		return false
	}
	u.cells[*pick].suppressed = true
	return true
}

// repair gives a cell that failed the audit one more suppression: the smallest published child among its relations,
// or a total when none is left (A7).
func (u *universe) repair(k Key) {
	rels := slices.Clone(u.of[k])
	slices.SortFunc(rels, func(a, b int) int {
		return cmp.Compare(u.smallestPublished(u.relations[a]), u.smallestPublished(u.relations[b]))
	})
	for _, i := range rels {
		if u.suppressSmallest(u.relations[i]) {
			return
		}
	}
}

// smallestPublished is the count of r's smallest published child, or a large number for none.
func (u *universe) smallestPublished(r relation) int {
	least := int(^uint(0) >> 1)
	for _, k := range r.children {
		if c := u.cells[k]; !c.suppressed {
			least = min(least, len(c.patients))
		}
	}
	return least
}

// unionOf is the distinct union of the cells' contributions: per clinic, the union of its patients.
func (u *universe) unionOf(keys []Key) Contribution {
	out := Contribution{}
	for _, k := range keys {
		for clinic, ps := range u.cells[k].share {
			out[clinic] = merge(out[clinic], ps)
		}
	}
	return out
}

// result is the universe's publication, rounded (A10).
func (u *universe) result() Result {
	r := Result{Published: map[Key]int{}, Suppressed: map[Key]bool{}}
	for k, c := range u.cells {
		if c.suppressed {
			r.Suppressed[k] = true
		} else {
			r.Published[k] = round5(len(c.patients))
		}
	}
	return r
}

// passes is A4: at least k patients from at least 3 clinics, and at least max(k, 0.1·x1) left once the two largest
// clinics' own (x1 >= x2) are taken out.
func passes(c Contribution, k int) bool {
	total := len(union(c))
	sizes := make([]int, 0, len(c))
	for _, ps := range c {
		if len(ps) > 0 {
			sizes = append(sizes, len(ps))
		}
	}
	if total < k || len(sizes) < 3 {
		return false
	}
	slices.SortFunc(sizes, func(a, b int) int { return b - a })
	rest := total - sizes[0] - sizes[1]
	return rest >= k && 10*rest >= sizes[0]
}

// round5 is n rounded to the nearest 5 (A10).
func round5(n int) int { return (n + 2) / 5 * 5 }

// union is the distinct patients of c, sorted.
func union(c Contribution) []int32 {
	var out []int32
	for _, clinic := range sortedClinics(c) {
		out = merge(out, c[clinic])
	}
	return out
}

// sortedClinics are c's clinics in order.
func sortedClinics(c Contribution) []int32 {
	out := make([]int32, 0, len(c))
	for clinic := range c {
		out = append(out, clinic)
	}
	slices.Sort(out)
	return out
}

// merge is the sorted union of two sorted lists without repeats.
func merge(a, b []int32) []int32 {
	out := make([]int32, 0, max(len(a), len(b)))
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case j == len(b) || (i < len(a) && a[i] < b[j]):
			out = append(out, a[i])
			i++
		case i == len(a) || b[j] < a[i]:
			out = append(out, b[j])
			j++
		default:
			out = append(out, a[i])
			i, j = i+1, j+1
		}
	}
	return out
}

// compareKeys orders keys by their nodes.
func compareKeys(a, b Key) int {
	for d := range 3 {
		if c := strings.Compare(a[d], b[d]); c != 0 {
			return c
		}
	}
	return 0
}
