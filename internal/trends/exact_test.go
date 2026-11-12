package trends_test

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/Shmeeshmee/salus/internal/trends"
)

// The exact integer check of the audit (§5 "Tests"), written apart from the package's LP so that it checks it: the
// relations are found again from the cells, and each suppressed cell's interval is enumerated over the integers.

type testRelation struct {
	parent   trends.Key
	children []trends.Key
	cover    bool
}

func unionOf(c trends.Contribution) []int32 {
	seen := map[int32]bool{}
	for _, ps := range c {
		for _, p := range ps {
			seen[p] = true
		}
	}
	out := make([]int32, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	slices.Sort(out)
	return out
}

func levelOf(node string) string {
	before, _, _ := strings.Cut(node, ":")
	return before
}

// relationsOf groups every cell under each of its parents, per dimension and level; a relation is a cover when the
// children's patients are the parent's.
func relationsOf(in trends.Input) []testRelation {
	type id struct {
		parent trends.Key
		level  string
		dim    int
	}
	byID := map[id]*testRelation{}
	var keys []trends.Key
	for k := range in.Cells {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return strings.Join(keys[i][:], "|") < strings.Join(keys[j][:], "|") })
	var order []id
	for _, k := range keys {
		for d := range 3 {
			for _, p := range in.Parents[d][k[d]] {
				parent := k
				parent[d] = p
				if _, ok := in.Cells[parent]; !ok {
					continue
				}
				i := id{parent: parent, dim: d, level: levelOf(k[d])}
				if byID[i] == nil {
					byID[i] = &testRelation{parent: parent}
					order = append(order, i)
				}
				byID[i].children = append(byID[i].children, k)
			}
		}
	}
	out := make([]testRelation, 0, len(order))
	for _, i := range order {
		r := byID[i]
		all := trends.Contribution{}
		for n, c := range r.children {
			all[int32(n)] = unionOf(in.Cells[c])
		}
		r.cover = slices.Equal(unionOf(all), unionOf(in.Cells[r.parent]))
		out = append(out, *r)
	}
	return out
}

// row is Σ coef·r <= rhs over the cells' residuals.
type row struct {
	coef map[int]int
	rhs  int
}

// checkExact asserts that every suppressed cell's integer interval, with its two largest clinics' own patients known
// in every cell, is at least k wide.
func checkExact(t *testing.T, in trends.Input, r trends.Result) {
	t.Helper()
	keys := sortedCells(in)
	rels := relationsOf(in)
	for xi, x := range keys {
		if !r.Suppressed[x] {
			continue
		}
		lo, hi, rows := systemOf(in, r, keys, rels, largestTwo(in.Cells[x]))
		minV, maxV, ok := extremes(lo, hi, rows, xi)
		if !ok {
			t.Errorf("%v: no integer solution, but the true counts are one", x)
			continue
		}
		if maxV-minV < in.K {
			t.Errorf("%v suppressed with an integer interval [%d, %d] narrower than %d", x, minV, maxV, in.K)
		}
	}
}

func sortedCells(in trends.Input) []trends.Key {
	var keys []trends.Key
	for k := range in.Cells {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return strings.Join(keys[i][:], "|") < strings.Join(keys[j][:], "|") })
	return keys
}

// systemOf is the integer system over the cells' residuals with the known clinics' patients taken out: published
// cells within [v-2, v+2], each child within its parent, and a cover parent's residual within its children's.
func systemOf(in trends.Input, r trends.Result, keys []trends.Key, rels []testRelation, known []int32) (lo, hi []int, rows []row) {
	index := map[trends.Key]int{}
	a := make([]int, len(keys))
	upper := 0
	for i, k := range keys {
		index[k] = i
		ps := map[int32]bool{}
		for _, c := range known {
			for _, p := range in.Cells[k][c] {
				ps[p] = true
			}
		}
		a[i] = len(ps)
		upper = max(upper, len(unionOf(in.Cells[k]))*3+10)
	}
	lo, hi = make([]int, len(keys)), make([]int, len(keys))
	for i, k := range keys {
		lo[i], hi[i] = 0, upper
		if v, ok := r.Published[k]; ok {
			lo[i], hi[i] = max(0, v-2-a[i]), v+2-a[i]
		}
	}
	for _, rel := range rels {
		p := index[rel.parent]
		for _, c := range rel.children {
			ci := index[c]
			rows = append(rows, row{coef: map[int]int{ci: 1, p: -1}, rhs: a[p] - a[ci]})
		}
		if rel.cover {
			coef := map[int]int{p: 1}
			for _, c := range rel.children {
				coef[index[c]]--
			}
			rows = append(rows, row{coef: coef, rhs: 0})
		}
	}
	return lo, hi, rows
}

func largestTwo(c trends.Contribution) []int32 {
	clinics := make([]int32, 0, len(c))
	for k := range c {
		clinics = append(clinics, k)
	}
	slices.Sort(clinics)
	slices.SortStableFunc(clinics, func(a, b int32) int { return len(c[b]) - len(c[a]) })
	return clinics[:min(2, len(clinics))]
}

// extremes is the smallest and largest integer value of variable x over the integer points of the rows within the
// bounds.
func extremes(lo, hi []int, rows []row, x int) (minV, maxV int, ok bool) {
	l, h := slices.Clone(lo), slices.Clone(hi)
	if !propagate(l, h, rows) {
		return 0, 0, false
	}
	minV, ok = -1, false
	for v := l[x]; v <= h[x]; v++ {
		if feasibleAt(l, h, rows, x, v) {
			minV, ok = v, true
			break
		}
	}
	if !ok {
		return 0, 0, false
	}
	for v := h[x]; v >= minV; v-- {
		if feasibleAt(l, h, rows, x, v) {
			return minV, v, true
		}
	}
	return minV, minV, true
}

func feasibleAt(lo, hi []int, rows []row, x, v int) bool {
	l, h := slices.Clone(lo), slices.Clone(hi)
	l[x], h[x] = v, v
	return feasible(l, h, rows)
}

// feasible searches the integer points by bound propagation and splitting.
func feasible(l, h []int, rows []row) bool {
	if !propagate(l, h, rows) {
		return false
	}
	best, width := -1, 0
	for i := range l {
		if w := h[i] - l[i]; w > 0 && (best < 0 || w < width) {
			best, width = i, w
		}
	}
	if best < 0 {
		return true // every variable fixed and every row holds
	}
	mid := l[best] + width/2
	l1, h1 := slices.Clone(l), slices.Clone(h)
	h1[best] = mid
	if feasible(l1, h1, rows) {
		return true
	}
	l2, h2 := slices.Clone(l), slices.Clone(h)
	l2[best] = mid + 1
	return feasible(l2, h2, rows)
}

// propagate tightens the bounds until each row holds at its loosest; false when a bound crosses.
func propagate(l, h []int, rows []row) bool {
	for changed := true; changed; {
		changed = false
		for _, r := range rows {
			tightened, ok := tighten(l, h, r)
			if !ok {
				return false
			}
			changed = changed || tightened
		}
	}
	return true
}

// tighten tightens the bounds by one row; false when the row cannot hold.
func tighten(l, h []int, r row) (changed, ok bool) {
	least := 0
	for j, c := range r.coef {
		if c > 0 {
			least += c * l[j]
		} else {
			least += c * h[j]
		}
	}
	if least > r.rhs {
		return false, false
	}
	for j, c := range r.coef {
		switch {
		case c > 0:
			if nb := (r.rhs - (least - c*l[j])) / c; nb < h[j] {
				h[j], changed = nb, true
			}
		case c < 0: // c·r_j <= rhs - others: r_j >= ceil((rhs - others) / c)
			if nb := ceilDiv(r.rhs-(least-c*h[j]), c); nb > l[j] {
				l[j], changed = nb, true
			}
		}
		if l[j] > h[j] {
			return changed, false
		}
	}
	return changed, true
}

func ceilDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) == (b < 0)) {
		q++
	}
	return q
}

func TestExactCheck_FindsTheIntervalOfASmallSystem(t *testing.T) {
	// x0 = parent in [8, 12]; x1, x2 children; x1 in [0, 3]; x0 <= x1 + x2; x1, x2 <= x0.
	rows := []row{{map[int]int{0: 1, 1: -1, 2: -1}, 0}, {map[int]int{1: 1, 0: -1}, 0}, {map[int]int{2: 1, 0: -1}, 0}}
	lo, hi, ok := extremes([]int{8, 0, 0}, []int{12, 3, 100}, rows, 2)
	if !ok || lo != 5 || hi != 12 {
		t.Errorf("x2 in [%d, %d] %v, want [5, 12]", lo, hi, ok)
	}
	if _, _, ok := extremes([]int{8, 0, 0}, []int{8, 3, 4}, rows, 2); ok {
		t.Errorf("an infeasible system has a solution")
	}
}

func TestExactCheck_Arithmetic(t *testing.T) {
	for _, tt := range []struct{ a, b, want int }{{7, -2, -3}, {-7, -2, 4}, {6, -2, -3}, {-6, -2, 3}, {7, 2, 4}} {
		if got := ceilDiv(tt.a, tt.b); got != tt.want {
			t.Errorf("ceilDiv(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
