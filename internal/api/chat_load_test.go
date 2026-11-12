//go:build loadtest

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/appointment"
	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/communication"
	"github.com/Shmeeshmee/salus/internal/consent"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitioner"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
	"github.com/Shmeeshmee/salus/pkg/redis/redistest"
	"github.com/Shmeeshmee/salus/pkg/wire"
)

// The chat's targets (SAL-27): the latest page of a conversation under 200 ms p95 with millions of messages in the
// table, and a message delivered (a send to the frame on another participant's socket) under 300 ms p95. make
// bench-chat runs this against the compose Postgres and Redis, on the chat as newChat composes it for the api (every
// service real: the role lookup, the consent check, the audit write); the numbers go in docs/plans/chat.PROGRESS.md.
//
// The table: CHAT_LOAD_MESSAGES synthetic messages (default 3 000 000) over 24 monthly partitions, in 50 000 small
// conversations of 20 to 60 messages and 10 long ones sharing the rest (about 100 000 each). One practitioner takes
// part in all of them, with each patient's active consent. pgtest caps the pool at 8 connections; the api's pool has
// one per CPU, so the delivery numbers are pessimistic.

const (
	loadSmall   = 50_000
	loadLong    = 10
	loadSamples = 1000
	// loadWords is the size of the synthetic vocabulary the bodies draw one word from ("w0" to "w4999", SAL-30's
	// search): a word is in about one message in 5000, "placeholder" in every one.
	loadWords = 5000

	loadClinic       = "a0000000-0000-4000-8000-00000000000a"
	loadRole         = "a1000000-0000-4000-8000-0000000000a1"
	loadPractitioner = "b1000000-0000-4000-8000-0000000000b1"
	loadPractAccount = "ac100000-0000-4000-8000-0000000000a1"
)

// loadWall is the wall clock the measurements read.
var loadWall = clock.System{}

var loadPract = appointment.Caller{AccountID: loadPractAccount, OrganizationID: loadClinic, PractitionerID: loadPractitioner}

func loadMessages(t *testing.T) int {
	if s := os.Getenv("CHAT_LOAD_MESSAGES"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	return 3_000_000
}

// loadPatient is the caller of the i-th seeded patient (1-based, conversation i is theirs).
func loadPatient(i int) appointment.Caller {
	return appointment.Caller{AccountID: fmt.Sprintf("10000000-0000-4000-8000-%012x", i), PatientID: fmt.Sprintf("20000000-0000-4000-8000-%012x", i)}
}

func loadConversation(i int) string { return fmt.Sprintf("30000000-0000-4000-8000-%012x", i) }

// seedLoad fills the database as the owner.
func seedLoad(t *testing.T, owner *pgxpool.Pool, total int) {
	t.Helper()
	ctx := t.Context()
	began := loadWall.Now()
	smallTotal := loadSmall * 40
	perLong := max((total-smallTotal)/loadLong, 1200) // page 22 exists
	if _, err := owner.Exec(ctx, `WITH o AS (INSERT INTO organization (id, name) VALUES ($1, 'Synthetic clinic') RETURNING id),
		p AS (INSERT INTO practitioner (id, organization_id, given_name, family_name) SELECT $3, id, 'Cy', 'Example' FROM o RETURNING id, organization_id),
		r AS (INSERT INTO practitioner_role (id, organization_id, practitioner_id) SELECT $2, organization_id, id FROM p RETURNING organization_id)
		INSERT INTO account (id, email, password_hash, role, organization_id, practitioner_id) SELECT $4, 'p1@example.test', 'h', 'practitioner', organization_id, $3 FROM r`,
		loadClinic, loadRole, loadPractitioner, loadPractAccount); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`SELECT communication_add_partitions('2024-11-01', 24)`,
		// The patients, each with an account and an active care consent with clinic A.
		`INSERT INTO account (id, email, password_hash, role)
			SELECT ('10000000-0000-4000-8000-' || lpad(to_hex(i), 12, '0'))::uuid, 'load' || i || '@example.test', 'h', 'patient'
			FROM generate_series(1, ` + strconv.Itoa(loadSmall+loadLong) + `) i`,
		`INSERT INTO patient (id, account_id, given_name, family_name, birth_date)
			SELECT ('20000000-0000-4000-8000-' || lpad(to_hex(i), 12, '0'))::uuid, ('10000000-0000-4000-8000-' || lpad(to_hex(i), 12, '0'))::uuid,
				'Synthetic', 'Patient', '1990-01-01'
			FROM generate_series(1, ` + strconv.Itoa(loadSmall+loadLong) + `) i`,
		`INSERT INTO consent (patient_id, organization_id, status, date_time) SELECT id, '` + loadClinic + `', 'active', '2024-10-01' FROM patient WHERE id::text LIKE '20000000-%'`,
		// One conversation per patient with role A1: the first loadLong are the long ones.
		`INSERT INTO conversation (id, organization_id, patient_id, practitioner_role_id, created_by, created_at, last_seq, last_sent_at)
			SELECT ('30000000-0000-4000-8000-' || lpad(to_hex(i), 12, '0'))::uuid, '` + loadClinic + `',
				('20000000-0000-4000-8000-' || lpad(to_hex(i), 12, '0'))::uuid, '` + loadRole + `', '` + loadPractAccount + `', '2024-11-01', 0, NULL
			FROM generate_series(1, ` + strconv.Itoa(loadSmall+loadLong) + `) i`,
		`INSERT INTO conversation_participant (conversation_id, patient_id, role, added_by, added_at)
			SELECT id, patient_id, 'patient', created_by, created_at FROM conversation`,
		`INSERT INTO conversation_participant (conversation_id, practitioner_role_id, organization_id, role, added_by, added_at)
			SELECT id, practitioner_role_id, organization_id, 'practitioner', created_by, created_at FROM conversation`,
		// The messages: n per conversation, spread evenly from 2024-11-01 over 24 months, alternating sender.
		`WITH c AS (
			SELECT cv.id, cv.patient_id, row_number() OVER (ORDER BY cv.id) AS i,
				p.id AS patient_part, d.id AS doctor_part
			FROM conversation cv
			JOIN conversation_participant p ON p.conversation_id = cv.id AND p.role = 'patient'
			JOIN conversation_participant d ON d.conversation_id = cv.id AND d.role = 'practitioner'),
		n AS (SELECT c.*, CASE WHEN i <= ` + strconv.Itoa(loadLong) + ` THEN ` + strconv.Itoa(perLong) + ` ELSE 20 + (i % 41) END AS n FROM c)
		INSERT INTO communication (conversation_id, organization_id, patient_id, staff_only, seq, sent_at, sender_participant_id,
			sender_account_id, sender_role, client_message_id, kind, body)
		SELECT n.id, '` + loadClinic + `', n.patient_id, false, s,
			timestamptz '2024-11-01' + (interval '729 days') * s / (n.n + 1) + (n.i % 1000) * interval '1 second',
			CASE WHEN s % 2 = 0 THEN n.patient_part ELSE n.doctor_part END,
			CASE WHEN s % 2 = 0 THEN ('10000000-0000-4000-8000-' || lpad(to_hex(n.i), 12, '0'))::uuid ELSE '` + loadPractAccount + `'::uuid END,
			CASE WHEN s % 2 = 0 THEN 'patient' ELSE 'practitioner' END, gen_random_uuid(), 'text',
			'Synthetic placeholder message ' || s || ' w' || ((s * 7919 + n.i * 104729) % ` + strconv.Itoa(loadWords) + `)
		FROM n, generate_series(1, n.n) s`,
		`UPDATE conversation cv SET last_seq = m.seq, last_sent_at = m.sent_at
			FROM (SELECT DISTINCT ON (conversation_id) conversation_id, seq, sent_at FROM communication ORDER BY conversation_id, seq DESC) m
			WHERE m.conversation_id = cv.id`,
		`VACUUM (ANALYZE) communication`, // the bulk load's autovacuum and write-back are not the chat's
		`VACUUM (ANALYZE) conversation`,
		`VACUUM (ANALYZE) conversation_participant`,
		`CHECKPOINT`,
	} {
		if _, err := owner.Exec(ctx, stmt); err != nil {
			t.Fatalf("seed: %v\n%s", err, stmt)
		}
	}
	var rows int64
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM communication`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	t.Logf("seeded %d messages in %d conversations over 24 partitions in %v", rows, loadSmall+loadLong, loadWall.Now().Sub(began).Round(time.Second))
}

func percentile(d []time.Duration, p float64) time.Duration {
	s := slices.Clone(d)
	slices.Sort(s)
	return s[min(int(float64(len(s))*p), len(s)-1)]
}

func report(t *testing.T, name string, d []time.Duration, target time.Duration) {
	t.Helper()
	if len(d) == 0 {
		t.Fatalf("%s: no samples", name)
	}
	p50, p95, p99 := percentile(d, 0.5), percentile(d, 0.95), percentile(d, 0.99)
	if target == 0 {
		t.Logf("%s: n=%d p50=%v p95=%v p99=%v max=%v (capacity, no target)", name, len(d), p50, p95, p99, percentile(d, 1))
		return
	}
	t.Logf("%s: n=%d p50=%v p95=%v p99=%v max=%v (target p95 < %v)", name, len(d), p50, p95, p99, percentile(d, 1), target)
	if p95 >= target {
		t.Errorf("%s: p95 %v, target under %v", name, p95, target)
	}
}

// loadChat is the chat as the api composes it, on pool and Redis, for the caller the request's X-Load-As header
// names ("pract", or a patient's number); its routes and socket on one mux.
func loadChat(t *testing.T, pool *pgxpool.Pool) (*communication.Service, http.Handler) {
	t.Helper()
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clock.System{})
	clinics := organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clock.System{})
	roles := practitionerrole.NewService(practitionerrole.Ports{
		Repo: practitionerrole.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}, Clinics: clinics,
		People: practitioner.NewService(practitioner.NewPostgresRepository(pool), audit, tx),
	})
	consents := consent.NewService(consent.NewPostgresRepository(pool), audit, tx, clock.System{}, nil)
	caller := func(ctx context.Context) (appointment.Caller, bool, error) {
		as, _ := ctx.Value(loadAsKey{}).(string)
		if as == "pract" {
			return loadPract, true, nil
		}
		n, err := strconv.Atoi(as)
		return loadPatient(n), err == nil, nil
	}
	log := slog.New(slog.DiscardHandler)
	routes, stream, hub := newChat(chatPorts{
		db: pool, rdb: redistest.NewClient(t), audit: audit, tx: tx, consents: consents,
		visits: appointment.NewFacts(appointment.NewPostgresRepository(pool)), roles: roles, caller: caller,
		active: func(*http.Request) (bool, error) { return true, nil },
	}, log)
	t.Cleanup(func() { _ = hub.Close() })
	mux := http.NewServeMux()
	routes(mux)
	stream(mux)
	svc := communication.NewService(communication.Ports{
		Repo: communication.NewPostgresRepository(pool), Publisher: hub, Consents: consents,
		Visits: appointment.NewFacts(appointment.NewPostgresRepository(pool)), Roles: roles, Audit: audit, Tx: tx,
		Clock: clock.System{}, Log: log,
	})
	return svc, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), loadAsKey{}, r.Header.Get("X-Load-As"))))
	})
}

type loadAsKey struct{}

func TestChatLoad(t *testing.T) {
	owner, app := pgtest.NewAppDatabase(t)
	seedLoad(t, owner, loadMessages(t))
	svc, h := loadChat(t, app)
	ctx := context.Background()
	long := func() string { return loadConversation(1 + rand.IntN(loadLong)) }
	small := func() string { return loadConversation(loadLong + 1 + rand.IntN(loadSmall)) }

	explain(t, owner, `EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id, seq FROM communication
		WHERE conversation_id = $1 AND seq > 0 ORDER BY sent_at DESC, seq DESC LIMIT 51`, long())
	explain(t, owner, `EXPLAIN (ANALYZE, BUFFERS, COSTS OFF) SELECT id FROM communication
		WHERE conversation_id = $1 AND client_message_id = 'c1000000-0000-4000-8000-0000000000c1' AND sender_participant_id = 'd1000000-0000-4000-8000-0000000000d1'
		AND sent_at >= (SELECT last_sent_at - interval '1 day' FROM conversation WHERE id = $1)`, long())
	t.Run("latest page", func(t *testing.T) {
		var d []time.Duration
		for i := range loadSamples {
			id := small()
			if i%10 == 0 {
				id = long()
			}
			began := loadWall.Now()
			page, err := svc.Messages(ctx, loadPract, id, nil, 0)
			d = append(d, loadWall.Now().Sub(began))
			if err != nil || len(page.Messages) == 0 {
				t.Fatalf("%s: %v, %d messages", id, err, len(page.Messages))
			}
		}
		report(t, "latest page (service, audited, 50 messages)", d, 200*time.Millisecond)
	})
	t.Run("deep page of a long conversation", func(t *testing.T) {
		var d []time.Duration
		for range loadSamples / 4 {
			id := long()
			page, err := svc.Messages(ctx, loadPract, id, nil, 0)
			for range 20 { // twenty pages back: the cursor's cost does not grow with depth
				if err != nil || page.Next == nil {
					t.Fatalf("%v", err)
				}
				page, err = svc.Messages(ctx, loadPract, id, page.Next, 0)
			}
			began := loadWall.Now()
			_, err = svc.Messages(ctx, loadPract, id, page.Next, 0)
			d = append(d, loadWall.Now().Sub(began))
			if err != nil {
				t.Fatal(err)
			}
		}
		report(t, "page 22 of a long conversation", d, 200*time.Millisecond)
	})
	t.Run("send alone", func(t *testing.T) {
		var d []time.Duration
		for i := range 200 {
			n := loadLong + loadSmall - i
			began := loadWall.Now()
			if _, _, err := svc.Send(ctx, loadPatient(n), loadConversation(n), fmt.Sprintf("aaaaaaaa-0000-4000-8000-%012x", i), "Synthetic placeholder"); err != nil {
				t.Fatal(err)
			}
			d = append(d, loadWall.Now().Sub(began))
		}
		report(t, "a send with nothing else running", d, 300*time.Millisecond)
	})
	searches(t, owner, svc, long, small)
	// The targets hold at 200 messages a second (720 000 an hour, far above the clinics' load); 400 and 1000 a second
	// are measured as the capacity of durable commits on the test's pool of 8 (docs/plans/chat.PROGRESS.md).
	for k, rate := range []int{1, 2, 5} { // messages a second per conversation, 200 conversations
		t.Run(fmt.Sprintf("delivery at %d a second", 200*rate), func(t *testing.T) {
			stop := sampleWaits(t, owner)
			delivery(t, svc, h, loadLong+1+k*200, rate)
			stop()
		})
	}
}

// sampleWaits counts, every 20 ms until stop, what the database's active backends wait on, and logs the counts:
// where a send's time goes under load (CPU, a lock, the WAL's fsync, a client).
func sampleWaits(t *testing.T, owner *pgxpool.Pool) (stop func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	counts := map[string]int{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			rows, err := owner.Query(ctx, `SELECT coalesce(wait_event_type, 'CPU') || ':' || coalesce(wait_event, '-') FROM pg_stat_activity
				WHERE datname = current_database() AND state = 'active' AND pid <> pg_backend_pid()`)
			if err != nil {
				continue
			}
			for rows.Next() {
				var w string
				if rows.Scan(&w) == nil {
					counts[w]++
				}
			}
			rows.Close()
		}
	}()
	return func() {
		cancel()
		<-done
		keys := slices.SortedFunc(func(yield func(string) bool) {
			for k := range counts {
				if !yield(k) {
					return
				}
			}
		}, func(a, b string) int { return counts[b] - counts[a] })
		var top []string
		for _, k := range keys[:min(len(keys), 8)] {
			top = append(top, fmt.Sprintf("%s=%d", k, counts[k]))
		}
		t.Logf("backend waits (samples every 20 ms): %s", strings.Join(top, " "))
	}
}

// explain logs the plan of query on a long conversation: the latest page reads the newest partition first and stops
// (ADR 0012); the client id's lookup probes every partition.
func explain(t *testing.T, owner *pgxpool.Pool, query, id string) {
	rows, err := owner.Query(t.Context(), query, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var plan []string
	for rows.Next() {
		var line string
		_ = rows.Scan(&line)
		plan = append(plan, line)
	}
	t.Logf("plan on a long conversation:\n%s", strings.Join(plan, "\n"))
}

// delivery opens the practitioner's socket on each of 200 conversations (from the first-th) through the chat's own
// handler and hub, and has their patients send 20 messages each, rate a second per conversation, each sender with its
// own phase, measuring from each send's call to its frame on the socket, and the send's own time (its commit).
func delivery(t *testing.T, svc *communication.Service, h http.Handler, first, rate int) {
	target := 300 * time.Millisecond
	if rate > 1 {
		target = 0
	}
	const convs, perConv = 200, 20
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	var mu sync.Mutex
	sentAt := map[string]time.Time{}
	var d, commits []time.Duration
	var readers sync.WaitGroup
	for i := range convs {
		conn, res, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/conversations/"+loadConversation(first+i)+"/events",
			http.Header{"X-Load-As": {"pract"}})
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		t.Cleanup(func() { _ = conn.Close() })
		readers.Go(func() {
			for got := 0; got < perConv; {
				_ = conn.SetReadDeadline(loadWall.Now().Add(30 * time.Second))
				_, b, err := conn.ReadMessage()
				if err != nil {
					t.Errorf("socket %d: %v", i, err)
					return
				}
				at := loadWall.Now()
				env, err := wire.Decode(b)
				if err != nil || env.Type != communication.MessageType {
					continue
				}
				var m struct {
					ClientMessageID string `json:"client_message_id"`
				}
				_ = json.Unmarshal(env.Payload, &m)
				mu.Lock()
				if s, ok := sentAt[m.ClientMessageID]; ok {
					d = append(d, at.Sub(s))
					got++
				}
				mu.Unlock()
			}
		})
	}
	time.Sleep(200 * time.Millisecond) // the subscriptions reach Redis
	var senders sync.WaitGroup
	for i := range convs {
		senders.Go(func() {
			time.Sleep(rand.N(time.Second / time.Duration(rate))) // each sender its own phase: no lockstep bursts
			tick := time.NewTicker(time.Second / time.Duration(rate))
			defer tick.Stop()
			for n := range perConv {
				<-tick.C
				client := fmt.Sprintf("%08x-0000-4000-8000-%012x", first+i, n)
				mu.Lock()
				sentAt[client] = loadWall.Now()
				mu.Unlock()
				began := loadWall.Now()
				if _, _, err := svc.Send(context.Background(), loadPatient(first+i), loadConversation(first+i), client, "Synthetic placeholder load message"); err != nil {
					t.Errorf("send: %v", err)
					return
				}
				mu.Lock()
				commits = append(commits, loadWall.Now().Sub(began))
				mu.Unlock()
			}
		})
	}
	senders.Wait()
	readers.Wait()
	report(t, fmt.Sprintf("send's commit (%d conversations, %d messages a second)", convs, convs*rate), commits, target)
	report(t, fmt.Sprintf("delivery (%d conversations, %d messages a second)", convs, convs*rate), d, target)
}
