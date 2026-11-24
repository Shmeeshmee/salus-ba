package datarequest_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/datarequest"
)

// The deadline job queues the 7-day warning once a request's deadline is 7 days away and the 1-day warning once it is
// a day away, each once per deadline, for every target; an extension's new deadline is warned of again (S3).
func TestQueueWarnings_SevenDaysAndOneDayBeforeEachDeadline(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 12, 20, 6, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	f := newFixture(now)
	open := func(id, org string, deadline time.Time) datarequest.Request {
		return datarequest.Request{ID: id, AccountID: accountA, OrganizationID: org, Status: datarequest.StatusReceived, ReceivedAt: deadline.AddDate(0, -1, 0), DueAt: deadline, Version: 1}
	}
	f.repo.put(open("in-6-days", clinicA, now.Add(6*day)))
	f.repo.put(open("in-8-days", clinicA, now.Add(8*day)))
	f.repo.put(open("in-12-hours", "", now.Add(12*time.Hour)))
	extended := open("extended", clinicA, now.Add(-3*day))
	extended.ExtendedUntil = now.Add(5 * day)
	f.repo.put(extended)
	f.repo.warned["extended"] = now.Add(-10 * day) // warned of its first deadline
	answered := open("answered", clinicA, now.Add(day/2))
	answered.Status = datarequest.StatusFulfilled
	f.repo.put(answered)

	n, err := f.svc.QueueWarnings(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	want := []datarequest.WarningArgs{{RequestID: "in-12-hours", Days: 1}, {RequestID: "extended", Days: 7}, {RequestID: "in-6-days", Days: 7}}
	if n != 3 || !slices.Equal(f.jobs.warnings, want) {
		t.Errorf("QueueWarnings = %d, %+v; want %+v", n, f.jobs.warnings, want)
	}
	for _, e := range f.audit.events {
		if e.AgentName != datarequest.AgentDeadlines || e.AgentAccountID != "" || e.EntityName != "data_request" || e.Action != auditevent.ActionUpdate {
			t.Errorf("audit event %+v", e)
		}
	}
	if len(f.audit.events) != 3 {
		t.Errorf("%d audit events, want 3", len(f.audit.events))
	}

	// A second run the same hour queues nothing more. Five days later the 6-day one is a day away and the 8-day one 7
	// days away; the extended one's deadline has passed, and nothing is queued for a deadline past.
	if n, err := f.svc.QueueWarnings(t.Context()); err != nil || n != 0 {
		t.Errorf("a second run: %d, %v; want none", n, err)
	}
	later := newFixtureOn(f.repo, now.Add(5*day+time.Hour))
	if _, err := later.svc.QueueWarnings(t.Context()); err != nil {
		t.Fatal(err)
	}
	want = []datarequest.WarningArgs{{RequestID: "in-6-days", Days: 1}, {RequestID: "in-8-days", Days: 7}}
	if !slices.Equal(later.jobs.warnings, want) {
		t.Errorf("five days later: %+v, want %+v", later.jobs.warnings, want)
	}
}
