package datarequest

import (
	"context"
	"fmt"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
)

// OperatorTimeZone is the time zone of Salus, the answerer of the requests to the account (S1, S3).
const OperatorTimeZone = "Europe/Sarajevo"

// AgentDeadlines names the deadline job in its audit events (AuditEvent.agent.name): the job, which is no account.
const AgentDeadlines = "data_request_deadlines"

const day = 24 * time.Hour

// dueAt is when a request received now to the clinic org (the account for none) is due: the end of the same date
// DueMonths later in the answerer's time zone (S3).
func (s *Service) dueAt(ctx context.Context, org string) (time.Time, error) {
	loc, err := s.location(ctx, org)
	if err != nil {
		return time.Time{}, err
	}
	return endOfDayMonthsLater(s.p.Clock.Now(), DueMonths, loc), nil
}

// location is the time zone of the clinic org, or the operator's for none.
func (s *Service) location(ctx context.Context, org string) (*time.Location, error) {
	name := OperatorTimeZone
	if org != "" {
		o, err := s.p.Clinics.Clinic(ctx, org)
		if err != nil {
			return nil, wrap("time zone", err)
		}
		if o.TimeZone != "" {
			name = o.TimeZone
		}
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("datarequest: time zone: %w", err)
	}
	return loc, nil
}

// endOfDayMonthsLater is the last second of the date months after t's date in loc: the same day of the month, or the
// month's last day when it has none (31 January, one month on, is the end of February). As Regulation 1182/71 counts
// a period in months.
func endOfDayMonthsLater(t time.Time, months int, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	last := time.Date(y, m+time.Month(months)+1, 0, 0, 0, 0, 0, loc).Day() // day 0: the target month's last day
	return time.Date(y, m+time.Month(months), min(d, last), 23, 59, 59, 0, loc)
}

// addMonthsUTC adds months to t as Postgres adds an interval in UTC: the same time of day, the day of the month
// clamped to the target month's last.
func addMonthsUTC(t time.Time, months int) time.Time {
	t = t.UTC()
	y, m, d := t.Date()
	last := time.Date(y, m+time.Month(months)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return time.Date(y, m+time.Month(months), min(d, last), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
}

// extendedUntil is a request's deadline once extended: the end of the date ExtensionMonths after the due date in loc,
// but never past the due date plus ExtensionMonths as the database's constraint counts them, in UTC (an hour sooner
// when the clocks went back in between).
func extendedUntil(due time.Time, loc *time.Location) time.Time {
	until, limit := endOfDayMonthsLater(due, ExtensionMonths, loc), addMonthsUTC(due, ExtensionMonths)
	if until.After(limit) {
		return limit
	}
	return until
}

// QueueWarnings queues the deadline warnings due now for the open requests of every target (S3): WarnDays before
// each request's deadline, then LastWarnDays before it, each once per deadline (an extension's deadline is warned of
// again), none once the deadline has passed. Each is marked on its request, enqueued and audited as the job's in one
// transaction. It returns how many it queued. Run by the scheduled data_request_deadlines job; the sending is D5's.
func (s *Service) QueueWarnings(ctx context.Context) (int, error) {
	now := s.p.Clock.Now()
	n := 0
	err := s.p.Tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.p.Repo.SetDeadlines(ctx); err != nil {
			return err
		}
		open, err := s.p.Repo.OpenDeadlines(ctx, now.Add(WarnDays*day))
		if err != nil {
			return err
		}
		for _, d := range open {
			days, due := warning(d, now)
			if !due {
				continue
			}
			ok, err := s.p.Repo.MarkWarned(ctx, d.Request.ID, d.WarnedAt, now)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			if err := s.p.Jobs.EnqueueWarning(ctx, WarningArgs{RequestID: d.Request.ID, Days: days}); err != nil {
				return err
			}
			e := event("", d.Request, auditevent.CodeUpdate, auditevent.CodeHOPERAT, 1)
			e.AgentName = AgentDeadlines
			if err := s.p.Audit.Record(ctx, e); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("datarequest: queue warnings: %w", err)
	}
	return n, nil
}

// warning is the warning due now for d, if one is: the latest of the two whose time has come, unless one was queued
// since that time or the deadline has passed.
func warning(d Deadline, now time.Time) (int, bool) {
	deadline := d.Request.Deadline()
	if !now.Before(deadline) {
		return 0, false
	}
	for _, days := range []int{LastWarnDays, WarnDays} {
		at := deadline.Add(-time.Duration(days) * day)
		if !now.Before(at) {
			return days, d.WarnedAt.Before(at)
		}
	}
	return 0, false
}
