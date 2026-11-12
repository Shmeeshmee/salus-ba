package trends

import (
	"fmt"
	"strconv"
	"time"
)

// span is a period: its kind, its first day (the clinics' local calendar) and where it ends.
type span struct {
	name  string
	kind  string
	year  int
	month time.Month // the first month of a month or a quarter
	week  int        // an ISO week
}

// parsePeriod reads 2026-10 (a month), 2026-Q3 (a quarter) or 2026-W44 (an ISO week).
func parsePeriod(s string) (span, error) {
	if len(s) < 7 || s[4] != '-' {
		return span{}, ErrInvalidPeriod
	}
	year, err := strconv.Atoi(s[:4])
	if err != nil || year < 2000 {
		return span{}, ErrInvalidPeriod
	}
	p := span{name: s, year: year}
	switch {
	case len(s) == 7 && s[5] == 'Q':
		q, err := strconv.Atoi(s[6:])
		if err != nil || q < 1 || q > 4 {
			return span{}, ErrInvalidPeriod
		}
		p.kind, p.month = KindQuarter, time.Month(3*q-2)
	case len(s) == 8 && s[5] == 'W':
		w, err := strconv.Atoi(s[6:])
		if err != nil || w < 1 {
			return span{}, ErrInvalidPeriod
		}
		if y, iw := isoMonday(year, w).ISOWeek(); y != year || iw != w {
			return span{}, ErrInvalidPeriod
		}
		p.kind, p.week = KindWeek, w
	case len(s) == 7:
		m, err := strconv.Atoi(s[5:])
		if err != nil || m < 1 || m > 12 || s[5] == '+' || s[5] == '-' {
			return span{}, ErrInvalidPeriod
		}
		p.kind, p.month = KindMonth, time.Month(m)
	default:
		return span{}, ErrInvalidPeriod
	}
	return p, nil
}

// isoMonday is the Monday of ISO week w of year, as a UTC date.
func isoMonday(year, w int) time.Time {
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	return jan4.AddDate(0, 0, -((int(jan4.Weekday())+6)%7)+7*(w-1))
}

// start is the period's first instant in loc; a week's start is its Monday.
func (p span) start(loc *time.Location) time.Time {
	if p.kind == KindWeek {
		m := isoMonday(p.year, p.week)
		return time.Date(m.Year(), m.Month(), m.Day(), 0, 0, 0, 0, loc)
	}
	return time.Date(p.year, p.month, 1, 0, 0, 0, 0, loc)
}

// end is the instant after the period in loc.
func (p span) end(loc *time.Location) time.Time {
	s := p.start(loc)
	switch p.kind {
	case KindQuarter:
		return s.AddDate(0, 3, 0)
	case KindWeek:
		return s.AddDate(0, 0, 7)
	}
	return s.AddDate(0, 1, 0)
}

// due is when the period may be built (A3, §6): the 8th after a month or a quarter (seven days after it ends), the
// Tuesday after a week, at 00:00 UTC; never before every clinic's period has ended.
func (p span) due() time.Time {
	end := p.end(time.UTC)
	if p.kind == KindWeek {
		return end.AddDate(0, 0, 1)
	}
	return end.AddDate(0, 0, 7)
}

// weeksBack is the ISO week n weeks before p (a week).
func (p span) weeksBack(n int) span {
	y, w := isoMonday(p.year, p.week).AddDate(0, 0, -7*n).ISOWeek()
	return span{name: fmt.Sprintf("%04d-W%02d", y, w), kind: KindWeek, year: y, week: w}
}

// PreviousMonth, PreviousQuarter and PreviousWeek name the period before the one t falls in (UTC), for the
// scheduler: on the 8th the month before, on the 8th after a quarter the quarter before, on a Tuesday the week before.
func PreviousMonth(t time.Time) string {
	m := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	return fmt.Sprintf("%04d-%02d", m.Year(), int(m.Month()))
}

func PreviousQuarter(t time.Time) string {
	q := time.Date(t.Year(), t.Month()-(t.Month()-1)%3, 1, 0, 0, 0, 0, time.UTC).AddDate(0, -3, 0)
	return fmt.Sprintf("%04d-Q%d", q.Year(), (int(q.Month())-1)/3+1)
}

func PreviousWeek(t time.Time) string {
	y, w := t.UTC().AddDate(0, 0, -7).ISOWeek()
	return fmt.Sprintf("%04d-W%02d", y, w)
}
