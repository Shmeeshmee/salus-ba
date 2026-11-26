//go:build loadtest

package load_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
)

// budgets are each hot path's median on the small seed (3 clinics, 30 practitioners, 6 000 patients, about 26 000
// visits), about four times what docs/clinic-cms/load/REPORT.md measured there after its fixes, and well under what it
// measured before them where a fix changed the path: a query a row or a range read past its index fails the run, a
// slower CI machine does not. The median, not the p95: on 60 requests the p95 is the third slowest, which one pause of
// the machine (a new pooled connection preparing its statements, another process) moves past any budget. The plans'
// own tests (internal/appointment TestStaffRangeReads_ReadOnlyTheRangesRows and internal/slot
// TestSlotReads_TouchOnlyTheRangesRows) catch an index lost on a path whose small seed is too small to show it in time.
var budgets = map[string]time.Duration{
	"booking: a practitioner's schedule":    10 * time.Millisecond,
	"booking: a week's free slots":          10 * time.Millisecond,
	"booking: book a slot":                  75 * time.Millisecond,
	"desk: today":                           50 * time.Millisecond,  // 69 ms before RD2
	"calendar: a practitioner's week":       50 * time.Millisecond,  // 79 ms before RD2
	"calendar: the clinic's week (admin)":   250 * time.Millisecond, // 585 ms before RD2
	"patient: my visits":                    45 * time.Millisecond,
	"directory: listed clinics":             10 * time.Millisecond,
	"directory: a clinic's practitioners":   10 * time.Millisecond,
	"receipts: the last 30 days":            100 * time.Millisecond,
	"receipts report: a month":              100 * time.Millisecond,
	"receipts report: a year (CSV)":         230 * time.Millisecond,
	"clinic statistics: a year's visits":    100 * time.Millisecond,
	"clinic statistics: a year's diagnoses": 35 * time.Millisecond,
	"slot generation: the hourly run":       5 * time.Second,
}

// remindersBudget is the time from the sweep to the last job done of 200 overdue reminders (3.6 s measured).
const remindersBudget = 30 * time.Second

// TestBudgets is the hot paths' regression test (make load-budget, in CI): the small seed on a throwaway database of
// pgtest (cloned from the migrated template, dropped at the end), every path of TestLoad measured after a warm-up,
// each median within its budget and every answer the one wanted.
func TestBudgets(t *testing.T) {
	ctx := t.Context()
	url := pgtest.NewDatabaseURL(t)
	tg := target{owner: openPool(t, url), app: openPool(t, pgtest.AppURL(t, url)), stats: openPool(t, pgtest.StatsURL(t, url)),
		appURL: pgtest.AppURL(t, url), redisURL: redistest.URL(t)}
	began := wall.Now()
	if err := seed(ctx, tg.owner, tg.stats, small); err != nil {
		t.Fatal(err)
	}
	t.Logf("seeded in %s", wall.Now().Sub(began).Round(time.Millisecond))
	s, err := loadSeeded(ctx, tg.owner, 100)
	if err != nil {
		t.Fatal(err)
	}
	if err := openSessions(ctx, tg.owner, s); err != nil {
		t.Fatal(err)
	}
	srv, stop, err := newServer(ctx, tg)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	const n, conc = 60, 4
	slots, err := freeSlots(ctx, tg.owner, s, 2*n)
	if err != nil || len(slots) < 2*n {
		t.Fatalf("free slots: %d, %v", len(slots), err)
	}
	client := &http.Client{Timeout: time.Minute, Transport: &http.Transport{MaxIdleConnsPerHost: conc}}
	for _, p := range httpPaths(s, slots, wall.Now()) {
		warm := p
		if p.name == "booking: book a slot" {
			warm.next = func(i int) request { return p.next(n + i) }
		}
		measureHTTP(ctx, client, srv.URL, warm, 2*conc, conc)
		r := measureHTTP(ctx, client, srv.URL, p, n, conc)
		t.Log(r)
		check(t, r)
	}
	sl, err := measureSlots(ctx, tg, 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(sl)
	check(t, sl)
	rl, done, err := reminderReadings(ctx, tg, 200)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range rl {
		t.Log(l)
	}
	if done > remindersBudget {
		t.Errorf("reminders: dispatched and sent in %s, past their budget of %s", done, remindersBudget)
	}
}

// check fails r when an answer was not the one wanted or its median is past its budget.
func check(t *testing.T, r reading) {
	t.Helper()
	if len(r.failed) > 0 {
		t.Errorf("%s: answers not as wanted, by status: %v", r.name, r.failed)
	}
	b, ok := budgets[r.name]
	if !ok {
		t.Errorf("%s has no budget", r.name)
		return
	}
	if p50 := r.pct(50); p50 > b {
		t.Errorf("%s: median %s, past its budget of %s", r.name, p50, b)
	}
}
