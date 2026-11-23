package api_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/signup"
	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
	"github.com/Shmeeshmee/salus/pkg/postgres/pgtest"
)

type noInviteJobs struct{}

func (noInviteJobs) EnqueueInviteEmail(context.Context, string, string) error { return nil }

// signupJobs keeps the id of the last signup whose verification e-mail was enqueued.
type signupJobs struct{ last string }

func (j *signupJobs) EnqueueEmail(_ context.Context, id string) error { j.last = id; return nil }

// noLimits counts nothing: every request is the first of its window.
type noLimits struct{}

func (noLimits) Incr(context.Context, string, time.Duration) (int64, time.Duration, error) {
	return 1, time.Hour, nil
}

var signupNow = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)

// newSignupService builds the signup service on the real services as salus_app (app), with the default plan set.
func newSignupService(t *testing.T, owner, app *pgxpool.Pool, jobs *signupJobs) *signup.Service {
	t.Helper()
	if _, err := owner.Exec(t.Context(), `INSERT INTO subscription_plan (code, name, price_per_seat_minor) VALUES ($1, 'Synthetic plan', 1000)`, subscription.DefaultPlan); err != nil {
		t.Fatal(err)
	}
	tx := postgres.NewTransactor(app)
	c := clock.Fixed{T: signupNow}
	audit := auditevent.NewService(auditevent.NewPostgresRepository(app), tx, c)
	return signup.NewService(signup.Ports{
		Repo: signup.NewPostgresRepository(app), Jobs: jobs, Counters: noLimits{}, Tx: tx, Clock: c,
		Clinics:      organization.NewService(organization.NewPostgresRepository(app), audit, tx, c, nil),
		Admins:       auth.NewService(auth.Ports{Repo: auth.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: c, Jobs: noInviteJobs{}}),
		Trials:       subscription.NewService(subscription.Ports{Repo: subscription.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: c}),
		RateLimitKey: []byte("synthetic key"),
	})
}

// A verified signup creates its clinic through the real services as salus_app, as salusctl create-clinic does as the
// owner: the clinic unlisted, its admin's invite and its trial, each in the new clinic's scope (00066).
func TestCreateClinic_AsTheAppThroughTheRealServices(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	svc := newSignupService(t, owner, app, &signupJobs{})

	got, err := svc.CreateClinic(t.Context(), "Synthetic clinic", "admin@example.test")

	if err != nil || got.OrganizationID == "" || got.InviteID == "" || !got.Trial {
		t.Fatalf("CreateClinic() as %s = %+v, %v; want the clinic, its invite and its trial", pgtest.AppRole, got, err)
	}
	var listed bool
	var invites, trials int
	err = owner.QueryRow(t.Context(), `SELECT o.listed,
		(SELECT count(*) FROM invite WHERE organization_id = o.id AND role = 'admin'),
		(SELECT count(*) FROM clinic_subscription WHERE organization_id = o.id)
		FROM organization o WHERE o.id = $1`, got.OrganizationID).Scan(&listed, &invites, &trials)
	if err != nil || listed || invites != 1 || trials != 1 {
		t.Errorf("clinic listed %v, admin invites %d, subscriptions %d, %v; want unlisted, one, one", listed, invites, trials, err)
	}
}

// The audit events of a new clinic name who created it: the signup for a verified signup, salusctl for create-clinic.
func TestCreateClinic_AuditsTheSignupOrTheOperatorAsTheAgent(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	jobs := &signupJobs{}
	svc := newSignupService(t, owner, app, jobs)
	if err := svc.Start(t.Context(), signup.Request{Email: "admin@example.test", ClinicName: "Synthetic clinic", Language: "bs"}, "192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	v, err := svc.IssueVerification(t.Context(), jobs.last)
	if err != nil {
		t.Fatal(err)
	}
	signedUp, err := svc.Verify(t.Context(), v.Token, "192.0.2.1")
	if err != nil {
		t.Fatal(err)
	}
	made, err := svc.CreateClinic(t.Context(), "Synthetic clinic B", "admin-b@example.test")
	if err != nil {
		t.Fatal(err)
	}

	for org, want := range map[string]string{signedUp.OrganizationID: auditevent.AgentSignup, made.OrganizationID: auditevent.AgentOperator} {
		rows, err := owner.Query(t.Context(), `SELECT coalesce(agent_name, '') FROM audit_event WHERE organization_id = $1`, org)
		if err != nil {
			t.Fatal(err)
		}
		agents, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			t.Fatal(err)
		}
		// The clinic, its admin's invite and its trial.
		if len(agents) != 3 || slices.ContainsFunc(agents, func(a string) bool { return a != want }) {
			t.Errorf("audit agents of clinic %s = %q, want three of %q", org, agents, want)
		}
	}
}
