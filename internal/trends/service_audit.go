package trends

import (
	"cmp"
	"errors"
	"math"
	"slices"
)

// The audit's bounds (A7). Each suppressed cell is solved as an LP over its neighbourhood (neighbourhood), at most
// auditVars cells in a fixed order. Every other cell is held at its true value, which only adds constraints, so the
// interval found lies inside the one an attacker could find over the whole period: the audit can fail a cell that is
// safe, never pass one that is not.
const (
	auditDepth = 2
	auditVars  = 150
	roundSlack = 1e-6
)

// audit returns the suppressed cells whose interval of consistent values is narrower than k, in key order. A cell
// that passed is not solved again: more suppression only takes constraints away, so its interval only widens.
func (u *universe) audit(passed map[Key]bool) []Key {
	var failing []Key
	for _, k := range u.keys {
		if !u.cells[k].suppressed || passed[k] {
			continue
		}
		if u.width(k) >= u.k {
			passed[k] = true
		} else {
			failing = append(failing, k)
		}
	}
	return failing
}

// width is the width of the interval of values of the suppressed cell x consistent with the published cells (each
// read as [v-2, v+2] for the rounding, and with the patients outside the known clinics that A4 reveals: revealed),
// the inequality relations, and the own patients of x's two largest clinics in every cell treated as known; its ends
// rounded inward.
func (u *universe) width(x Key) int { return u.program(x).width() }

// width is the program's interval of r_0. Any failure of the solver, an unbounded program included (every residual
// is bounded by cap, so only a numerical failure gets there), counts as 0: the audit fails closed.
func (p program) width() int {
	hi, err := p.solve(-1)
	if err != nil {
		return 0
	}
	hi = -hi
	if hi >= p.cap-1 {
		return math.MaxInt32 // only the program's own bound, far above any count: no need for the lower end
	}
	lo, err := p.solve(1)
	if err != nil {
		return 0
	}
	return int(math.Floor(hi+roundSlack)) - int(math.Ceil(lo-roundSlack))
}

// program is the LP of one suppressed cell: minimise or maximise its residual r_0 subject to G·r <= h, r >= 0. A
// cell's count is a + r: a the known clinics' own patients in it, r the rest. at is the true residuals, a point of
// the program, from which the solver starts.
type program struct {
	g    [][]float64
	h    []float64
	at   []float64
	vars int
	cap  float64 // an upper bound on every residual, far above any count: reaching it is having no upper bound
}

// solve is the optimum of sign·r_0, or an error when the solver does not finish. The program is solved in the
// change from the true point, r = at + y⁺ - y⁻: its rows G·(y⁺ - y⁻) <= h - G·at and y⁻ - y⁺ <= at have no
// negative bound, so the slacks are a first feasible basis and no first phase is needed.
func (p program) solve(sign float64) (float64, error) {
	m, n := len(p.g), p.vars
	a := make([][]float64, 0, m+n)
	b := make([]float64, 0, m+n)
	for i, row := range p.g {
		r := make([]float64, 2*n)
		rhs := p.h[i]
		for j, v := range row {
			r[j], r[n+j] = v, -v
			rhs -= v * p.at[j]
		}
		a, b = append(a, r), append(b, math.Max(rhs, 0)) // the true point holds every row; Max only clears rounding
	}
	for j := range n {
		r := make([]float64, 2*n)
		r[j], r[n+j] = -1, 1
		a, b = append(a, r), append(b, p.at[j])
	}
	c := make([]float64, 2*n)
	c[0], c[n] = sign, -sign
	opt, y, err := simplex(a, b, c)
	if err != nil {
		return 0, err
	}
	r := make([]float64, n)
	for j := range n {
		r[j] = p.at[j] + y[j] - y[n+j]
	}
	if !p.holds(r) || math.Abs(sign*r[0]-(sign*p.at[0]+opt)) > checkSlack {
		return 0, errNoFinish
	}
	return sign*p.at[0] + opt, nil
}

// holds reports whether r is a point of the program, up to checkSlack: the solver's answer is checked against the
// original rows, so a numerical failure fails the audit instead of passing a cell.
func (p program) holds(r []float64) bool {
	for j := range p.vars {
		if r[j] < -checkSlack {
			return false
		}
	}
	for i, row := range p.g {
		sum := 0.0
		for j, v := range row {
			sum += v * r[j]
		}
		if sum > p.h[i]+checkSlack {
			return false
		}
	}
	return true
}

// The simplex's limits.
const (
	simplexEps   = 1e-9
	checkSlack   = 1e-6  // how far the solver's point may lie outside a row, in patients
	simplexPivot = 20000 // pivots before the solver gives up (and the audit fails closed)
)

var (
	errUnbounded = errors.New("trends: unbounded program")
	errNoFinish  = errors.New("trends: program not finished")
)

// simplex minimises c·x subject to a·x <= b and x >= 0, with b >= 0, from the slack basis: the tableau method with
// Bland's rule, which never cycles. It returns the optimum and the point that reaches it.
func simplex(a [][]float64, b, c []float64) (float64, []float64, error) {
	rows, cols := len(a), len(c)
	width := cols + rows + 1
	t := make([][]float64, rows+1)
	basis := make([]int, rows)
	for i := range rows {
		t[i] = make([]float64, width)
		copy(t[i], a[i])
		t[i][cols+i], t[i][width-1] = 1, b[i]
		basis[i] = cols + i
	}
	t[rows] = make([]float64, width)
	copy(t[rows], c)
	for range simplexPivot {
		enter := -1
		for j := range width - 1 {
			if t[rows][j] < -simplexEps {
				enter = j
				break
			}
		}
		if enter < 0 {
			return -t[rows][width-1], basicPoint(t, basis, cols), nil
		}
		leave := -1
		for i := range rows {
			if t[i][enter] <= simplexEps {
				continue
			}
			ratio := t[i][width-1] / t[i][enter]
			if leave < 0 {
				leave = i
				continue
			}
			best := t[leave][width-1] / t[leave][enter]
			if ratio < best-simplexEps || (ratio <= best+simplexEps && basis[i] < basis[leave]) {
				leave = i
			}
		}
		if leave < 0 {
			return 0, nil, errUnbounded
		}
		pivot(t, leave, enter)
		basis[leave] = enter
	}
	return 0, nil, errNoFinish
}

// basicPoint is the tableau's point: each basic variable at its row's right-hand side, every other at 0.
func basicPoint(t [][]float64, basis []int, cols int) []float64 {
	x := make([]float64, cols)
	for i, v := range basis {
		if v < cols {
			x[v] = t[i][len(t[i])-1]
		}
	}
	return x
}

// pivot makes column e basic in row l.
func pivot(t [][]float64, l, e int) {
	pr := t[l]
	inv := 1 / pr[e]
	for j := range pr {
		pr[j] *= inv
	}
	for i, row := range t {
		if i == l || row[e] == 0 {
			continue
		}
		f := row[e]
		for j, v := range pr {
			if v != 0 {
				row[j] -= f * v
			}
		}
	}
}

// program builds x's LP over its neighbourhood.
func (u *universe) program(x Key) program {
	vars := u.neighbourhood(x)
	index := make(map[Key]int, len(vars))
	for i, k := range vars {
		index[k] = i
	}
	known := twoLargest(u.cells[x].share)
	b := &builder{u: u, index: index, known: known, n: len(vars)}
	limit := float64(4*u.largest + 4*u.k + 10)
	for i, k := range vars {
		b.row(map[int]float64{i: 1}, limit)
		if c := u.cells[k]; !c.suppressed {
			v, a := float64(round5(len(c.patients))), float64(b.knownIn(k))
			b.row(map[int]float64{i: 1}, v+2-a)
			b.row(map[int]float64{i: -1}, -math.Max(v-2-a, float64(b.revealed(k))))
		}
	}
	seen := map[int]bool{}
	for _, k := range vars {
		for _, i := range u.of[k] {
			if !seen[i] {
				seen[i] = true
				b.relation(u.relations[i])
			}
		}
	}
	at := make([]float64, len(vars))
	for i, k := range vars {
		at[i] = float64(len(u.cells[k].patients) - b.knownIn(k))
	}
	return program{g: b.g, h: b.h, at: at, vars: len(vars), cap: limit}
}

// builder collects an LP's rows: a variable per neighbourhood cell, every other cell a constant at its true value.
type builder struct {
	u     *universe
	index map[Key]int
	known []int32
	g     [][]float64
	h     []float64
	n     int
}

// row adds Σ coef·r <= rhs, unless it has no variable.
func (b *builder) row(coef map[int]float64, rhs float64) {
	g := make([]float64, b.n)
	nonzero := false
	for j, v := range coef {
		if v != 0 {
			g[j], nonzero = v, true
		}
	}
	if nonzero {
		b.g, b.h = append(b.g, g), append(b.h, rhs)
	}
}

// term adds c·T_k to an inequality's left side: c·(a_k + r_k), r_k a variable or its true value.
func (b *builder) term(coef map[int]float64, rhs *float64, k Key, c float64, residualOnly bool) {
	if !residualOnly {
		*rhs -= c * float64(b.knownIn(k))
	}
	if i, ok := b.index[k]; ok {
		coef[i] += c
		return
	}
	*rhs -= c * float64(len(b.u.cells[k].patients)-b.knownIn(k))
}

// relation adds r's inequalities: each child within the parent (T_child <= T_parent); for a cover also the
// parent's residual within its children's (r_parent <= Σ r_child: a patient of the parent no known clinic saw in
// it is one of some child no known clinic saw in it).
func (b *builder) relation(r relation) {
	for _, c := range r.children {
		coef, rhs := map[int]float64{}, 0.0
		b.term(coef, &rhs, c, 1, false)
		b.term(coef, &rhs, r.parent, -1, false)
		b.row(coef, rhs)
	}
	if r.cover {
		coef, rhs := map[int]float64{}, 0.0
		b.term(coef, &rhs, r.parent, 1, true)
		for _, c := range r.children {
			b.term(coef, &rhs, c, -1, true)
		}
		b.row(coef, rhs)
	}
}

// revealed is the least residual a published cell k can have, from A4, a public rule: it passed
// T - x1 - x2 >= max(k, ceil(x1/10)) for its two largest clinics, so with the known clinics holding y1 and y2 of its
// patients, o of them shared, r = T - y1 - y2 + o >= o + max(k, ceil(ymax/10)), whichever two clinics are known.
func (b *builder) revealed(k Key) int {
	share := b.u.cells[k].share
	sum, ymax := 0, 0
	for _, clinic := range b.known {
		sum += len(share[clinic])
		ymax = max(ymax, len(share[clinic]))
	}
	overlap := sum - b.knownIn(k)
	return overlap + max(b.u.k, (ymax+9)/10)
}

// knownIn is the number of distinct patients of the known clinics in the cell k.
func (b *builder) knownIn(k Key) int {
	var ps []int32
	for _, clinic := range b.known {
		ps = merge(ps, b.u.cells[k].share[clinic])
	}
	return len(ps)
}

// neighbourhood is the cells the LP of x solves for: x, every suppressed cell reachable from it through suppressed
// cells and the published cells within auditDepth relations, and those published cells; breadth first in relation
// and key order, at most auditVars. A published cell further away, and any cell past the bound, is held at its true
// value: for a published cell that is within the [v-2, v+2] it is read as anyway.
func (u *universe) neighbourhood(x Key) []Key {
	out, seen := []Key{x}, map[Key]bool{x: true}
	type item struct {
		k     Key
		depth int
	}
	queue := []item{{x, 0}}
	for len(queue) > 0 {
		it := queue[0]
		queue = queue[1:]
		for _, i := range u.of[it.k] {
			r := u.relations[i]
			for _, m := range append([]Key{r.parent}, r.children...) {
				if seen[m] {
					continue
				}
				suppressed := u.cells[m].suppressed
				if !suppressed && it.depth+1 > auditDepth {
					continue
				}
				if len(out) == auditVars {
					return out
				}
				seen[m] = true
				out = append(out, m)
				queue = append(queue, item{m, it.depth + 1})
			}
		}
	}
	return out
}

// twoLargest are the two clinics with the most patients in c (ties by clinic).
func twoLargest(c Contribution) []int32 {
	clinics := sortedClinics(c)
	slices.SortStableFunc(clinics, func(a, b int32) int { return cmp.Compare(len(c[b]), len(c[a])) })
	return clinics[:min(2, len(clinics))]
}
