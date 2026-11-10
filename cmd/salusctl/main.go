// Command salusctl is the operator command (docs/accounts/SPEC.md "Operator commands"). It connects as the
// schema owner (DATABASE_URL), like migrate; every change it makes is audited with salusctl as the agent.
//
//	salusctl create-clinic -name <name> -admin-email <e-mail>   the clinic and its first admin's invite
//	salusctl reset-mfa <account-id>                             a staff member who lost their device
//	salusctl term-reviewer grant|revoke <account-id>             the term list's reviewer permission of a staff account
//	salusctl terminology import <seed.jsonl | terms.csv>        the term list's seed or the curator's edits
//	salusctl terminology import-explanations <explanations.jsonl>   the drafted plain-language explanations, as unreviewed
//	salusctl terminology export -o <terms.csv> [-status unreviewed|approved|needs_changes]   the term list for the curator
//	salusctl document-templates seed -clinic <id> <template.json>...      the dev seed of placeholder templates
//
// It prints opaque ids only, never an e-mail. The terminology commands are not audited: the terms are reference
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

	"github.com/Shmeeshmee/salus/internal/auditevent"
	"github.com/Shmeeshmee/salus/internal/auth"
	"github.com/Shmeeshmee/salus/internal/composition"
	"github.com/Shmeeshmee/salus/internal/config"
	"github.com/Shmeeshmee/salus/internal/organization"
	"github.com/Shmeeshmee/salus/internal/terminology"
	"github.com/Shmeeshmee/salus/internal/worker"
	"github.com/Shmeeshmee/salus/pkg/clock"
	"github.com/Shmeeshmee/salus/pkg/postgres"
)

const usage = "usage: salusctl create-clinic -name <name> -admin-email <e-mail> | reset-mfa <account-id> | term-reviewer grant|revoke <account-id> | terminology import|export | document-templates seed"

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
	tx        *postgres.Transactor
	clinics   *organization.Service
	accounts  *auth.Service
	terms     *terminology.Curator
	templates *composition.Service
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
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
		tx:      tx,
		clinics: organization.NewService(organization.NewPostgresRepository(pool), audit, tx, clock.System{}),
		accounts: auth.NewService(auth.Ports{
			Repo: auth.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}, Jobs: auth.NewRiverJobs(jobs),
		}),
		terms: terminology.NewCurator(terminology.NewPostgresCurationRepository(pool), tx, clock.System{}),
		// Only SeedTemplate is asked of the documents: it needs the repository, the audit and the clock.
		templates: composition.NewService(composition.Ports{Repo: composition.NewPostgresRepository(pool), Audit: audit, Tx: tx, Clock: clock.System{}}),
	}
	switch args[0] {
	case "create-clinic":
		return s.createClinic(ctx, args[1:], stdout)
	case "terminology":
		return s.terminologyCommand(ctx, args[1:], stdout)
	case "document-templates":
		return s.templatesCommand(ctx, args[1:], stdout)
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

// createClinic creates the clinic and its first admin's invite in one transaction; the notifier sends the
// invite e-mail.
func (s services) createClinic(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("create-clinic", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	name := flags.String("name", "", "the clinic's name")
	email := flags.String("admin-email", "", "the first admin's e-mail")
	if err := flags.Parse(args); err != nil || *name == "" || *email == "" {
		return errors.New(usage)
	}
	var clinic organization.Organization
	var inv auth.Invite
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if clinic, err = s.clinics.Create(ctx, *name); err != nil {
			return err
		}
		inv, err = s.accounts.InviteAdmin(ctx, clinic.ID, *email)
		return err
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "organization %s\ninvite %s\n", clinic.ID, inv.ID)
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
