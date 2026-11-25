package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/practitionerrole"
	"github.com/Shmeeshmee/salus/internal/schedule"
	"github.com/Shmeeshmee/salus/internal/signup"
	"github.com/Shmeeshmee/salus/internal/slot"
	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/pkg/clock"
)

const demoUsage = "usage: salusctl demo-seed [-api <the api's base URL, default http://api:8080>]"

// errDemoRefused is the seed's refusal of a stack that is not a developer's or not fresh: nothing was written.
var errDemoRefused = errors.New("demo-seed: refused")

// demoSeed fills a fresh development stack with the demo clinic (docs/demo/SCRIPT.md, docs/clinic-cms/WAVE7.md RD5).
type demoSeed struct {
	signups  *signup.Service
	accounts *auth.Service
	members  *subscription.Service // with the clinic's time zone and seats, as the monthly job builds it
	slots    *slot.Service
	db       *pgxpool.Pool
	http     *http.Client
	loc      *time.Location
	base     string
	org      string
	admin    demoStaffUser
	desk     demoStaffUser
	// what the seed made, in order
	practitioners []demoStaffUser
	locations     []demoPlace
	services      map[string]string // service code to id
	people        []demoPatient
	month         []demoVisit
	followUp      demoVisit // the visit with the lab result, which the chat is about
}

// demoSeedCommand runs demo-seed. It refuses, writing nothing:
//   - outside a developer's stack: SALUS_ENV must be dev (config.IsDev), which deployments/docker-compose.yml's
//     services and make demo-seed set and production's env file does not;
//   - on any database but a fresh one: one with a clinic in it that is not the demo clinic, which every production or
//     staging database has from its first clinic on; and one the seed stopped on partway.
//
// On a database it has filled already it changes nothing and prints the sign-in sheet again, without the secrets.
func (s services) demoSeedCommand(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("demo-seed", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	base := flags.String("api", "http://api:8080", "the api's base URL")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
		return errors.New(demoUsage)
	}
	if !config.IsDev() {
		return fmt.Errorf("%w: SALUS_ENV is not dev; the demo seed runs only on a developer's compose stack", errDemoRefused)
	}
	clinics, demo, complete, err := demoState(ctx, s.pool)
	switch {
	case err != nil:
		return err
	case clinics == 1 && demo != "" && complete:
		_, err := fmt.Fprintln(stdout, "The demo clinic is there already: nothing changed.")
		if err == nil {
			err = printSheet(stdout, demoSheet(), false)
		}
		return err
	case demo != "" && clinics == 1:
		return fmt.Errorf("%w: an earlier run stopped partway; empty the stack's database (make down, then remove its postgres volume) and run again", errDemoRefused)
	case demo != "":
		return fmt.Errorf("%w: the database holds %d clinics besides the demo's; the demo seed fills only a fresh stack", errDemoRefused, clinics-1)
	case clinics > 0:
		return fmt.Errorf("%w: the database holds %d clinics; the demo seed fills only a fresh stack", errDemoRefused, clinics)
	}
	d := s.newDemoSeed(*base)
	sheet, err := d.run(ctx)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "The demo clinic is ready: organization %s.\n", d.org); err != nil {
		return err
	}
	return printSheet(stdout, sheet, true)
}

// newDemoSeed builds what the seed calls besides the api, on the owner's connection.
func (s services) newDemoSeed(base string) *demoSeed {
	clk := clock.System{}
	audit := auditevent.NewService(auditevent.NewPostgresRepository(s.pool), s.tx, clk)
	roles := practitionerrole.NewService(practitionerrole.Ports{Repo: practitionerrole.NewPostgresRepository(s.pool), Audit: audit, Tx: s.tx, Clock: clk})
	schedules := schedule.NewService(schedule.Ports{Repo: schedule.NewPostgresRepository(s.pool), Audit: audit, Tx: s.tx, Clock: clk, Roles: roles})
	clinics := organization.NewService(organization.NewPostgresRepository(s.pool), audit, s.tx, clk, nil)
	return &demoSeed{
		signups: s.signups, accounts: s.accounts, db: s.pool, base: base, http: &http.Client{Timeout: 30 * time.Second},
		members: subscription.NewService(subscription.Ports{
			Repo: subscription.NewPostgresRepository(s.pool), Audit: audit, Tx: s.tx, Clock: clk, Clinics: clinics, Seats: roles,
			Jobs: subscription.NewRiverJobs(s.jobs),
		}),
		slots: slot.NewService(slot.NewPostgresRepository(s.pool), s.tx, clk, schedules, clinics),
	}
}

// user is a new, signed-out user of the api.
func (d *demoSeed) user() *demoUser { return &demoUser{http: d.http, base: d.base} }

// run seeds the clinic step by step; the membership invoice comes last and marks the seed complete.
func (d *demoSeed) run(ctx context.Context) ([]demoSignIn, error) {
	loc, err := time.LoadLocation("Europe/Sarajevo") // the clinic's zone, organization's default
	if err != nil {
		return nil, err
	}
	d.loc = loc
	for _, step := range []func(context.Context) error{
		d.clinic, d.staff, d.hours, d.settings, d.patients, d.visits, d.receipts, d.labResult, d.quotes, d.carePlan,
		d.questionnaire, d.chat, d.dataRequests, d.membership,
	} {
		if err := step(ctx); err != nil {
			return nil, err
		}
	}
	sheet := []demoSignIn{d.admin.signIn, d.desk.signIn}
	for _, p := range d.practitioners {
		sheet = append(sheet, p.signIn)
	}
	return sheet, nil
}

// settings are the admin's: the chat's hours and the setup wizard marked done.
func (d *demoSeed) settings(ctx context.Context) error {
	var hours []map[string]string
	for _, day := range []string{"mon", "tue", "wed", "thu", "fri"} {
		hours = append(hours, map[string]string{"day": day, "start": "08:00", "end": "16:00"})
	}
	if _, err := d.admin.user.must(ctx, http.StatusOK, http.MethodPut, "/clinic/chat-settings",
		map[string]any{"hours": hours, "response_time_hours": 24, "email_after_minutes": 60}, nil); err != nil {
		return err
	}
	_, err := d.admin.user.must(ctx, http.StatusOK, http.MethodPut, "/clinic/setup", map[string]string{
		"details": "done", "billing": "done", "hours": "done", "practitioner": "done", "services": "done",
	}, nil)
	return err
}

// membership ends the clinic's trial before this month (demoEndTrial) and issues this month's invoice as the monthly
// job does: the clinic is a member with its first invoice open.
func (d *demoSeed) membership(ctx context.Context) error {
	if err := demoEndTrial(ctx, d.db, d.org); err != nil {
		return err
	}
	_, err := d.members.IssueMonthly(ctx, d.org, clock.System{}.Now())
	return err
}

// demoSheet is the sheet without the secrets: every staff account, in the order the seed made them.
func demoSheet() []demoSignIn {
	sheet := []demoSignIn{{who: "Clinic admin", email: demoAdminEmail}, {who: "Reception", email: demoReceptionEmail}}
	for _, p := range demoPractitioners {
		sheet = append(sheet, demoSignIn{who: "Dr. " + p.given + " " + p.family, email: p.email})
	}
	return sheet
}

// printSheet prints how to sign in: the staff in the web app, with the TOTP secret each was enrolled with when this
// run made them, and the app patients in the patient app. Synthetic accounts of a developer's stack only.
func printSheet(w io.Writer, staff []demoSignIn, secrets bool) error {
	if _, err := fmt.Fprintf(w, "\nPassword of every demo account: %s\n\nStaff (web app; add the TOTP secret to an authenticator app):\n", demoPassword); err != nil {
		return err
	}
	for _, s := range staff {
		secret := "enrolled on the first run (salusctl reset-mfa <account-id> enrols again)"
		if secrets {
			secret = s.totpSecret
		}
		if _, err := fmt.Fprintf(w, "  %s %-36s TOTP %s\n", pad(s.who, 24), s.email, secret); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "\nPatients (patient app, no second factor):"); err != nil {
		return err
	}
	for n := range demoAppPatients {
		given, family, _, _ := demoPerson(n)
		if _, err := fmt.Fprintf(w, "  %s %s\n", pad(given+" "+family, 24), demoPatientEmail(n)); err != nil {
			return err
		}
	}
	return nil
}

// pad is s with spaces up to width characters (fmt pads by bytes, and the names have letters of two).
func pad(s string, width int) string {
	if n := utf8.RuneCountInString(s); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}
