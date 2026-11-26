// Command salusctl is the operator command (docs/accounts/SPEC.md "Operator commands"). It connects as the
// schema owner (DATABASE_URL), like migrate; every change it makes is audited with salusctl as the agent.
//
//	salusctl create-clinic -name <name> -admin-email <e-mail>   the clinic and its first admin's invite
//	salusctl reset-mfa <account-id>                             a staff member who lost their device
//	salusctl term-reviewer grant|revoke <account-id>             the term list's reviewer permission of a staff account
//	salusctl terminology import <seed.jsonl | terms.csv>        the term list's seed or the curator's edits
//	salusctl terminology import-explanations <explanations.jsonl>   the drafted plain-language explanations, as unreviewed
//	salusctl terminology import-flags <translation-flags.jsonl>   wrong labels found, marked as needing changes
//	salusctl terminology export -o <terms.csv> [-status unreviewed|approved|needs_changes]   the term list for the curator
//	salusctl document-templates seed -clinic <id> <template.json>...      the dev seed of placeholder templates
//	salusctl conditions unconfirmed                               the backfilled diagnoses nobody confirmed (SAL-19)
//	salusctl trends build <2026-10 | 2026-Q3 | 2026-W44>          a missed build of the platform figures (SAL-21)
//	salusctl subscription plan|start-trial|mark-paid|status ...   the clinics' membership (subscription.go)
//	salusctl signup list|approve <organization-id>                the clinics that signed themselves up (signup.go)
//	salusctl datarequest list|show|answer|extend|reopen-account ... the data requests to the account (datarequest.go)
//	salusctl demo-seed [-api <url>]                               the demo clinic on a fresh development stack (demo.go)
//
// It prints opaque ids only, never an e-mail, but for demo-seed, which prints the sign-in sheet of its synthetic
// accounts (their made-up e-mails, the shared demo password and, on the first run, their TOTP secrets) on a
// development stack only (demo.go). The terminology commands are not audited: the terms are reference
// data with no personal data (docs/terminology/SPEC.md §4).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/condition"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/datarequest"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/signup"
	"github.com/Shmeeshmee/salus/internal/subscription"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/internal/trends"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

const usage = "usage: salusctl create-clinic -name <name> -admin-email <e-mail> | reset-mfa <account-id> | term-reviewer grant|revoke <account-id> | terminology import|export | document-templates seed | conditions unconfirmed | trends build <period> | subscription plan|start-trial|mark-paid|status | signup list|approve | datarequest list|show|answer|extend|reopen-account | demo-seed"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:], os.Stdout)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "salusctl:", err)
		os.Exit(1)
	}
}

// services are what the commands call, on the owner's connection.
type services struct {
	tx         *postgres.Transactor
	clinics    *organization.Service
	accounts   *auth.Service
	terms      *terminology.Curator
	templates  *composition.Service
	conditions *condition.Service
	trends     *trends.Builder // nil while ANALYTICS_PLATFORM_ENABLED is off
	members    *subscription.Service
	signups    *signup.Service // create-clinic's one path, and the signups' approval
	operator   *datarequest.Operator
	pool       *pgxpool.Pool         // the owner's, for demo-seed (demo.go)
	jobs       *river.Client[pgx.Tx] // the job inserter, for demo-seed
	devCluster string                // the cluster_name demo-seed requires (runOn)
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	return runOn(ctx, args, stdout, demoCluster)
}

// runOn is run, with devCluster the cluster_name demo-seed requires of the database server: demoCluster from run;
// the tests give their own server's, so that they run on a stack whose Postgres predates the setting.
func runOn(ctx context.Context, args []string, stdout io.Writer, devCluster string) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	jobs, err := worker.NewInserter(pool)
	if err != nil {
		return err
	}
	tx := postgres.NewTransactor(pool)
	audit := auditevent.NewService(auditevent.NewPostgresRepository(pool), tx, clock.System{})
	s := services{
		tx: tx, pool: pool, jobs: jobs, devCluster: devCluster,
		clinics: organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clock.System{}, nil),
		accounts: auth.NewService(auth.Ports{
			Repo: auth.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}, Jobs: auth.NewRiverJobs(jobs),
		}),
		terms: terminology.NewCurator(terminology.NewPostgresCurationRepository(pool), tx, clock.System{}),
		// Only SeedTemplate is asked of the documents: it needs the repository, the audit and the clock.
		templates:  composition.NewService(composition.Ports{Repo: composition.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}}),
		conditions: condition.NewService(condition.NewPostgresRepository(pool)),
		// The owner's operations need no clinic's time zone and no seats: the worker issues the invoices.
		members: subscription.NewService(subscription.Ports{Repo: subscription.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}}),
	}
	s.signups = signup.NewService(signup.Ports{
		Repo: signup.NewPostgresRepository(pool), Tx: tx, Clock: clock.System{}, Clinics: s.clinics, Admins: s.accounts, Trials: s.members,
	})
	// The operator's requests need the repository, the audit, the transaction and the clock (the operator's time zone,
	// no clinic's); reopening an account the closure's scope and the accounts.
	requests := datarequest.NewPostgresRepository(pool)
	s.operator = datarequest.NewOperator(
		datarequest.NewService(datarequest.Ports{Repo: requests, Clock: clock.System{}, Tx: tx, Audit: audit}),
		datarequest.NewClosure(datarequest.ClosurePorts{Tx: tx, Repo: requests, Scope: datarequest.NewErasureScope(), Accounts: s.accounts, Clock: clock.System{}}),
	)
	if cfg.Analytics.PlatformEnabled {
		s.trends = worker.NewTrendsBuilder(pool, pool, cfg.Analytics.MinCount, auditevent.AgentOperator)
	}
	switch args[0] {
	case "create-clinic":
		return s.createClinic(ctx, args[1:], stdout)
	case "terminology":
		return s.terminologyCommand(ctx, args[1:], stdout)
	case "document-templates":
		return s.templatesCommand(ctx, args[1:], stdout)
	case "conditions":
		return s.conditionsCommand(ctx, args[1:], stdout)
	case "trends":
		return s.trendsCommand(ctx, args[1:], stdout)
	case "subscription":
		return s.subscriptionCommand(ctx, args[1:], stdout)
	case "signup":
		return s.signupCommand(ctx, args[1:], stdout)
	case "datarequest":
		return s.dataRequestCommand(ctx, args[1:], stdout)
	case "demo-seed":
		return s.demoSeedCommand(ctx, args[1:], stdout)
	case "term-reviewer":
		return s.termReviewer(ctx, args[1:], stdout)
	case "reset-mfa":
		if len(args) != 2 {
			return errors.New(usage)
		}
		if err := s.accounts.ResetMFA(ctx, args[1]); err != nil {
			return err
		}
		_, err := fmt.Fprintln(stdout, "reset MFA of account", args[1]+"; its sessions are revoked")
		return err
	default:
		return fmt.Errorf("unknown command %q; %s", args[0], usage)
	}
}

// createClinic creates the clinic, its first admin's invite and its trial in one transaction, through the one path a
// verified signup takes too (signup.Service.CreateClinic); the notifier sends the invite e-mail.
func (s services) createClinic(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("create-clinic", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	name := flags.String("name", "", "the clinic's name")
	email := flags.String("admin-email", "", "the first admin's e-mail")
	if err := flags.Parse(args); err != nil || *name == "" || *email == "" {
		return errors.New(usage)
	}
	created, err := s.signups.CreateClinic(ctx, *name, *email)
	if err != nil {
		return err
	}
	trial := "started"
	if !created.Trial {
		trial = "none: set the plan " + subscription.DefaultPlan + ", then run subscription start-trial"
	}
	_, err = fmt.Fprintf(stdout, "organization %s\ninvite %s\ntrial %s\n", created.OrganizationID, created.InviteID, trial)
	return err
}

// termReviewer grants or revokes the reviewer permission of the term list on a staff account (docs/plans/v1-finish.md
// "Decisions": the operator's only, never a clinic admin's, as the term list is every clinic's).
func (s services) termReviewer(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) != 2 || (args[0] != "grant" && args[0] != "revoke") {
		return errors.New(usage)
	}
	grant := args[0] == "grant"
	if err := s.accounts.SetTermReviewer(ctx, args[1], grant); err != nil {
		return err
	}
	msg := "account " + args[1] + " is a term reviewer"
	if !grant {
		msg = "account " + args[1] + " is no longer a term reviewer"
	}
	_, err := fmt.Fprintln(stdout, msg)
	return err
}
