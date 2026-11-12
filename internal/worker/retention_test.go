package worker_test

import (
	"bufio"
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/videosession"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

// syncBuffer collects a client's log output; River writes it while the test reads it.
type syncBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// logins are four pools on one test database: the owner, salus_app, salus_retention and salus_stats.
type logins struct {
	owner, app, retention, stats *pgxpool.Pool
}

func newLogins(t *testing.T) logins {
	t.Helper()
	dbURL := pgtest.NewDatabaseURL(t)
	open := func(u string) *pgxpool.Pool {
		pool, err := postgres.Open(t.Context(), u)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(pool.Close)
		return pool
	}
	return logins{
		owner: open(dbURL), app: open(pgtest.AppURL(t, dbURL)), retention: open(pgtest.RetentionURL(t, dbURL)),
		stats: open(pgtest.StatsURL(t, dbURL)),
	}
}

// The synthetic personal data of seedRetention, which no log line may hold.
var seeded = []string{"ana@example.test", "bo@example.test", "cy@example.test", "Bo", "Example"}

// seedRetention creates, as the owner, a patient with a session that ended 31 days ago and one still open,
// and a clinic with an invite accepted 31 days ago and one still open.
func seedRetention(t *testing.T, owner *pgxpool.Pool) {
	t.Helper()
	_, err := owner.Exec(t.Context(), `
		INSERT INTO organization (id, name) VALUES ('`+clinicID+`', 'Synthetic clinic');
		WITH a AS (INSERT INTO account (email, password_hash, role) VALUES ('ana@example.test', 'h', 'patient') RETURNING id)
		INSERT INTO session (account_id, token_hash, mfa_verified, expires_at, revoked_at)
			SELECT id, 'old'::bytea, false, now() + interval '1 day', now() - interval '31 days' FROM a
			UNION ALL SELECT id, 'open'::bytea, false, now() + interval '1 day', NULL FROM a;
		INSERT INTO invite (organization_id, expires_at, accepted_at, role, email, given_name, family_name)
			VALUES ('`+clinicID+`', now() - interval '29 days', now() - interval '31 days', 'practitioner', 'bo@example.test', 'Bo', 'Example'),
			       ('`+clinicID+`', now() + interval '1 day', NULL, 'admin', 'cy@example.test', '', '');`)
	if err != nil {
		t.Fatal(err)
	}
}

// runRetention starts the worker on db, enqueues retention_apply the way the scheduler does, and returns how
// the job's first attempt ended (completed or failed) and the worker's log.
func runRetention(t *testing.T, db logins) (river.EventKind, string) {
	t.Helper()
	return runWorkerJob(t, db, worker.RetentionApplyArgs{})
}

// runWorkerJob starts the worker on db, enqueues args, and returns how the job's first attempt ended (completed,
// failed or cancelled) and the worker's log.
func runWorkerJob(t *testing.T, db logins, args river.JobArgs) (river.EventKind, string) {
	t.Helper()
	return runWorkerJobWith(t, db, liveKit(t), args)
}

// runWorkerJobWith is runWorkerJob with the worker on the video provider p; a snooze ends the run too.
func runWorkerJobWith(t *testing.T, db logins, p videosession.Provider, args river.JobArgs) (river.EventKind, string) {
	t.Helper()
	return runJobOn(t, db, p, testFiles(t), args, nil)
}

// runJobOn starts the worker on db with the video provider p and the files' dependencies, enqueues args with
// opts, and returns how the job's first attempt ended (completed, failed, cancelled or snoozed) and the log.
func runJobOn(t *testing.T, db logins, p videosession.Provider, files worker.Files, args river.JobArgs, opts *river.InsertOpts) (river.EventKind, string) {
	t.Helper()
	logs := &syncBuffer{}
	client, err := worker.New(db.app, db.retention, db.stats, p, files, webhookKey, slog.New(slog.NewJSONHandler(logs, nil)))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	events, cancel := client.Subscribe(river.EventKindJobCompleted, river.EventKindJobFailed, river.EventKindJobCancelled, river.EventKindJobSnoozed)
	defer cancel()
	if err := client.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Stop(context.WithoutCancel(t.Context())) }()
	inserted, err := client.Insert(t.Context(), args, opts)
	if err != nil {
		t.Fatalf("Insert() error = %v", err)
	}
	timeout := time.After(15 * time.Second)
	for { // the job may enqueue others, which the client works too: wait for this one's end
		select {
		case ev := <-events:
			if ev.Job.ID == inserted.Job.ID {
				return ev.Kind, logs.String()
			}
		case <-timeout:
			t.Fatalf("the %s job did not finish within 15s", args.Kind())
			return "", ""
		}
	}
}

func count(t *testing.T, owner *pgxpool.Pool, table string) int {
	t.Helper()
	var n int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRetentionApply_DeletesEndedSessionsAndInvitesAndLogsOnlyTablesAndCounts(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	seedRetention(t, db.owner)

	ended, logs := runRetention(t, db)

	if ended != river.EventKindJobCompleted {
		t.Fatalf("job %s, want completed; log:\n%s", ended, logs)
	}
	if s, i := count(t, db.owner, "session"), count(t, db.owner, "invite"); s != 1 || i != 1 {
		t.Errorf("%d sessions and %d invites left, want the open one of each", s, i)
	}
	for _, line := range []string{
		`"msg":"retention applied","table":"audit_event","deleted":0,"overdue":0`,
		`"msg":"retention applied","table":"session","deleted":1,"overdue":0`,
		`"msg":"retention applied","table":"invite","deleted":1,"overdue":0`,
		`"msg":"retention applied","table":"slot","deleted":0,"overdue":0`,
		`"msg":"retention applied","table":"video_session","deleted":0,"overdue":0`,
		`"msg":"retention applied","table":"questionnaire_response","deleted":0,"overdue":0`,
		`"msg":"retention applied","table":"binary","deleted":0,"overdue":0`,
	} {
		if !strings.Contains(logs, line) {
			t.Errorf("log lacks %s:\n%s", line, logs)
		}
	}
	assertNoPersonalData(t, logs)
	var audited []string
	rows, err := db.owner.Query(t.Context(), `SELECT entity_name || ' ' || entity_count FROM audit_event
		WHERE agent_name = 'retention_apply' AND action = 'D' ORDER BY entity_name`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var row string
		if err := rows.Scan(&row); err != nil {
			t.Fatal(err)
		}
		audited = append(audited, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"binary 0", "invite 1", "questionnaire_response 0", "session 1", "slot 0", "video_session 0", "webhook_dead_letter 0"}; !slices.Equal(audited, want) {
		t.Errorf("audited %v, want one event per table with its row count", audited)
	}
}

func TestRetentionApply_FailsWithTheCountWhenAnAuditEventIsOverdue(t *testing.T) {
	t.Parallel()
	db := newLogins(t)
	seedRetention(t, db.owner)
	if _, err := db.owner.Exec(t.Context(), `INSERT INTO audit_event
		(recorded, outcome, action, type_system, type_code, subtype_system, subtype_code, purpose_system, purpose_code)
		VALUES (now() - interval '3654 days', 0, 'E', 'http://dicom.nema.org/resources/ontology/DCM', '110114',
		        'http://dicom.nema.org/resources/ontology/DCM', '110122',
		        'http://terminology.hl7.org/CodeSystem/v3-ActReason', 'HOPERAT')`); err != nil {
		t.Fatal(err)
	}

	ended, logs := runRetention(t, db)

	if ended != river.EventKindJobFailed {
		t.Fatalf("job %s, want failed (River retries it); log:\n%s", ended, logs)
	}
	if !strings.Contains(logs, "1 audit_event rows are past their period") {
		t.Errorf("log lacks the overdue count:\n%s", logs)
	}
	if count(t, db.owner, "session") != 1 {
		t.Error("an overdue audit event stopped the purges")
	}
	assertNoPersonalData(t, logs)
}

func assertNoPersonalData(t *testing.T, logs string) {
	t.Helper()
	for _, v := range seeded {
		if regexp.MustCompile(`\b` + regexp.QuoteMeta(v) + `\b`).MatchString(logs) {
			t.Errorf("log holds %q:\n%s", v, logs)
		}
	}
}

func TestNew_RefusesToStartWithoutTheRetentionPool(t *testing.T) {
	_, app := pgtest.NewAppDatabase(t)

	if _, err := worker.New(app, nil, app, liveKit(t), testFiles(t), webhookKey, slog.New(slog.DiscardHandler)); err == nil {
		t.Fatal("New() error = nil without the retention pool")
	}
}

// Only the worker service has the retention and the stats roles' logins in the compose file; the anchors before
// services count as their own section, since every app service merges them.
func TestCompose_OnlyTheWorkerHasTheRetentionAndStatsDatabaseURLs(t *testing.T) {
	for _, name := range []string{"RETENTION_DATABASE_URL", "STATS_DATABASE_URL"} {
		if holders := composeSectionsWith(t, name); !slices.Equal(holders, []string{"worker"}) {
			t.Errorf("%s is set in %v, want only the worker service", name, holders)
		}
	}
}

// composeSectionsWith returns the compose file's services (or "(top level)") whose lines name variable.
func composeSectionsWith(t *testing.T, variable string) []string {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "deployments", "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	service := regexp.MustCompile(`^  ([a-z0-9-]+):`)
	section, inServices := "(top level)", false
	var holders []string
	for sc := bufio.NewScanner(f); sc.Scan(); {
		line, _, _ := strings.Cut(sc.Text(), "#")
		switch m := service.FindStringSubmatch(line); {
		case line == "services:":
			inServices = true
		case !strings.HasPrefix(line, " ") && line != "":
			section, inServices = "(top level)", false
		case inServices && m != nil:
			section = m[1]
		}
		if strings.Contains(line, variable) && !slices.Contains(holders, section) {
			holders = append(holders, section)
		}
	}
	return holders
}

// The retention pool reaches one place: New hands it to newRetention, which builds the retention purgers'
// repositories on it. Every other job, and River itself, runs on the salus_app pool.
func TestNew_HandsTheRetentionPoolToTheRetentionPurgersOnly(t *testing.T) {
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var uses []string
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, arg := range call.Args {
				if id, ok := arg.(*ast.Ident); ok && id.Name == "retentionPool" {
					uses = append(uses, callee(call.Fun))
				}
			}
			return true
		})
	}
	slices.Sort(uses)
	if want := []string{"auth.NewPostgresRepository", "binary.NewPostgresRepository", "newRetention", "notification.NewPostgresRepository",
		"questionnaireresponse.NewPostgresRepository", "slot.NewPostgresRepository", "videosession.NewPostgresRepository"}; !slices.Equal(uses, want) {
		t.Errorf("retentionPool is passed to %v, want only %v", uses, want)
	}
}

// callee names a called function: f or pkg.F.
func callee(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		if x, ok := f.X.(*ast.Ident); ok {
			return x.Name + "." + f.Sel.Name
		}
	}
	return "?"
}
