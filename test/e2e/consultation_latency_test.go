//go:build e2e && loadtest

package e2e_test

import (
	"fmt"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Shmeeshmee/salus/pkg/clock"
)

// The join endpoint's latency through the running stack (`make bench-join-http`, docs/consultation/PROGRESS.md,
// package 4): joinRequests requests, joinConcurrency at a time, alternating the patient and the practitioner,
// against the api container on the compose Postgres and LiveKit. Not part of `make e2e`.
const (
	joinRequests    = 1000
	joinConcurrency = 10
)

// TestJoinLatency books a synthetic appointment through the API, moves its start 5 minutes ahead (as the
// owner, so the join window is open; the slots start at least an hour ahead), opens the session with one join
// each, then measures the steady state: every request checks the session, decides and audits the join, reads
// the other person's name and calls LiveKit's CreateRoom and signs a token.
func TestJoinLatency(t *testing.T) {
	clinic := newBookableClinic(t)
	p := newPatientWithProfile(t)
	appt := p.bookFirstTime(clinic)
	soon(t, appt.ID)

	path := "/appointments/" + appt.ID + "/consultation/join"
	callers := []*api{p.api, clinic.practitioner}
	for _, c := range callers {
		c.must(http.StatusOK, http.MethodPost, path, nil, nil)
	}

	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: joinConcurrency}}
	took := make([]time.Duration, joinRequests)
	failed := make([]error, joinRequests)
	next := make(chan int)
	var wg sync.WaitGroup
	for range joinConcurrency {
		wg.Go(func() {
			for i := range next {
				took[i], failed[i] = join(t, client, callers[i%2], path)
			}
		})
	}
	began := clock.System{}.Now()
	for i := range joinRequests {
		next <- i
	}
	close(next)
	wg.Wait()
	wall := clock.System{}.Now().Sub(began)
	for i, err := range failed {
		if err != nil {
			t.Fatalf("join %d: %v", i, err)
		}
	}

	var patient, practitioner []time.Duration
	for i, d := range took {
		if i%2 == 0 {
			patient = append(patient, d)
		} else {
			practitioner = append(practitioner, d)
		}
	}
	t.Logf("%d joins, %d concurrent, in %v (%.0f/s)", joinRequests, joinConcurrency, wall.Round(time.Millisecond), float64(joinRequests)/wall.Seconds())
	for _, row := range []struct {
		name string
		d    []time.Duration
	}{{"all", took}, {"patient", patient}, {"practitioner", practitioner}} {
		t.Logf("%-12s p50 %v  p95 %v  p99 %v  max %v", row.name, pct(row.d, 50), pct(row.d, 95), pct(row.d, 99), pct(row.d, 100))
	}
}

// join sends one join request as c and returns how long it took; an error for anything but 200.
func join(t *testing.T, client *http.Client, c *api, path string) (time.Duration, error) {
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, c.base+path, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.bearer)
	began := clock.System{}.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()
	took := clock.System{}.Now().Sub(began)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status %d", resp.StatusCode)
	}
	return took, nil
}

// pct is the p-th percentile of d (nearest rank), rounded to 10 µs.
func pct(d []time.Duration, p int) time.Duration {
	s := slices.Clone(d)
	slices.Sort(s)
	i := max(0, (len(s)*p+99)/100-1)
	return s[i].Round(10 * time.Microsecond)
}

// soon moves the synthetic appointment id to start 5 minutes from now, for 30 minutes, as the owner
// (DATABASE_URL of the tools service).
func soon(t *testing.T, id string) {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), env(t, "DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(t.Context()) }()
	if _, err := conn.Exec(t.Context(), `UPDATE appointment SET period = tstzrange(date_trunc('second', now()) + interval '5 minutes',
		date_trunc('second', now()) + interval '35 minutes') WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
}
