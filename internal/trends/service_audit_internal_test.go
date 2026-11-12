package trends

import "testing"

// A7 fails closed: a program the solver finds unbounded can only come from a numerical failure (every residual is
// bounded by the program's cap), so the cell is read as 0 wide, never as unbounded.
func TestWidth_AnUnboundedProgramFailsClosed(t *testing.T) {
	p := program{at: []float64{5}, vars: 1, cap: 100} // no row bounds r_0 from above
	if got := p.width(); got != 0 {
		t.Errorf("width = %d, want 0", got)
	}
	bounded := program{g: [][]float64{{1}, {-1}}, h: []float64{40, -10}, at: []float64{20}, vars: 1, cap: 100}
	if got := bounded.width(); got != 30 {
		t.Errorf("width of [10, 40] = %d, want 30", got)
	}
}
