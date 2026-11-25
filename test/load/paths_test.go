//go:build loadtest

package load_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// path is one hot path: next gives its i-th request.
type path struct {
	next func(i int) request
	name string
}

// freeSlot is a free slot a booking takes, and the index of its clinic in seeded.clinics.
type freeSlot struct {
	id     string
	clinic int
}

// freeSlots picks n free slots from tomorrow on of the seeded clinics, in a scattered order.
func freeSlots(ctx context.Context, owner *pgxpool.Pool, s seeded, n int) ([]freeSlot, error) {
	index := map[string]int{}
	ids := make([]string, len(s.clinics))
	for i, c := range s.clinics {
		index[c.id], ids[i] = i, c.id
	}
	rows, err := owner.Query(ctx, `SELECT id::text, organization_id::text FROM slot
		WHERE status = 'free' AND lower(period) > now() + interval '1 day' AND organization_id = ANY($1::uuid[])
		ORDER BY hashtext(id::text) LIMIT $2`, ids, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []freeSlot
	for rows.Next() {
		var id, org string
		if err := rows.Scan(&id, &org); err != nil {
			return nil, err
		}
		out = append(out, freeSlot{id: id, clinic: index[org]})
	}
	return out, rows.Err()
}

// instant is t as a FHIR search instant (UTC, so no '+' to escape).
func instant(t time.Time) string { return url.QueryEscape(t.UTC().Format(time.RFC3339)) }

// httpPaths are WAVE7.md RD2's hot paths through the api, in the order of its list, at now. Each request is of
// clinic i modulo the clinics, by its own staff or patients.
func httpPaths(s seeded, slots []freeSlot, now time.Time) []path {
	loc, _ := time.LoadLocation(zone)
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	monday := today.AddDate(0, 0, -((int(today.Weekday()) + 6) % 7))
	clinic := func(i int) seededClinic { return s.clinics[i%len(s.clinics)] }
	role := func(i int) seededRole { c := clinic(i); return c.roles[(i/len(s.clinics))%len(c.roles)] }
	patient := func(i int) seededPatient {
		ps := s.patients[i%len(s.clinics)]
		return ps[(i/len(s.clinics))%len(ps)]
	}
	day := func(t time.Time) string { return t.Format(time.DateOnly) }
	return []path{
		{name: "booking: a practitioner's schedule", next: func(i int) request {
			return request{method: http.MethodGet, path: "/fhir/R4/Schedule?actor=PractitionerRole/" + role(i).id, want: http.StatusOK}
		}},
		{name: "booking: a week's free slots", next: func(i int) request {
			return request{method: http.MethodGet, want: http.StatusOK, path: "/fhir/R4/Slot?schedule=Schedule/" + role(i).schedule +
				"&status=free&start=ge" + instant(now) + "&start=lt" + instant(now.AddDate(0, 0, 7))}
		}},
		{name: "booking: book a slot", next: func(i int) request {
			sl := slots[i%len(slots)]
			ps := s.patients[sl.clinic]
			return request{method: http.MethodPost, path: "/appointments", account: ps[i%len(ps)].account,
				body: `{"slotId":"` + sl.id + `"}`, want: http.StatusCreated}
		}},
		{name: "desk: today", next: func(i int) request {
			return request{method: http.MethodGet, path: "/clinic/desk/today", account: clinic(i).reception, want: http.StatusOK}
		}},
		{name: "calendar: a practitioner's week", next: func(i int) request {
			r := role(i)
			return request{method: http.MethodGet, account: r.account, want: http.StatusOK, path: "/fhir/R4/Appointment?practitioner=Practitioner/" +
				r.practitioner + "&date=ge" + instant(monday) + "&date=lt" + instant(monday.AddDate(0, 0, 7))}
		}},
		{name: "calendar: the clinic's week (admin)", next: func(i int) request {
			return request{method: http.MethodGet, account: clinic(i).admin, want: http.StatusOK,
				path: "/fhir/R4/Appointment?date=ge" + instant(monday) + "&date=lt" + instant(monday.AddDate(0, 0, 7))}
		}},
		{name: "patient: my visits", next: func(i int) request {
			return request{method: http.MethodGet, path: "/fhir/R4/Appointment", account: patient(i).account, want: http.StatusOK}
		}},
		{name: "directory: listed clinics", next: func(int) request {
			return request{method: http.MethodGet, path: "/fhir/R4/Organization?listed=true", want: http.StatusOK}
		}},
		{name: "directory: a clinic's practitioners", next: func(i int) request {
			return request{method: http.MethodGet, want: http.StatusOK,
				path: "/fhir/R4/PractitionerRole?organization=" + clinic(i).id + "&_include=PractitionerRole:practitioner"}
		}},
		{name: "receipts: the last 30 days", next: func(i int) request {
			return request{method: http.MethodGet, account: clinic(i).admin, want: http.StatusOK,
				path: "/clinic/receipts?from=" + instant(now.AddDate(0, 0, -30)) + "&to=" + instant(now)}
		}},
		{name: "receipts report: a month", next: func(i int) request {
			return request{method: http.MethodGet, account: clinic(i).admin, want: http.StatusOK,
				path: "/clinic/reports/receipts?from=" + instant(now.AddDate(0, -1, 0)) + "&to=" + instant(now)}
		}},
		{name: "receipts report: a year (CSV)", next: func(i int) request {
			return request{method: http.MethodGet, account: clinic(i).admin, want: http.StatusOK,
				path: "/clinic/reports/receipts?format=csv&from=" + instant(now.AddDate(-1, 0, 0)) + "&to=" + instant(now)}
		}},
		{name: "clinic statistics: a year's visits", next: func(i int) request {
			return request{method: http.MethodGet, account: clinic(i).admin, want: http.StatusOK,
				path: "/analytics/clinic/appointments?from=" + day(today.AddDate(-1, 0, 1)) + "&to=" + day(today)}
		}},
		{name: "clinic statistics: a year's diagnoses", next: func(i int) request {
			return request{method: http.MethodGet, account: clinic(i).admin, want: http.StatusOK,
				path: "/analytics/clinic/diagnoses?by=group&from=" + day(today.AddDate(-1, 0, 1)) + "&to=" + day(today)}
		}},
	}
}

// reading is a path's latencies, sorted, and its answers that were not the one wanted, by status (0: no answer).
type reading struct {
	failed map[int]int
	name   string
	d      []time.Duration
}

func (r reading) pct(p int) time.Duration {
	if len(r.d) == 0 {
		return 0
	}
	return r.d[min(len(r.d)-1, len(r.d)*p/100)]
}

func (r reading) String() string {
	ms := func(d time.Duration) string { return fmt.Sprintf("%.1f", float64(d)/float64(time.Millisecond)) }
	failed := "none"
	if len(r.failed) > 0 {
		failed = fmt.Sprint(r.failed)
	}
	return fmt.Sprintf("| %s | %d | %s | %s | %s | %s | %s |", r.name, len(r.d), ms(r.pct(50)), ms(r.pct(95)), ms(r.pct(99)), ms(r.d[len(r.d)-1]), failed)
}

// measure runs n calls of fn (its argument the call's index) on conc goroutines and returns their latencies; fn
// returns the call's status (0 for a call that is not HTTP) and whether it was the one wanted.
func measure(ctx context.Context, name string, n, conc int, fn func(ctx context.Context, i int) (int, bool)) reading {
	r := reading{name: name, failed: map[int]int{}}
	var mu sync.Mutex
	work := make(chan int)
	var wg sync.WaitGroup
	for range conc {
		wg.Go(func() {
			for i := range work {
				began := wall.Now()
				status, ok := fn(ctx, i)
				took := wall.Now().Sub(began)
				mu.Lock()
				r.d = append(r.d, took)
				if !ok {
					r.failed[status]++
				}
				mu.Unlock()
			}
		})
	}
	for i := range n {
		work <- i
	}
	close(work)
	wg.Wait()
	slices.Sort(r.d)
	return r
}

// measureHTTP measures path p through the server at base.
func measureHTTP(ctx context.Context, client *http.Client, base string, p path, n, conc int) reading {
	return measure(ctx, p.name, n, conc, func(ctx context.Context, i int) (int, bool) {
		req := p.next(i)
		status, err := do(ctx, client, base, req)
		return status, err == nil && status == req.want
	})
}
