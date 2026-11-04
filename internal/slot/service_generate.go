package slot

import (
	"time"

	"github.com/Shmeeshmee/salus/internal/schedule"
)

// horizon is how far ahead slots are generated (SPEC.md "Slots": 4 weeks).
const horizon = 28 * 24 * time.Hour

// periods are the slots the rules give, slotMinutes long, that start after now and before now + 28 days, by
// start. The rules are times of day in loc: a slot starts at the rule's start and every slot length after,
// while it ends by the rule's end. A local time that does not exist (the spring change) is skipped; one that
// happens twice (autumn) is taken at its first occurrence.
func periods(rules []schedule.Rule, slotMinutes int32, loc *time.Location, now time.Time) []Period {
	until := now.Add(horizon)
	length := time.Duration(slotMinutes) * time.Minute
	local := now.In(loc)
	first := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC) // a calendar date
	var out []Period
	for i := 0; i <= int(horizon/(24*time.Hour))+1; i++ {
		date := first.AddDate(0, 0, i)
		weekday := (int(date.Weekday())+6)%7 + 1 // ISO: Monday 1 to Sunday 7
		for _, r := range rules {
			if r.Day != weekday {
				continue
			}
			start, _ := schedule.Minutes(r.Start)
			end, _ := schedule.Minutes(r.End)
			for m := start; m+int(slotMinutes) <= end; m += int(slotMinutes) {
				t, ok := localTime(date, m, loc)
				if ok && t.After(now) && t.Before(until) {
					out = append(out, Period{Start: t, End: t.Add(length)})
				}
			}
		}
	}
	return out
}

// localTime is the first instant whose time in loc is minutes after midnight on date (a calendar date), and
// false when there is none. It tries the zone's offsets a day before and after: a zone changes its offset at
// most once in two days, so they are the offsets that can apply.
func localTime(date time.Time, minutes int, loc *time.Location) (time.Time, bool) {
	wall := date.Add(time.Duration(minutes) * time.Minute) // the wall time, read as UTC
	var best time.Time
	found := false
	for _, probe := range []time.Time{wall.Add(-24 * time.Hour), wall.Add(24 * time.Hour)} {
		_, offset := probe.In(loc).Zone()
		t := wall.Add(-time.Duration(offset) * time.Second)
		lt := t.In(loc)
		same := lt.Year() == date.Year() && lt.Month() == date.Month() && lt.Day() == date.Day() &&
			lt.Hour()*60+lt.Minute() == minutes && lt.Second() == 0
		if same && (!found || t.Before(best)) {
			best, found = t, true
		}
	}
	return best, found
}
