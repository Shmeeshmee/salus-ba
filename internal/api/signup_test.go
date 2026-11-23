package api_test

import (
	"context"
	"testing"
	"time"

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

var signupNow = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)

// A verified signup creates its clinic through the real services as salus_app, as salusctl create-clinic does as the
// owner: the clinic unlisted, its admin's invite and its trial, each in the new clinic's scope (00068).
func TestCreateClinic_AsTheAppThroughTheRealServices(t *testing.T) {
	t.Parallel()
	owner, app := pgtest.NewAppDatabase(t)
	if _, err := owner.Exec(t.Context(), `INSERT INTO subscription_plan (code, name, price_per_seat_minor) VALUES ($1, 'Synthetic plan', 1000)`, subscription.DefaultPlan); err != nil {
		t.Fatal(err)
	}
	tx := postgres.NewTransactor(app)
	c := clock.Fixed{T: signupNow}
	audit := auditevent.NewService(auditevent.NewPostgresRepository(app), tx, c)
	svc := signup.NewService(signup.Ports{
		Repo: signup.NewPostgresRepository(app), Tx: tx, Clock: c,
		Clinics: organization.NewService(organization.NewPostgresRepository(app), audit, tx, c, nil),
		Admins:  auth.NewService(auth.Ports{Repo: auth.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: c, Jobs: noInviteJobs{}}),
		Trials:  subscription.NewService(subscription.Ports{Repo: subscription.NewPostgresRepository(app), Audit: audit, Tx: tx, Clock: c}),
	})

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
