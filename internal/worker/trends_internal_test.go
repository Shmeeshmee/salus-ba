package worker

import (
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// §2a: off, the scheduler has no trends build; on, the month, the quarter and the week.
func TestTrendsPeriodics_OnlyWhileThePlatformViewIsOn(t *testing.T) {
	if got := trendsPeriodics(false, clock.System{}); len(got) != 0 {
		t.Errorf("off: %d builds, want none", len(got))
	}
	if got := trendsPeriodics(true, clock.System{}); len(got) != 3 {
		t.Errorf("on: %d builds, want the month, the quarter and the week", len(got))
	}
}

func TestTrendsSchedules(t *testing.T) {
	at := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	for _, tt := range []struct {
		next interface{ Next(time.Time) time.Time }
		now  string
		want string
	}{
		{monthlyOn{day: 8, hour: 4}, "2026-11-12T10:00:00Z", "2026-12-08T04:00:00Z"},
		{monthlyOn{day: 8, hour: 4}, "2026-11-08T03:59:00Z", "2026-11-08T04:00:00Z"},
		{monthlyOn{day: 8, hour: 4, quarterly: true}, "2026-11-12T10:00:00Z", "2027-01-08T04:00:00Z"},
		{monthlyOn{day: 8, hour: 4, quarterly: true}, "2026-10-08T04:00:00Z", "2027-01-08T04:00:00Z"},
		{weeklyOn{weekday: time.Tuesday, hour: 4}, "2026-11-12T10:00:00Z", "2026-11-17T04:00:00Z"},
		{weeklyOn{weekday: time.Tuesday, hour: 4}, "2026-11-17T03:00:00+01:00", "2026-11-17T04:00:00Z"},
	} {
		if got := tt.next.Next(at(tt.now)); !got.Equal(at(tt.want)) {
			t.Errorf("%T%+v.Next(%s) = %v, want %s", tt.next, tt.next, tt.now, got, tt.want)
		}
	}
	job := trendsPeriodics(true, clock.Fixed{T: at("2026-11-08T04:00:00Z")})
	if len(job) != 3 {
		t.Fatal("no builds")
	}
}
