package auditevent_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/migrations"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

func TestMain(m *testing.M) { pgtest.Main(m, migrations.FS) }

// stored is an audit_event row as text, NULL as "<null>".
type stored struct {
	recorded, agent, agentName, org, entity, request string
	typ, subtype, purpose, action, outcomeAsText     string
}

func readOnly(t *testing.T, owner *pgxpool.Pool) stored {
	t.Helper()
	var s stored
	err := owner.QueryRow(t.Context(), `SELECT
		to_char(recorded AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS'),
		coalesce(agent_account_id::text, '<null>'), coalesce(agent_name, '<null>'), coalesce(organization_id::text, '<null>'),
		coalesce(entity_reference, '<null>'), coalesce(request_id, '<null>'),
		type_system || '|' || type_code, subtype_system || '|' || subtype_code,
		purpose_system || '|' || purpose_code, action, outcome::text
		FROM audit_event`).Scan(&s.recorded, &s.agent, &s.agentName, &s.org, &s.entity, &s.request,
		&s.typ, &s.subtype, &s.purpose, &s.action, &s.outcomeAsText)
	if err != nil {
		t.Fatalf("read the one audit event: %v", err)
	}
	return s
}

// newAppService records through the Postgres repository logged in as the application role.
func newAppService(t *testing.T) (*auditevent.Service, *postgres.Transactor, *pgxpool.Pool) {
	t.Helper()
	owner, app := pgtest.NewAppDatabase(t)
	tr := postgres.NewTransactor(app)
	return auditevent.NewService(auditevent.NewPostgresRepository(app), tr, clock.Fixed{T: now}), tr, owner
}

func TestInsert_StoresEveryElementAsTheAppRole(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	e := login()
	e.Recorded, e.OrganizationID, e.RequestID = now, "c0000000-0000-4000-8000-00000000000c", "req-1"

	if err := auditevent.NewPostgresRepository(app).Insert(t.Context(), e); err != nil {
		t.Fatalf("Insert() error = %v", err)
	}
	got := readOnly(t, owner)
	want := stored{
		recorded: "2026-11-04T09:30:00",
		agent:    e.AgentAccountID, agentName: "<null>", org: e.OrganizationID, entity: e.EntityReference, request: "req-1",
		typ:     auditevent.SystemDCM + "|" + auditevent.CodeUserAuthentication,
		subtype: auditevent.SystemDCM + "|" + auditevent.CodeLogin,
		purpose: auditevent.SystemActReason + "|" + auditevent.CodeHOPERAT,
		action:  "E", outcomeAsText: "0",
	}
	if got != want {
		t.Errorf("stored %+v,\nwant %+v", got, want)
	}
}

func TestRecord_StoresNullForAbsentAgentClinicEntityAndRequest(t *testing.T) {
	t.Parallel()
	svc, _, owner := newAppService(t)
	e := login() // a failed login with an unknown e-mail: no agent, no entity
	e.AgentAccountID, e.EntityReference, e.Outcome = "", "", auditevent.OutcomeMinorFailure

	if err := svc.Record(t.Context(), e); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	got := readOnly(t, owner)
	if got.agent != "<null>" || got.agentName != "<null>" || got.org != "<null>" || got.entity != "<null>" || got.request != "<null>" {
		t.Errorf("stored %+v, want NULL agents, clinic, entity and request id", got)
	}
}

func TestInsert_RejectsAMalformedIDWithoutEchoingIt(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := auditevent.NewPostgresRepository(app)
	for _, change := range []func(*auditevent.AuditEvent){
		func(e *auditevent.AuditEvent) { e.AgentAccountID = "agent@example.test" },
		func(e *auditevent.AuditEvent) { e.OrganizationID = "agent@example.test" },
	} {
		e := login()
		change(&e)
		err := repo.Insert(t.Context(), e)
		if err == nil || strings.Contains(err.Error(), "example.test") {
			t.Errorf("Insert() error = %v, want an error that does not echo the value", err)
		}
	}
}

func TestRecord_RollsBackWithTheCallersTransaction(t *testing.T) {
	t.Parallel()
	svc, tr, owner := newAppService(t)
	errAccessFailed := errors.New("the access failed")

	err := tr.WithTx(t.Context(), func(ctx context.Context) error {
		if err := svc.Record(ctx, login()); err != nil {
			return err
		}
		return errAccessFailed
	})
	if !errors.Is(err, errAccessFailed) {
		t.Fatalf("WithTx() error = %v", err)
	}
	var n int
	if err := owner.QueryRow(t.Context(), "SELECT count(*) FROM audit_event").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d audit events after the rollback, want 0: Record must join the caller's transaction", n)
	}
}

func TestRecord_StoresAnAgentThatIsNoAccountByName(t *testing.T) {
	t.Parallel()
	svc, _, owner := newAppService(t)
	e := login()
	e.AgentAccountID, e.AgentName = "", auditevent.AgentOperator

	if err := svc.Record(t.Context(), e); err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if got := readOnly(t, owner); got.agent != "<null>" || got.agentName != "salusctl" {
		t.Errorf("stored agent %q named %q, want no account and the operator command, salusctl", got.agent, got.agentName)
	}
}

func TestCountRecordedBefore_CountsTheEventsRecordedBeforeTheCutOffAsTheAppRole(t *testing.T) {
	t.Parallel()
	_, app := pgtest.NewAppDatabase(t)
	repo := auditevent.NewPostgresRepository(app)
	for _, at := range []time.Time{now.Add(-2 * time.Second), now.Add(-time.Second), now, now.Add(time.Second)} {
		e := login()
		e.Recorded = at
		if err := repo.Insert(t.Context(), e); err != nil {
			t.Fatal(err)
		}
	}

	n, err := repo.CountRecordedBefore(t.Context(), now)

	if err != nil || n != 2 {
		t.Errorf("CountRecordedBefore() = %d, %v; want the 2 events recorded before the cut-off", n, err)
	}
}
