package trends_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/Shmeeshmee/salus/internal/trends"
)

// week is a week's contribution: per clinic, that many distinct patients (numbered apart per week and clinic).
func week(w int, perClinic ...int) trends.Contribution {
	c := trends.Contribution{}
	for clinic, n := range perClinic {
		c[int32(clinic)] = patients(int32(w*100000+clinic*1000), n)
	}
	return c
}

// series is ten weeks of four equal clinics with the given totals (each rounded down to a multiple of four).
func series(totals ...int) [10]trends.Contribution {
	var s [10]trends.Contribution
	for i, t := range totals {
		s[i] = week(i, t/4, t/4, t/4, t/4)
	}
	return s
}

// poisson draws a Poisson(λ) count (Knuth's method; λ is small).
func poisson(rng *rand.Rand, lambda float64) int {
	limit, k, p := math.Exp(-lambda), 0, 1.0
	for {
		p *= rng.Float64()
		if p <= limit {
			return k
		}
		k++
	}
}

// §6: a flat series is flagged in at most 1% of 1,000 seeds (four clinics, Poisson(8) each, about 32 a week).
func TestRise_AFlatSeriesIsRarelyFlagged(t *testing.T) {
	flagged := 0
	for seed := range uint64(1000) {
		rng := rand.New(rand.NewPCG(seed, 1))
		var s [10]trends.Contribution
		for i := range s {
			s[i] = week(i, poisson(rng, 8), poisson(rng, 8), poisson(rng, 8), poisson(rng, 8))
		}
		if trends.Rise(s, 10) {
			flagged++
		}
	}
	if flagged > 10 {
		t.Errorf("%d of 1,000 flat series flagged, want at most 10", flagged)
	}
}

func TestRise_FixedSeries(t *testing.T) {
	ramp := make([]int, 10)
	for i := range ramp {
		ramp[i] = int(math.Round(40 * math.Pow(1.05, float64(i))))
	}
	dominated := series(12, 12, 12, 12, 12, 12, 12, 12, 12, 0)
	dominated[9] = week(9, 60, 4, 4, 4) // one clinic holds most of the week
	tests := []struct {
		s    [10]trends.Contribution
		name string
		want bool
	}{
		{series(12, 12, 12, 12, 12, 12, 12, 12, 12, 36), "a step to three times the level", true},
		{series(4, 4, 4, 4, 4, 4, 4, 4, 4, 12), "a step from a baseline under k in all", false},
		{series(40, 40, 40, 40, 40, 40, 40, 40, 40, 8), "a spike below k", false},
		{series(ramp...), "a ramp of 5% a week", false},
		{series(12, 12, 12, 12, 12, 12, 12, 12, 12, 12), "a flat series", false},
		{dominated, "a week dominated by one clinic", false},
		{func() [10]trends.Contribution {
			s := series(12, 12, 12, 12, 12, 12, 12, 12, 12, 36)
			s[9] = week(9, 18, 18) // two clinics only
			return s
		}(), "a rise from two clinics", false},
	}
	for _, tt := range tests {
		if got := trends.Rise(tt.s, 10); got != tt.want {
			t.Errorf("%s: flagged %v, want %v", tt.name, got, tt.want)
		}
	}
}
