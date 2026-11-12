package trends

import (
	"context"
	"math"
	"slices"
	"strings"
)

// riseWeeks are the weeks a rise is computed over (§6): w-9 to w-2 the baseline, w-1 skipped, w the week.
const riseWeeks = 10

// Rise reports whether the last of ten weeks is an unusual rise (§6, an EARS C2 adaptation): weeks[0] to weeks[7]
// are the baseline (w-9 to w-2), weeks[8] (w-1) is skipped, weeks[9] is the week w. Flagged when the week's cell
// passes all of A4, the baseline holds at least k patients in all, and the week's count c is at least 1.5·μ and
// above μ + 3σ', with μ the baseline mean, σ its sample standard deviation and σ' = max(σ, √μ, 1).
func Rise(weeks [riseWeeks]Contribution, k int) bool {
	if !passes(weeks[9], k) {
		return false
	}
	var counts [8]float64
	sum := 0.0
	for i := range counts {
		counts[i] = float64(len(union(weeks[i])))
		sum += counts[i]
	}
	if sum < float64(k) {
		return false
	}
	mu := sum / 8
	variance := 0.0
	for _, c := range counts {
		variance += (c - mu) * (c - mu)
	}
	sigma := math.Max(math.Max(math.Sqrt(variance/7), math.Sqrt(mu)), 1)
	c := float64(len(union(weeks[9])))
	return c >= 1.5*mu && c > mu+3*sigma
}

// week builds an ISO week's rise flags (§6) per leaf region and diagnosis group, over the clinics that counted, with
// the same location, through all ten weeks: a clinic joining, leaving or moving never raises a flag by itself.
func (b *Builder) week(ctx context.Context, p span, src sources, key []byte) (Summary, error) {
	first := p.weeksBack(riseWeeks - 1)
	type cellKey struct{ region, group string }
	series := map[cellKey]*[riseWeeks]cells{}
	people, read := interner{}, 0
	for i, loc := range src.locations {
		z := zone(loc)
		from := first.start(z)
		if !counts(loc, from, src.entity) {
			continue
		}
		recs, err := b.p.Diagnoses.Recorded(ctx, loc.ID, from, p.end(z), key)
		if err != nil {
			return Summary{}, err
		}
		read += len(recs)
		for _, r := range recs {
			w := weekly(r.RecordedAt.In(z), from, riseWeeks)
			if w < 0 {
				continue
			}
			group, _ := src.groups.Of(r.Code)
			ck := cellKey{loc.Region, group}
			if series[ck] == nil {
				series[ck] = &[riseWeeks]cells{}
			}
			if series[ck][w] == nil {
				series[ck][w] = cells{}
			}
			series[ck][w].add(Key{}, int32(i), people.of(r.Patient))
		}
	}
	if err := b.audit(ctx, "condition", read); err != nil {
		return Summary{}, err
	}
	var flags []Flag
	for ck, s := range series {
		var weeks [riseWeeks]Contribution
		for w, c := range s {
			weeks[w] = c.contributions()[Key{}]
		}
		if Rise(weeks, b.p.K) {
			flags = append(flags, Flag{Week: p.name, Region: ck.region, Group: ck.group})
		}
	}
	slices.SortFunc(flags, func(a, b Flag) int {
		if a.Region != b.Region {
			return strings.Compare(a.Region, b.Region)
		}
		return strings.Compare(a.Group, b.Group)
	})
	period := Period{Period: p.name, Kind: KindWeek, Status: StatusPublished, MinCount: b.p.K, CodeLevel: false}
	if err := b.save(ctx, period, nil, flags); err != nil {
		return Summary{}, err
	}
	return Summary{Rises: len(flags)}, nil
}
